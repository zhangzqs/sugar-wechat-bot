from wxauto import WeChat
from wxauto.msgs import FriendMessage, BaseMessage, SystemMessage, TickleMessage,TimeMessage,SelfMessage
import time
from pydantic import BaseModel
import nats
from nats.aio.client import Client as NATS
from nats.js import JetStreamContext
from logger import LoggerConfig, init_logger
from config import load_config_from_args
import logging
import asyncio
import json

class ChatTransferConfig(BaseModel):
    chat: str
    """微信会话名称"""
    
    subject: str
    """NATS主题名称"""
    
    def __str__(self):
        return f"{self.chat} -> {self.subject}"
    

class Config(BaseModel):
    logger: LoggerConfig
    """日志配置"""
    
    nats_url: str = "nats://localhost:4222"
    """NATS服务器"""
    
    chat_transfer_config: list[ChatTransferConfig] = []
    """某个微信会话中的消息转发到NATS的主题的映射"""

# 微信消息处理函数
async def on_wxchat_message(
    wxmsg: BaseMessage, # 微信消息对象
    wxchat: str, # 微信会话名称
    subject: str, # 期望转发到主题
    js: JetStreamContext,
):
    print(f'收到消息：[{wxmsg.type} {wxmsg.attr}]{wxchat} - {wxmsg.content}')
    if isinstance(wxmsg, FriendMessage):
        logging.info(f"好友消息: {wxmsg.content} from {wxmsg.sender} ({wxmsg.sender_remark})")
        
        await js.publish(
            subject=subject,
            payload=json.dumps({
                'type': wxmsg.type, # 消息内容类型
                'attr': 'friend', # 消息来源类型
                'id': wxmsg.id, # 消息ID
                'content': wxmsg.content, # 消息内容
                'sender': wxmsg.sender, # 消息发送者
                'sender_remark': wxmsg.sender_remark, # 消息发送者备注
                'info': wxmsg.chat_info(), # 消息详情
            }).encode('utf-8')
        )
    elif isinstance(wxmsg, SystemMessage):
        logging.info(f"系统消息: {wxmsg.content}")
    elif isinstance(wxmsg, TickleMessage):
        logging.info(f"收到戳一戳消息: {wxmsg.tickle_list}")
    elif isinstance(wxmsg, TimeMessage):
        logging.info(f"收到时间消息: {wxmsg.time}")
    elif isinstance(wxmsg, SelfMessage):
        logging.info(f"收到自己的消息: {wxmsg.content}")
    else:
        logging.info(f"收到未知类型消息: {wxmsg.content}")


async def main():
    cfg = load_config_from_args(Config)
    init_logger(cfg.logger, logging.getLogger())
    logging.info("WeChat Agent Started")
    nc: NATS = await nats.connect(cfg.nats_url)
    js: JetStreamContext = nc.jetstream()
    logging.info(f"Connected to NATS at {cfg.nats_url}")
    
    loop = asyncio.get_event_loop()
    wx = WeChat()
    for item in cfg.chat_transfer_config:
        chat, topic = item.chat, item.subject
    
        wx.AddListenChat(
            nickname=chat, 
            callback=lambda msg, chat: loop.call_soon_threadsafe(asyncio.create_task, 
                on_wxchat_message(msg, chat, topic, js)
            )
        )
        logging.info(f"Listening to chat '{chat}' and publishing to topic '{topic}'")
    logging.info("WeChat Agent is running. Press Ctrl+C to exit.")
    
    ps = await js.pull_subscribe(
        subject="",
        durable="",
    )
    
    try:
        while True:
            msg = await ps.fetch(batch=1, timeout=1)
            for m in msg:
                logging.info(f"Received message from NATS: {m.data.decode('utf-8')}")
                await m.ack()
    except KeyboardInterrupt:
        logging.info("WeChat Agent stopped by user.")
    finally:
        await nc.close()
        wx.Close()
        logging.info("NATS connection closed and WeChat window closed.")

if __name__ == "__main__":
    asyncio.run(main())

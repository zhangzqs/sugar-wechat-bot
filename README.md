# sugar-wechat-bot

基于 MCP 的糖糖 Bot 微信机器人

## 消息队列

使用 NATS JetStream 作为消息队列，使用 wxauto 将微信消息转发到 NATS Stream 中。

```bash
# 创建 NATS Stream 流，用于持久化 Bot 消息
nats stream add BOTS_STREAM \
    --subjects "BOTS.*" \
    --ack \
    --max-age=1y \
    --storage file \
    --retention limits \
    --max-msg-size=-1 \
    --discard=old

# 创建消息接收器
nats consumer add BOTS_STREAM WX_MSGS_CONSUMER \
    --filter BOTS.received_msgs \
    --ack explicit \
    --pull \
    --deliver all \
    --max-deliver=-1

# 创建消息发送器
nats consumer add BOTS_STREAM WX_MSGS_SENDER_CONSUMER \
    --filter BOTS.send_msgs \
    --ack explicit \
    --pull \
    --deliver all \
    --max-deliver=-1
```

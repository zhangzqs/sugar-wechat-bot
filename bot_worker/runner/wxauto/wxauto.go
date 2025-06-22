package wxauto

// 消息属性（来源属性）
type MessageAttr string

const (
	MessageAttrSystem MessageAttr = "system" // 系统消息
	MessageAttrTime   MessageAttr = "time"   // 时间消息
	MessageAttrTickle MessageAttr = "tickle" // 拍一拍消息
	MessageAttrSelf   MessageAttr = "self"   // 自己发送的消息
	MessageAttrFriend MessageAttr = "friend" // 好友消息
	MessageAttrOther  MessageAttr = "other"  // 其他消息
)

// 消息类型（内容属性）
type MessageType string

const (
	MessageTypeText         MessageType = "text"          // 文本消息
	MessageTypeQuote        MessageType = "quote"         // 引用消息
	MessageTypeVoice        MessageType = "voice"         // 语音消息
	MessageTypeImage        MessageType = "image"         // 图片消息
	MessageTypeVideo        MessageType = "video"         // 视频消息
	MessageTypeFile         MessageType = "file"          // 文件消息
	MessageTypeLocation     MessageType = "location"      // 位置消息
	MessageTypeLink         MessageType = "link"          // 链接消息
	MessageTypeEmotion      MessageType = "emotion"       // 表情消息
	MessageTypeMerge        MessageType = "merge"         // 合并转发消息
	MessageTypePersonalCard MessageType = "personal_card" // 个人名片消息
	MessageTypeNote         MessageType = "note"          // 笔记消息
	MessageTypeOther        MessageType = "other"         // 其他消息
)

// 会话类型
type ChatType string

const (
	ChatTypeFriend   ChatType = "friend"   // 好友会话
	ChatTypeGroup    ChatType = "group"    // 群聊会话
	ChatTypeSystem   ChatType = "service"  // 客服会话
	ChatTypeOfficial ChatType = "official" // 公众号会话
)

type GroupInfo struct {
	GroupMemberCount int `json:"group_member_count"` // 群成员数量
}

type ChatInfo struct {
	ChatType  string           `json:"chat_type"` // 会话类型
	ChatName  string           `json:"chat_name"` // 会话名称
	GroupInfo `json:",inline"` // 群信息，只有在群聊会话时才有
}

type ReceivedMessage struct {
	ID           string      `json:"id"`            // 消息唯一 ID
	Type         MessageType `json:"type"`          // 消息类型（内容属性），如 text/image/voice 等
	Attr         MessageAttr `json:"attr"`          // 消息属性（来源属性），如 self/friend/system 等
	Content      string      `json:"content"`       // 消息内容
	Sender       string      `json:"sender"`        // 发送者
	SenderRemark string      `json:"sender_remark"` // 发送者备注
}

type SendMessage struct {
	ReplyToMsgID string   `json:"reply_to_msg_id,omitempty"` // 回复的消息 ID，只有在回复消息时才有意义
	SendToChat   string   `json:"send_to_chat"`              // 接收者，可以是好友或群聊名称
	Content      string   `json:"content"`                   // 消息内容
	At           []string `json:"at,omitempty"`              // @ 的人列表，只有在群聊时才有意义e
	Exact        bool     `json:"exact,omitempty"`           // 是否精确匹配接收者名称，默认为 false
}

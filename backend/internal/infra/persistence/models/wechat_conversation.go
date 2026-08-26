package model

// WeChatConversation 微信联系人级会话映射（P0-b 向前映射）。
// 存量 bot 账户级会话（如 1341）保留为 legacy 归档；新消息起按 hub 消息
// payload 的 Sender（wxid）每联系人独立建会话。
// 复合唯一索引 (bot_account_id, wxid, conversation_id) 允许同一联系人
// 在切换后保留多条历史映射记录；查询侧按 updated_at DESC 取最新一条。
type WeChatConversation struct {
	ControlPlaneModel
	BotAccountID   uint   `gorm:"not null;uniqueIndex:idx_wechat_conv_unique,priority:1;comment:bot账号ID(wechat_bot_accounts.id)"`
	Wxid           string `gorm:"size:128;not null;uniqueIndex:idx_wechat_conv_unique,priority:2;comment:联系人Sender(hub消息payload)"`
	ConversationID uint   `gorm:"not null;uniqueIndex:idx_wechat_conv_unique,priority:3;comment:会话ID(chat_conversations.id)"`
}

// TableName 指定表名。
func (WeChatConversation) TableName() string {
	return "wechat_conversations"
}

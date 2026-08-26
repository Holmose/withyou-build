package wechatbot

import "time"

type BotAccountStatus string

const (
	BotStatusOffline BotAccountStatus = "offline"
	BotStatusPairing BotAccountStatus = "pairing"
	BotStatusScanned BotAccountStatus = "scanned"
	BotStatusOnline  BotAccountStatus = "online"
	BotStatusExpired BotAccountStatus = "expired"
	BotStatusError   BotAccountStatus = "error"
)

type BotAccountWithUser struct {
	ID                uint
	UserID            uint
	Username          string
	UserDisplayName   string
	ConversationID    uint
	ConversationPublicID string
	ConversationModel string
	BotModel          string
	BotOpenID         string
	HubBotID          string
	HubChannelKey     string
	HubCursorID       int64
	Status            BotAccountStatus
	Nickname          string
	WeChatUserID      string
	AvatarURL         string
	ExpiresAt         *time.Time
	LastStatusChangeAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type BotAccount struct {
	ID                uint
	UserID            uint
	ConversationID    uint
	BotOpenID         string
	Status            BotAccountStatus
	HubBotID          string
	HubChannelKey     string
	HubCursorID       int64
	QRCodeToken       string
	BotToken          string
	BotBaseURL        string
	BotUIN            string
	LastQRCodeData    string
	Nickname          string
	WeChatUserID      string
	AvatarURL         string
	Model             string
	ExpiresAt         *time.Time
	LastStatusChangeAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ContactConversation 微信联系人与会话的映射（wechat_conversations 表）。
type ContactConversation struct {
	ID             uint
	BotAccountID   uint
	Wxid           string
	ConversationID uint
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

package model

import "time"

type WeChatBotAccount struct {
	ControlPlaneModel
	UserID             uint       `gorm:"uniqueIndex:idx_wechat_bot_accounts_user_id;not null;comment:用户ID"`
	ConversationID     uint       `gorm:"index:idx_wechat_bot_accounts_conversation_id;not null;default:0;comment:归属会话ID"`
	BotOpenID          string     `gorm:"size:64;not null;default:'';comment:机器人微信openid"`
	HubBotID           string     `gorm:"size:64;not null;default:'';comment:openilink-hub 中的 bot ID"`
	HubChannelKey      string     `gorm:"size:128;not null;default:'';comment:openilink-hub 频道 API key"`
	HubCursorID        int64      `gorm:"not null;default:0;comment:已处理的最后一条消息ID"`
	Status             string     `gorm:"size:32;not null;default:'offline';index:idx_wechat_bot_accounts_status;comment:状态(offline/pairing/scanned/online/expired/error)"`
	Model              string     `gorm:"size:128;not null;default:'';comment:bot专属pool-profile模型名(空=用配置默认)"`
	QRCodeToken        string     `gorm:"size:128;not null;default:'';comment:当前二维码token"`
	BotToken           string     `gorm:"type:text;not null;default:'';comment:iLink bot token"`
	BotBaseURL         string     `gorm:"type:text;not null;default:'';comment:iLink base URL"`
	BotUIN             string     `gorm:"size:64;not null;default:'';comment:微信UIN"`
	LastQRCodeData     string     `gorm:"type:text;not null;default:'';comment:最近一次二维码原始数据"`
	Nickname           string     `gorm:"size:64;not null;default:'';comment:微信昵称"`
	WeChatUserID       string     `gorm:"size:128;not null;default:'';comment:微信侧用户ID(ilink_user_id)"`
	AvatarURL          string     `gorm:"type:text;not null;default:'';comment:头像URL"`
	ExpiresAt          *time.Time `gorm:"comment:Token过期时间"`
	LastStatusChangeAt *time.Time `gorm:"comment:最近一次状态变更时间"`
}

func (WeChatBotAccount) TableName() string {
	return "wechat_bot_accounts"
}

package repository

import (
	"context"
	"time"

	domain "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/wechatbot"
)

type WeChatBotRepository interface {
	GetUserBotByUserID(ctx context.Context, userID uint) (*domain.BotAccount, error)
	GetAllOnlineBots(ctx context.Context) ([]*domain.BotAccount, error)
	ListAllBots(ctx context.Context, page, pageSize int) ([]domain.BotAccountWithUser, int64, error)
	GetBotByUserIDWithUser(ctx context.Context, userID uint) (*domain.BotAccountWithUser, error)
	SaveUserBotAccount(ctx context.Context, account *domain.BotAccount) error
	UpdateUserBotToken(ctx context.Context, userID uint, botToken, botBaseURL, qrCodeToken string, expiresAt *time.Time) error
	UpdateUserBotStatus(ctx context.Context, userID uint, status string) error
	UpdateBotCursor(ctx context.Context, userID uint, cursorID int64) error
	UpdateBotChannelKey(ctx context.Context, userID uint, channelKey string) error
	UpdateUserBotNickname(ctx context.Context, id uint, nickname string) error
	UpdateUserBotWeChatUserID(ctx context.Context, id uint, wechatUserID string) error
	UpdateBotModel(ctx context.Context, userID uint, modelName string) error
	GetContactConversation(ctx context.Context, botAccountID uint, wxid string) (*domain.ContactConversation, error)
	UpsertContactConversation(ctx context.Context, cc *domain.ContactConversation) (*domain.ContactConversation, error)
	ListContactConversations(ctx context.Context, botAccountID uint) ([]domain.ContactConversation, error)
	DeleteUserBot(ctx context.Context, userID uint) error
}

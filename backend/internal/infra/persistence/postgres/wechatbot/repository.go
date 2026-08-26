package wechatbot

import (
	"context"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/dberror"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	domain "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/wechatbot"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func toDomain(m *model.WeChatBotAccount) *domain.BotAccount {
	if m == nil {
		return nil
	}
	return &domain.BotAccount{
		ID:                m.ID,
		UserID:            m.UserID,
		ConversationID:    m.ConversationID,
		BotOpenID:         m.BotOpenID,
		HubBotID:          m.HubBotID,
		HubChannelKey:     m.HubChannelKey,
		HubCursorID:       m.HubCursorID,
		Status:            domain.BotAccountStatus(m.Status),
		QRCodeToken:       m.QRCodeToken,
		BotToken:          m.BotToken,
		BotBaseURL:        m.BotBaseURL,
		BotUIN:            m.BotUIN,
		LastQRCodeData:    m.LastQRCodeData,
		Nickname:          m.Nickname,
		WeChatUserID:      m.WeChatUserID,
		AvatarURL:         m.AvatarURL,
		Model:             m.Model,
		ExpiresAt:         m.ExpiresAt,
		LastStatusChangeAt: m.LastStatusChangeAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func toModel(d *domain.BotAccount) *model.WeChatBotAccount {
	if d == nil {
		return nil
	}
	return &model.WeChatBotAccount{
		ControlPlaneModel: model.ControlPlaneModel{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
		UserID:            d.UserID,
		ConversationID:    d.ConversationID,
		BotOpenID:         d.BotOpenID,
		HubBotID:          d.HubBotID,
		HubChannelKey:     d.HubChannelKey,
		HubCursorID:       d.HubCursorID,
		Status:            string(d.Status),
		QRCodeToken:       d.QRCodeToken,
		BotToken:          d.BotToken,
		BotBaseURL:        d.BotBaseURL,
		BotUIN:            d.BotUIN,
		LastQRCodeData:    d.LastQRCodeData,
		Nickname:          d.Nickname,
		WeChatUserID:      d.WeChatUserID,
		AvatarURL:         d.AvatarURL,
		Model:             d.Model,
		ExpiresAt:         d.ExpiresAt,
		LastStatusChangeAt: d.LastStatusChangeAt,
	}
}

func (r *Repository) GetUserBotByUserID(ctx context.Context, userID uint) (*domain.BotAccount, error) {
	var m model.WeChatBotAccount
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if err != nil {
		if dberror.IsRecordNotFound(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toDomain(&m), nil
}

func (r *Repository) GetAllOnlineBots(ctx context.Context) ([]*domain.BotAccount, error) {
	var ms []model.WeChatBotAccount
	err := r.db.WithContext(ctx).Where("status = ?", "online").Find(&ms).Error
	if err != nil {
		return nil, err
	}
	result := make([]*domain.BotAccount, len(ms))
	for i := range ms {
		result[i] = toDomain(&ms[i])
	}
	return result, nil
}

func (r *Repository) SaveUserBotAccount(ctx context.Context, account *domain.BotAccount) error {
	m := toModel(account)
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *Repository) UpdateUserBotToken(ctx context.Context, userID uint, botToken, botBaseURL, qrCodeToken string, expiresAt *time.Time) error {
	return r.db.WithContext(ctx).Model(&model.WeChatBotAccount{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"bot_token":            botToken,
			"bot_base_url":         botBaseURL,
			"qrcode_token":         qrCodeToken,
			"status":               "online",
			"expires_at":           expiresAt,
			"last_status_change_at": time.Now(),
		}).Error
}

func (r *Repository) UpdateUserBotStatus(ctx context.Context, userID uint, status string) error {
	return r.db.WithContext(ctx).Model(&model.WeChatBotAccount{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"status":               status,
			"last_status_change_at": time.Now(),
		}).Error
}

func (r *Repository) ListAllBots(ctx context.Context, page, pageSize int) ([]domain.BotAccountWithUser, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.WeChatBotAccount{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	rows, err := r.db.WithContext(ctx).Table("wechat_bot_accounts w").
		Select(`w.*, u.username, u.display_name as user_display_name`).
		Joins("LEFT JOIN identity_users u ON u.id = w.user_id").
		Order("w.id DESC").
		Offset(offset).
		Limit(pageSize).
		Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	type row struct {
		model.WeChatBotAccount
		Username        string `gorm:"column:username"`
		UserDisplayName string `gorm:"column:user_display_name"`
	}
	var result []domain.BotAccountWithUser
	for rows.Next() {
		var rw row
		if err := r.db.ScanRows(rows, &rw); err != nil {
			return nil, 0, err
		}
		result = append(result, domain.BotAccountWithUser{
			ID:              rw.ID,
			UserID:          rw.UserID,
			Username:        rw.Username,
			UserDisplayName: rw.UserDisplayName,
			ConversationID:  rw.ConversationID,
			BotOpenID:       rw.BotOpenID,
			HubBotID:        rw.HubBotID,
			HubChannelKey:   rw.HubChannelKey,
			HubCursorID:     rw.HubCursorID,
			Status:          domain.BotAccountStatus(rw.Status),
			Nickname:        rw.Nickname,
			WeChatUserID:    rw.WeChatUserID,
			AvatarURL:       rw.AvatarURL,
			BotModel:        rw.Model,
			ExpiresAt:       rw.ExpiresAt,
			LastStatusChangeAt: rw.LastStatusChangeAt,
			CreatedAt:       rw.CreatedAt,
			UpdatedAt:       rw.UpdatedAt,
		})
	}
	return result, total, nil
}

func (r *Repository) GetBotByUserIDWithUser(ctx context.Context, userID uint) (*domain.BotAccountWithUser, error) {
	type row struct {
		model.WeChatBotAccount
		Username        string `gorm:"column:username"`
		UserDisplayName string `gorm:"column:user_display_name"`
	}
	var rw row
	err := r.db.WithContext(ctx).Table("wechat_bot_accounts w").
		Select(`w.*, u.username, u.display_name as user_display_name`).
		Joins("LEFT JOIN identity_users u ON u.id = w.user_id").
		Where("w.user_id = ?", userID).
		Take(&rw).Error
	if err != nil {
		return nil, err
	}
	return &domain.BotAccountWithUser{
		ID:              rw.ID,
		UserID:          rw.UserID,
		Username:        rw.Username,
		UserDisplayName: rw.UserDisplayName,
		ConversationID:  rw.ConversationID,
		BotOpenID:       rw.BotOpenID,
		HubBotID:        rw.HubBotID,
		HubChannelKey:   rw.HubChannelKey,
		HubCursorID:     rw.HubCursorID,
		Status:          domain.BotAccountStatus(rw.Status),
		Nickname:        rw.Nickname,
		WeChatUserID:    rw.WeChatUserID,
		AvatarURL:       rw.AvatarURL,
		BotModel:        rw.Model,
		ExpiresAt:       rw.ExpiresAt,
		LastStatusChangeAt: rw.LastStatusChangeAt,
		CreatedAt:       rw.CreatedAt,
		UpdatedAt:       rw.UpdatedAt,
	}, nil
}

func (r *Repository) UpdateBotCursor(ctx context.Context, userID uint, cursorID int64) error {
	return r.db.WithContext(ctx).Model(&model.WeChatBotAccount{}).
		Where("user_id = ?", userID).
		Update("hub_cursor_id", cursorID).Error
}

func (r *Repository) UpdateBotChannelKey(ctx context.Context, userID uint, channelKey string) error {
	return r.db.WithContext(ctx).Model(&model.WeChatBotAccount{}).
		Where("user_id = ?", userID).
		Update("hub_channel_key", channelKey).Error
}

func (r *Repository) UpdateUserBotNickname(ctx context.Context, id uint, nickname string) error {
	return r.db.WithContext(ctx).Model(&model.WeChatBotAccount{}).
		Where("id = ?", id).
		Update("nickname", nickname).Error
}

func (r *Repository) UpdateUserBotWeChatUserID(ctx context.Context, id uint, wechatUserID string) error {
	return r.db.WithContext(ctx).Model(&model.WeChatBotAccount{}).
		Where("id = ?", id).
		Update("we_chat_user_id", wechatUserID).Error
}

func (r *Repository) DeleteUserBot(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Unscoped().Where("user_id = ?", userID).
		Delete(&model.WeChatBotAccount{}).Error
}

func (r *Repository) UpdateBotModel(ctx context.Context, userID uint, modelName string) error {
	return r.db.WithContext(ctx).Model(&model.WeChatBotAccount{}).
		Where("user_id = ?", userID).
		Update("model", modelName).Error
}

func toContactDomain(m *model.WeChatConversation) *domain.ContactConversation {
	if m == nil {
		return nil
	}
	return &domain.ContactConversation{
		ID:             m.ID,
		BotAccountID:   m.BotAccountID,
		Wxid:           m.Wxid,
		ConversationID: m.ConversationID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func (r *Repository) GetContactConversation(ctx context.Context, botAccountID uint, wxid string) (*domain.ContactConversation, error) {
	var m model.WeChatConversation
	err := r.db.WithContext(ctx).
		Where("bot_account_id = ? AND wxid = ?", botAccountID, wxid).
		Order("updated_at DESC").
		First(&m).Error
	if err != nil {
		if dberror.IsRecordNotFound(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toContactDomain(&m), nil
}

func (r *Repository) UpsertContactConversation(ctx context.Context, cc *domain.ContactConversation) (*domain.ContactConversation, error) {
	if cc == nil {
		return nil, repository.ErrNotFound
	}
	m := model.WeChatConversation{
		BotAccountID:   cc.BotAccountID,
		Wxid:           cc.Wxid,
		ConversationID: cc.ConversationID,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return toContactDomain(&m), nil
}

func (r *Repository) ListContactConversations(ctx context.Context, botAccountID uint) ([]domain.ContactConversation, error) {
	var ms []model.WeChatConversation
	err := r.db.WithContext(ctx).
		Where("bot_account_id = ?", botAccountID).
		Order("updated_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	result := make([]domain.ContactConversation, 0, len(ms))
	for i := range ms {
		result = append(result, *toContactDomain(&ms[i]))
	}
	return result, nil
}

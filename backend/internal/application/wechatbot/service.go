package wechatbot

import (
	"bytes"
	"context"
	"errors"

	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	appchannel "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/channel"
	appaudit "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/audit"
	appconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/conversation"
	domainaudit "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/audit"
	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	domain "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/wechatbot"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"go.uber.org/zap"
)

type userBot struct {
	hubBotID string
	cancel   context.CancelFunc
	account  *domain.BotAccount

	// streamDone 指向当前在途流的完成信号（watcher 串行，每 bot 至多一条在途流）。
	streamMu   sync.Mutex
	streamDone chan struct{}
}

// setStreamDone 注册在途流信号；返回旧值（应为 nil）。
func (ub *userBot) setStreamDone(done chan struct{}) {
	ub.streamMu.Lock()
	defer ub.streamMu.Unlock()
	ub.streamDone = done
}

// clearStreamDone 清理在途流信号（仅当仍指向 done 时）。
func (ub *userBot) clearStreamDone(done chan struct{}) {
	ub.streamMu.Lock()
	defer ub.streamMu.Unlock()
	if ub.streamDone == done {
		ub.streamDone = nil
	}
}

// waitStreamDone 等待在途流结束，最多等待 timeout；返回是否在超时前结束。
func (ub *userBot) waitStreamDone(timeout time.Duration) bool {
	ub.streamMu.Lock()
	done := ub.streamDone
	ub.streamMu.Unlock()
	if done == nil {
		return true
	}
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// ErrAlreadyBound 表示用户已存在在线绑定，禁止直接重新发起绑定，需先解绑。
var ErrAlreadyBound = errors.New("already_bound")

// ErrInvalidChannelKey 表示 hub channel api_key 无效（DB 存错或过期），触发自动刷新。
var ErrInvalidChannelKey = errors.New("invalid_channel_key")

// ErrBotNotFound 表示用户尚未绑定微信机器人（可作为"未开通"判断）。
var ErrBotNotFound = errors.New("bot not found")

type AdminBotDetailResult struct {
	Bot                  domain.BotAccountWithUser
	ConversationPublicID string
	DefaultModel         string
}

type Service struct {
	cfg        *config.Runtime
	repo       repository.WeChatBotRepository
	log        *zap.Logger
	bots       map[uint]*userBot
	mu         sync.Mutex
	convSvc    *appconversation.Service
	convRepo   repository.ConversationRepository
	channelSvc *appchannel.Service
	auditSvc   auditWriter
	auditReader auditReader

	hubHTTP      *http.Client
	hubWS        *websocket.Dialer
	hubBaseURL   string
	hubUser      string
	hubPass      string
	hubCookieJar *cookiejar.Jar
	hubAuthMu    sync.Mutex
	hubCookies   []*http.Cookie

	bindStatus   map[string]*bindStatusEntry
	bindStatusMu sync.Mutex
}

type bindStatusEntry struct {
	status       string
	botID        string
	qrURL        string
	hubSessionID string
	err          error
}

// auditWriter 复用 audit 服务写入审计日志（audit_logs 表）。
type auditWriter interface {
	Write(ctx context.Context, requestID string, actorUserID uint, action string, resource string, resourceID string, ip string, userAgent string, detail interface{})
}

// auditReader 复用 audit 服务查询审计日志。
type auditReader interface {
	List(ctx context.Context, page int, pageSize int, filter appaudit.ListFilter) ([]domainaudit.Log, int64, error)
}

// SetAuditWriter 注入微信渠道审计写入器。
func (s *Service) SetAuditWriter(writer auditWriter) {
	s.auditSvc = writer
}

// SetAuditReader 注入微信渠道审计读取器。
func (s *Service) SetAuditReader(reader auditReader) {
	s.auditReader = reader
}

func NewService(cfg *config.Runtime, repo repository.WeChatBotRepository, log *zap.Logger) *Service {
	jar, _ := cookiejar.New(nil)
	s := &Service{
		cfg:          cfg,
		repo:         repo,
		log:          log,
		bots:         make(map[uint]*userBot),
		hubHTTP:      &http.Client{Jar: jar, Timeout: 30 * time.Second},
		hubWS:        &websocket.Dialer{HandshakeTimeout: 10 * time.Second},
		hubBaseURL:   cfg.Snapshot().WeChatBotHubURL,
		hubUser:      cfg.Snapshot().WeChatBotHubUsername,
		hubPass:      cfg.Snapshot().WeChatBotHubPassword,
		hubCookieJar: jar,
		bindStatus:   make(map[string]*bindStatusEntry),
	}
	// P0-b 防静默带病上线：联系人级会话未开启时，启动即告警跨联系人上下文污染风险。
	if cfg.Snapshot().WeChatBotEnabled && !s.perContactSessionsEnabled() {
		log.Warn("cross-contact pollution active",
			zap.String("hint", "wechat_bot.per_contact_sessions=false; all WeChat contacts share one conversation context"),
		)
	}
	return s
}

func (s *Service) SetConversationService(svc *appconversation.Service) {
	s.convSvc = svc
}

func (s *Service) SetChannelService(svc *appchannel.Service) {
	s.channelSvc = svc
}

// CanUseWeChatBot 判断用户是否有权限使用微信机器人（基于权限组配置，实时判断）。
func (s *Service) CanUseWeChatBot(ctx context.Context, userID uint) (bool, error) {
	if s.channelSvc == nil {
		return false, nil
	}
	return s.channelSvc.CanUseWeChatBot(ctx, userID)
}

func (s *Service) ensureHubAuth(ctx context.Context) error {
	s.hubAuthMu.Lock()
	defer s.hubAuthMu.Unlock()

	body, _ := json.Marshal(map[string]string{
		"username": s.hubUser,
		"password": s.hubPass,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", s.hubBaseURL+"/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.hubHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("hub login: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return fmt.Errorf("hub not available at %s", s.hubBaseURL)
	}
	if resp.StatusCode == 401 || resp.StatusCode == 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hub login failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hub login returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *Service) hubLogin(ctx context.Context) error {
	// Try login first, if 404 the user doesn't exist — register
	body, _ := json.Marshal(map[string]string{
		"username": s.hubUser,
		"password": s.hubPass,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", s.hubBaseURL+"/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.hubHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("hub login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 || resp.StatusCode == 400 {
		// User doesn't exist or other error — try register
		body, _ := json.Marshal(map[string]string{
			"username": s.hubUser,
			"password": s.hubPass,
			"name":     "DEEIX Bot",
		})
		req2, _ := http.NewRequestWithContext(ctx, "POST", s.hubBaseURL+"/api/auth/register", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		resp2, err := s.hubHTTP.Do(req2)
		if err != nil {
			return fmt.Errorf("hub register: %w", err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != 200 && resp2.StatusCode != 201 {
			b, _ := io.ReadAll(resp2.Body)
			return fmt.Errorf("hub register failed (HTTP %d): %s", resp2.StatusCode, string(b))
		}
		s.log.Info("hub_user_registered", zap.String("username", s.hubUser))
		// Now login
		return s.ensureHubAuth(ctx)
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hub login failed (HTTP %d): %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *Service) StartBot(ctx context.Context) error {
	if !s.cfg.Snapshot().WeChatBotEnabled {
		return nil
	}
	if err := s.hubLogin(ctx); err != nil {
		return fmt.Errorf("wechat_bot hub login: %w", err)
	}
	s.log.Info("hub_connected")

	accounts, err := s.repo.GetAllOnlineBots(ctx)
	if err != nil {
		return err
	}
	for _, a := range accounts {
		s.startMsgWatcher(ctx, a)
	}

	// 启动后台定期巡检：DB 与 Hub 状态漂移检测
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.reconcileHubState(context.Background())
			}
		}
	}()
	return nil
}

func (s *Service) StopBot() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ub := range s.bots {
		if ub.cancel != nil {
			ub.cancel()
		}
	}
	s.bots = make(map[uint]*userBot)
}

// stopUserBot 停止并移除指定用户的机器人消息监听。
func (s *Service) stopUserBot(userID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ub, ok := s.bots[userID]; ok {
		if ub.cancel != nil {
			ub.cancel()
		}
		delete(s.bots, userID)
	}
}

func (s *Service) FetchLoginQRCode(ctx context.Context, userID uint) (string, string, error) {
	if err := s.ensureHubAuth(ctx); err != nil {
		return "", "", err
	}

	// 已在线绑定检测：避免无条件删除旧绑定导致在线连接丢失
	existing, err := s.repo.GetUserBotByUserID(ctx, userID)
	if err == nil && existing != nil && existing.Status == domain.BotStatusOnline && existing.HubBotID != "" {
		return "", "", ErrAlreadyBound
	}

	body, _ := json.Marshal(map[string]string{})
	req, err := http.NewRequestWithContext(ctx, "POST", s.hubBaseURL+"/api/bots/bind/start", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.hubHTTP.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("hub bind start: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("hub bind start failed (HTTP %d): %s", resp.StatusCode, string(b))
	}

	var r struct {
		SessionID string `json:"session_id"`
		QRURL     string `json:"qr_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", "", fmt.Errorf("hub bind start decode: %w", err)
	}

	if err := s.repo.DeleteUserBot(ctx, userID); err != nil {
		s.log.Warn("delete_user_bot_failed", zap.Uint("user_id", userID), zap.Error(err))
	}
	account := &domain.BotAccount{
		UserID:             userID,
		Status:             domain.BotStatusPairing,
		LastStatusChangeAt: nowPtr(),
	}
	if err := s.repo.SaveUserBotAccount(ctx, account); err != nil {
		return "", "", err
	}

	s.bindStatusMu.Lock()
	s.bindStatus[fmt.Sprint(userID)] = &bindStatusEntry{
		status:       "wait",
		hubSessionID: r.SessionID,
		qrURL:        r.QRURL,
	}
	s.bindStatusMu.Unlock()

	// Start watching bind status via Hub WebSocket
	go s.watchBindStatus(context.Background(), fmt.Sprint(userID), r.SessionID)

	return r.SessionID, r.QRURL, nil
}

func (s *Service) watchBindStatus(ctx context.Context, userIDKey, sessionID string) {
	if err := s.ensureHubAuth(ctx); err != nil {
		s.setBindStatus(userIDKey, "error", "", "")
		return
	}

	u := s.hubBaseURL + "/api/bots/bind/status/" + sessionID
	u = "ws" + u[4:] // http:// → ws://

	hubURL, _ := url.Parse(s.hubBaseURL)
	cookies := s.hubCookieJar.Cookies(hubURL)
	header := http.Header{}
	for _, c := range cookies {
		header.Add("Cookie", c.Name+"="+c.Value)
	}

	conn, _, err := s.hubWS.Dial(u, header)
	if err != nil {
		s.log.Error("hub_bind_ws_dial", zap.Error(err))
		s.setBindStatus(userIDKey, "error", "", "")
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			s.log.Warn("hub_bind_ws_read", zap.Error(err))
			s.setBindStatus(userIDKey, "error", "", "")
			return
		}
		var ev struct {
			Event   string          `json:"event"`
			Status  string          `json:"status"`
			BotID   string          `json:"bot_id"`
			QRURL   string          `json:"qr_url"`
			IsNew   bool            `json:"is_new"`
			Message string          `json:"message"`
			Raw     json.RawMessage `json:"data,omitempty"`
		}
		if err := json.Unmarshal(msg, &ev); err != nil {
			continue
		}

		switch ev.Event {
		case "status":
			switch ev.Status {
			case "wait":
				s.setBindStatus(userIDKey, "wait", "", ev.QRURL)
			case "scanned":
				s.setBindStatus(userIDKey, "scanned", "", "")
			case "refreshed":
				if ev.QRURL != "" {
					s.setBindStatus(userIDKey, "wait", "", ev.QRURL)
					// update QR URL in our DB/store for frontend refresh
				}
			case "connected":
				s.setBindStatus(userIDKey, "confirmed", ev.BotID, "")
				s.onBotConfirmed(ctx, userIDKey, ev.BotID)
				return
			}
		case "error":
			s.setBindStatus(userIDKey, "error", "", "")
			s.log.Error("hub_bind_error", zap.String("message", ev.Message))
			return
		}
	}
}

func (s *Service) setBindStatus(userIDKey, status, botID, qrURL string) {
	s.bindStatusMu.Lock()
	defer s.bindStatusMu.Unlock()
	if entry, ok := s.bindStatus[userIDKey]; ok && entry != nil {
		entry.status = status
		if botID != "" {
			entry.botID = botID
		}
		if qrURL != "" {
			entry.qrURL = qrURL
		}
	}
}

func (s *Service) onBotConfirmed(ctx context.Context, userIDKey, hubBotID string) {
	var userID uint
	if _, err := fmt.Sscan(userIDKey, &userID); err != nil {
		s.log.Error("parse_user_id", zap.String("key", userIDKey), zap.Error(err))
		return
	}

	// Find the bot in Hub and get its channels
	bots, err := s.listHubBots(ctx)
	if err != nil {
		s.log.Error("list_hub_bots", zap.Error(err))
		return
	}

	var matchBot *hubBotInfo
	for _, b := range bots {
		if b.ID == hubBotID {
			matchBot = &b
			break
		}
	}
	if matchBot == nil {
		s.log.Error("hub_bot_not_found", zap.String("bot_id", hubBotID))
		return
	}

	// Create a channel for receiving messages
	ch, err := s.createHubChannel(ctx, hubBotID, "deeix-channel")
	if err != nil {
		s.log.Error("create_hub_channel", zap.Error(err))
		return
	}

	// Save to DB
	account, err := s.repo.GetUserBotByUserID(ctx, userID)
	if err != nil {
		// Create a new record
		account = &domain.BotAccount{
			UserID: userID,
		}
	}
	account.HubBotID = hubBotID
	account.HubChannelKey = ch.APIKey
	account.Nickname = matchBot.Name
	if matchBot.DisplayName != "" {
		account.Nickname = matchBot.DisplayName
	}
	account.WeChatUserID = matchBot.Extra.IlinkUserID
	account.Status = domain.BotStatusOnline
	account.LastStatusChangeAt = nowPtr()
	if err := s.repo.SaveUserBotAccount(ctx, account); err != nil {
		s.log.Error("save_bot_account", zap.Error(err))
		return
	}

	// Start WebSocket watcher for messages
	s.startMsgWatcher(ctx, account)
}

func (s *Service) PollQRCodeStatus(ctx context.Context, userID string) (string, string, error) {
	s.bindStatusMu.Lock()
	entry, ok := s.bindStatus[userID]
	s.bindStatusMu.Unlock()
	if !ok {
		return "wait", "", nil
	}
	return entry.status, entry.qrURL, nil
}

func (s *Service) GetBotStatus(ctx context.Context, userID uint) (*domain.BotAccount, error) {
	account, err := s.repo.GetUserBotByUserID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrBotNotFound
	}
	return account, err
}

func (s *Service) DestroyUserBot(ctx context.Context, userID uint) error {
	if err := s.ensureHubAuth(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	if ub, ok := s.bots[userID]; ok {
		if ub.cancel != nil {
			ub.cancel()
		}
		delete(s.bots, userID)
	}
	s.mu.Unlock()

	account, err := s.repo.GetUserBotByUserID(ctx, userID)
	if err == nil && account.HubBotID != "" {
		req, _ := http.NewRequestWithContext(ctx, "DELETE", s.hubBaseURL+"/api/bots/"+account.HubBotID, nil)
		s.hubHTTP.Do(req)
	}

	if err := s.repo.DeleteUserBot(ctx, userID); err != nil {
		return err
	}
	s.writeAudit(ctx, "", userID, "wechat_bot.delete", userID, "", "", nil)
	return nil
}

func (s *Service) AdminListBots(ctx context.Context, page, pageSize int) ([]domain.BotAccountWithUser, int64, error) {
	items, total, err := s.repo.ListAllBots(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		s.backfillNickname(ctx, &items[i])
		if items[i].ConversationID != 0 {
			conv, err := s.convSvc.GetConversation(ctx, items[i].UserID, items[i].ConversationID)
			if err == nil {
				items[i].ConversationPublicID = conv.PublicID
				items[i].ConversationModel = conv.Model
			}
		}
	}
	return items, total, nil
}

func (s *Service) AdminDeleteBot(ctx context.Context, userID uint) error {
	return s.DestroyUserBot(ctx, userID)
}

func (s *Service) GetBotDefaultModel() string {
	return s.botDefaultModel()
}

func (s *Service) SetConversationRepo(repo repository.ConversationRepository) {
	s.convRepo = repo
}

func (s *Service) AdminGetBotDetail(ctx context.Context, userID uint) (*AdminBotDetailResult, error) {
	bot, err := s.repo.GetBotByUserIDWithUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.backfillNickname(ctx, bot)
	var convPublicID string
	if bot.ConversationID != 0 {
		if conv, err := s.convSvc.GetConversationByID(ctx, bot.ConversationID); err == nil {
			convPublicID = conv.PublicID
			bot.ConversationModel = conv.Model
		}
	}
	return &AdminBotDetailResult{Bot: *bot, ConversationPublicID: convPublicID, DefaultModel: s.botDefaultModel()}, nil
}

// botDefaultModel 返回机器人发消息使用的目标模型：优先跟随后台"默认会话模型"设置，
// 其次回退到配置/env 中的 wechat_bot.default_model。
func (s *Service) botDefaultModel() string {
	if s.convSvc != nil {
		if m := s.convSvc.GetConversationSystemDefaultModel(); m != "" {
			return m
		}
	}
	if s.cfg != nil {
		return s.cfg.Snapshot().WeChatBotDefaultModel
	}
	return ""
}

// botModel 返回指定 bot 的最终目标模型（P1 解析链：bot 账户配置 > 全局默认）。
func (s *Service) botModel(ub *userBot) string {
	if ub != nil && ub.account != nil {
		if m := strings.TrimSpace(ub.account.Model); m != "" {
			return m
		}
	}
	return s.botDefaultModel()
}

func (s *Service) AdminGetBotMessages(ctx context.Context, userID uint, limit int) ([]domainconversation.Message, int64, error) {
	if s.convRepo == nil {
		return nil, 0, fmt.Errorf("convRepo not set")
	}
	bot, err := s.repo.GetUserBotByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	if bot.ConversationID == 0 {
		return nil, 0, nil
	}
	return s.convRepo.ListRecentMessages(ctx, bot.ConversationID, limit)
}

// ── P2 · 后台三合一：联系人会话 / 模型配置 / 切换 / 审计 ──────────────────────

const (
	auditResourceWeChatBot    = "wechat_bot"
	switchDrainTimeout        = 10 * time.Second
	contactConversationTitle  = "微信对话"
)

// AdminContactItem 后台联系人会话列表项。
type AdminContactItem struct {
	Wxid              string    `json:"wxid"`
	ConversationID    uint      `json:"conversation_id"`
	ConversationTitle string    `json:"conversation_title,omitempty"`
	ConversationModel string    `json:"conversation_model,omitempty"`
	MessageCount      int       `json:"message_count"`
	CreatedAt         time.Time `json:"created_at"`
	SwitchedAt        time.Time `json:"switched_at"`
}

// AdminListBotContacts 返回该 bot 下每联系人的当前会话映射（按最近切换排序）。
func (s *Service) AdminListBotContacts(ctx context.Context, userID uint) ([]AdminContactItem, error) {
	bot, err := s.repo.GetUserBotByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListContactConversations(ctx, bot.ID)
	if err != nil {
		return nil, err
	}
	items := make([]AdminContactItem, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows { // 已按 updated_at DESC 排序；每联系人取最新一条
		if _, dup := seen[row.Wxid]; dup {
			continue
		}
		seen[row.Wxid] = struct{}{}
		item := AdminContactItem{
			Wxid:           row.Wxid,
			ConversationID: row.ConversationID,
			CreatedAt:      row.CreatedAt,
			SwitchedAt:     row.UpdatedAt,
		}
		if s.convSvc != nil {
			if conv, cErr := s.convSvc.GetConversationByID(ctx, row.ConversationID); cErr == nil && conv != nil {
				item.ConversationTitle = conv.Title
				item.ConversationModel = conv.Model
				item.MessageCount = conv.MessageCount
			}
			if item.ConversationTitle == "" {
				item.ConversationTitle = s.firstUserMessagePreview(ctx, bot.UserID, row.ConversationID)
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// AdminUpdateBotModel 更新 bot 专属模型配置（P1），并记录审计。
func (s *Service) AdminUpdateBotModel(ctx context.Context, operatorUserID, botUserID uint, modelName, requestID, ip, userAgent string) error {
	modelName = strings.TrimSpace(modelName)
	bot, err := s.repo.GetUserBotByUserID(ctx, botUserID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateBotModel(ctx, botUserID, modelName); err != nil {
		return err
	}
	// 同步内存态（bot 在线时立即生效，无需重启 watcher）
	s.mu.Lock()
	if ub, ok := s.bots[botUserID]; ok {
		ub.account.Model = modelName
	}
	s.mu.Unlock()
	s.writeAudit(ctx, requestID, operatorUserID, "wechat_bot.model_update", botUserID, ip, userAgent, map[string]interface{}{
		"from": bot.Model,
		"to":   modelName,
	})
	return nil
}

// AdminSwitchContactConversation 将指定联系人切换到目标会话（targetConvID=0 表示新建干净会话），
// 切换前 drain 该 bot 在途流（≤10s），并记录审计。
func (s *Service) AdminSwitchContactConversation(ctx context.Context, operatorUserID, botUserID uint, wxid string, targetConvID uint, requestID, ip, userAgent string) (uint, error) {
	wxid = strings.TrimSpace(wxid)
	if wxid == "" {
		return 0, fmt.Errorf("wxid is required")
	}
	bot, err := s.repo.GetUserBotByUserID(ctx, botUserID)
	if err != nil {
		return 0, err
	}

	// drain 在途流：watcher 串行处理消息，每 bot 至多一条在途流。
	s.mu.Lock()
	ub, online := s.bots[botUserID]
	s.mu.Unlock()
	if online {
		if !ub.waitStreamDone(switchDrainTimeout) {
			s.log.Warn("contact_switch_drain_timeout", zap.Uint("user_id", botUserID), zap.String("wxid", wxid))
		}
	}

	var fromConvID uint
	if current, gErr := s.repo.GetContactConversation(ctx, bot.ID, wxid); gErr == nil && current != nil {
		fromConvID = current.ConversationID
	}

	if targetConvID == 0 {
		conv, cErr := s.convSvc.CreateConversation(ctx, bot.UserID, contactConversationTitle, s.botModel(&userBot{account: bot}), "")
		if cErr != nil {
			return 0, cErr
		}
		targetConvID = conv.ID
	} else {
		conv, gErr := s.convSvc.GetConversationByID(ctx, targetConvID)
		if gErr != nil || conv == nil {
			return 0, fmt.Errorf("target conversation not found")
		}
		if conv.UserID != bot.UserID {
			return 0, fmt.Errorf("target conversation does not belong to bot user")
		}
	}
	if fromConvID == targetConvID {
		return targetConvID, nil // 已指向目标会话，无需变更
	}

	if _, uErr := s.repo.UpsertContactConversation(ctx, &domain.ContactConversation{
		BotAccountID:   bot.ID,
		Wxid:           wxid,
		ConversationID: targetConvID,
	}); uErr != nil {
		return 0, uErr
	}
	// 同步内存态账户（legacy 路径兜底一致性）
	s.mu.Lock()
	if online {
		ub.account.ConversationID = targetConvID
	}
	s.mu.Unlock()

	s.writeAudit(ctx, requestID, operatorUserID, "wechat_bot.contact_switch", botUserID, ip, userAgent, map[string]interface{}{
		"wxid":            wxid,
		"from_conversation": fromConvID,
		"to_conversation":   targetConvID,
	})
	return targetConvID, nil
}

// UserSwitchContactConversation 用户切换自己 bot 的联系人会话。
// 权限红线：operator 与 bot 属主均为 ctx userID，只能操作自己的 bot；
// drain/校验/落库/审计逻辑与 admin 版本完全一致。
func (s *Service) UserSwitchContactConversation(ctx context.Context, userID uint, wxid string, targetConvID uint, requestID, ip, userAgent string) (uint, error) {
	return s.AdminSwitchContactConversation(ctx, userID, userID, wxid, targetConvID, requestID, ip, userAgent)
}

// firstUserMessagePreview 会话无标题时取首条用户消息前 30 字作展示名。
func (s *Service) firstUserMessagePreview(ctx context.Context, userID uint, conversationID uint) string {
	msgs, _, err := s.convSvc.ListMessages(ctx, userID, conversationID, 1, 10)
	if err != nil {
		return ""
	}
	for i := range msgs {
		if msgs[i].Role != "user" {
			continue
		}
		runes := []rune(strings.TrimSpace(msgs[i].Content))
		if len(runes) > 30 {
			runes = runes[:30]
		}
		return string(runes)
	}
	return ""
}

// AdminGetBotAudit 查询该 bot 的操作审计（复用 audit_logs 表）。
func (s *Service) AdminGetBotAudit(ctx context.Context, userID uint, limit int) ([]domainaudit.Log, int64, error) {
	if s.auditReader == nil {
		return nil, 0, fmt.Errorf("audit reader not set")
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.auditReader.List(ctx, 1, limit, appaudit.ListFilter{
		Resource:   auditResourceWeChatBot,
		ResourceID: fmt.Sprint(userID),
	})
}

// UserListContacts 返回当前用户 bot 的联系人会话列表（用户 scope）。
func (s *Service) UserListContacts(ctx context.Context, userID uint) ([]AdminContactItem, error) {
	bot, err := s.repo.GetUserBotByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListContactConversations(ctx, bot.ID)
	if err != nil {
		return nil, err
	}
	items := make([]AdminContactItem, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, dup := seen[row.Wxid]; dup {
			continue
		}
		seen[row.Wxid] = struct{}{}
		item := AdminContactItem{
			Wxid:           row.Wxid,
			ConversationID: row.ConversationID,
			CreatedAt:      row.CreatedAt,
			SwitchedAt:     row.UpdatedAt,
		}
		if s.convSvc != nil {
			if conv, cErr := s.convSvc.GetConversationByID(ctx, row.ConversationID); cErr == nil && conv != nil {
				item.ConversationTitle = conv.Title
				item.ConversationModel = conv.Model
				item.MessageCount = conv.MessageCount
			}
			if item.ConversationTitle == "" {
				item.ConversationTitle = s.firstUserMessagePreview(ctx, bot.UserID, row.ConversationID)
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// UserGetAudit 返回当前用户自己的 wechat_bot 操作记录（三条件：ActorUserID=self AND Resource=wechat_bot AND ResourceID=self）。
func (s *Service) UserGetAudit(ctx context.Context, userID uint) ([]domainaudit.Log, error) {
	if s.auditReader == nil {
		return nil, fmt.Errorf("audit reader not set")
	}
	logs, _, err := s.auditReader.List(ctx, 1, 20, appaudit.ListFilter{
		ActorUserID: userID,
		Resource:    auditResourceWeChatBot,
		ResourceID:  fmt.Sprint(userID),
	})
	return logs, err
}

func (s *Service) writeAudit(ctx context.Context, requestID string, operatorUserID uint, action string, botUserID uint, ip, userAgent string, detail interface{}) {
	if s.auditSvc == nil {
		return
	}
	s.auditSvc.Write(ctx, requestID, operatorUserID, action, auditResourceWeChatBot, fmt.Sprint(botUserID), ip, userAgent, detail)
}

// backfillNickname 当数据库昵称或微信用户ID为空时，从 Hub 拉取并回填。
func (s *Service) backfillNickname(ctx context.Context, bot *domain.BotAccountWithUser) {
	if bot == nil || bot.HubBotID == "" {
		return
	}
	if bot.Nickname != "" && bot.WeChatUserID != "" {
		return
	}
	bots, err := s.listHubBots(ctx)
	if err != nil {
		return
	}
	for _, b := range bots {
		if b.ID == bot.HubBotID {
			if bot.Nickname == "" {
				name := b.Name
				if b.DisplayName != "" {
					name = b.DisplayName
				}
				if name != "" {
					bot.Nickname = name
					if err := s.repo.UpdateUserBotNickname(ctx, bot.ID, name); err != nil {
						s.log.Warn("update_bot_nickname", zap.Uint("user_id", bot.UserID), zap.Error(err))
					}
				}
			}
			if bot.WeChatUserID == "" && b.Extra.IlinkUserID != "" {
				bot.WeChatUserID = b.Extra.IlinkUserID
				if err := s.repo.UpdateUserBotWeChatUserID(ctx, bot.ID, b.Extra.IlinkUserID); err != nil {
					s.log.Warn("update_bot_wechat_user_id", zap.Uint("user_id", bot.UserID), zap.Error(err))
				}
			}
			return
		}
	}
}

func (s *Service) startMsgWatcher(ctx context.Context, account *domain.BotAccount) {
	ctx, cancel := context.WithCancel(ctx)
	ub := &userBot{
		hubBotID: account.HubBotID,
		cancel:   cancel,
		account:  account,
	}

	s.mu.Lock()
	if old, ok := s.bots[account.UserID]; ok && old.cancel != nil {
		old.cancel()
	}
	s.bots[account.UserID] = ub
	s.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("msg_watcher_panic",
					zap.Uint("user_id", account.UserID),
					zap.String("hub_bot_id", account.HubBotID),
					zap.Any("recover", r),
				)
				// 重新拉起 watcher（避免 panic 后永久消失）；重载 account 拿最新 cursor，2s 防崩溃热循环
				go func() {
					time.Sleep(2 * time.Second)
					fresh, err := s.repo.GetUserBotByUserID(context.Background(), account.UserID)
					if err != nil || fresh == nil {
						s.log.Warn("watcher_restart_no_account", zap.Uint("user_id", account.UserID))
						return
					}
					s.startMsgWatcher(context.Background(), fresh)
				}()
			}
		}()
		lastID := account.HubCursorID
		for {
			if err := s.ensureHubAuth(ctx); err != nil {
				s.log.Error("hub_auth", zap.Uint("user_id", account.UserID), zap.Error(err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
				continue
			}

			u := fmt.Sprintf("%s/api/bots/%s/messages?limit=10", s.hubBaseURL, account.HubBotID)
			req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
			resp, err := s.hubHTTP.Do(req)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
				continue
			}
			if resp.StatusCode != http.StatusOK {
				b, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				s.log.Warn("hub_messages_status",
					zap.Uint("user_id", account.UserID),
					zap.String("hub_bot_id", account.HubBotID),
					zap.Int("status", resp.StatusCode),
					zap.String("body", string(b)),
				)
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
				continue
			}

			var result struct {
				Messages []struct {
					ID         int64  `json:"id"`
					Direction  string `json:"direction"`
					FromUserID string `json:"from_user_id"`
					ItemList   []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"item_list"`
				} `json:"messages"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				resp.Body.Close()
				s.log.Warn("hub_messages_decode",
					zap.Uint("user_id", account.UserID),
					zap.String("hub_bot_id", account.HubBotID),
					zap.Error(err),
				)
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
				continue
			}
			resp.Body.Close()

			maxID := lastID
			for _, m := range result.Messages {
				if m.ID > maxID {
					maxID = m.ID
				}
			}

			// first run: skip history, just establish cursor
			if lastID == 0 {
				lastID = maxID
				if err := s.repo.UpdateBotCursor(ctx, account.UserID, lastID); err != nil {
					s.log.Error("init_bot_cursor", zap.Uint("user_id", account.UserID), zap.Int64("cursor", lastID), zap.Error(err))
				}
				continue
			}

			processedID := lastID
			for i := len(result.Messages) - 1; i >= 0; i-- {
				m := result.Messages[i]
				if m.ID <= lastID {
					continue
				}
				if m.Direction != "inbound" {
					processedID = m.ID
					continue
				}
				var text string
				for _, item := range m.ItemList {
					if item.Type == "text" && item.Text != "" {
						text = item.Text
						break
					}
				}
				if text == "" {
					processedID = m.ID
					continue
				}

				data, _ := json.Marshal(map[string]interface{}{
					"seq_id":    m.ID,
					"sender":    m.FromUserID,
					"timestamp": time.Now().Unix(),
					"items":     []map[string]string{{"type": "text", "text": text}},
				})
				if !s.handleIncomingMessage(data, account.UserID) {
					break
				}
				processedID = m.ID
			}
			if processedID > lastID {
				lastID = processedID
				if err := s.repo.UpdateBotCursor(ctx, account.UserID, lastID); err != nil {
					s.log.Error("update_bot_cursor", zap.Uint("user_id", account.UserID), zap.Int64("cursor", lastID), zap.Error(err))
				}
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}()
}

// ensureConversation 确保用户有一个有效的微信对话会话，不存在则创建。
// perContactSessionsEnabled 返回联系人级会话开关状态（P0-b 特性开关）。
func (s *Service) perContactSessionsEnabled() bool {
	return s.cfg != nil && s.cfg.Snapshot().WeChatBotPerContactSessions
}

// ensureConversation 解析本次消息应使用的会话：
//   - 开关开启且 sender 非空：按 (bot, wxid) 联系人维度查/建会话（wechat_conversations）
//   - 否则：走旧账户级逻辑（wechat_bot_accounts.conversation_id 单会话，跨联系人共享上下文）
func (s *Service) ensureConversation(ctx context.Context, ub *userBot, sender string) (uint, error) {
	if s.perContactSessionsEnabled() && strings.TrimSpace(sender) != "" {
		return s.ensureContactConversation(ctx, ub, strings.TrimSpace(sender))
	}
	return s.ensureLegacyConversation(ctx, ub)
}

func (s *Service) ensureLegacyConversation(ctx context.Context, ub *userBot) (uint, error) {
	if ub.account.ConversationID != 0 {
		return ub.account.ConversationID, nil
	}
	conv, err := s.convSvc.CreateConversation(ctx, ub.account.UserID, "微信对话", s.botModel(ub), "")
	if err != nil {
		return 0, err
	}
	ub.account.ConversationID = conv.ID
	if err := s.repo.SaveUserBotAccount(ctx, ub.account); err != nil {
		s.log.Warn("save_conversation_id_failed", zap.Error(err))
	}
	return conv.ID, nil
}

func (s *Service) ensureContactConversation(ctx context.Context, ub *userBot, wxid string) (uint, error) {
	cc, err := s.repo.GetContactConversation(ctx, ub.account.ID, wxid)
	if err == nil && cc != nil && cc.ConversationID != 0 {
		return cc.ConversationID, nil
	}
	conv, err := s.convSvc.CreateConversation(ctx, ub.account.UserID, "微信对话", s.botModel(ub), "")
	if err != nil {
		return 0, err
	}
	if _, uErr := s.repo.UpsertContactConversation(ctx, &domain.ContactConversation{
		BotAccountID:   ub.account.ID,
		Wxid:           wxid,
		ConversationID: conv.ID,
	}); uErr != nil {
		s.log.Warn("save_contact_conversation_failed", zap.String("wxid", wxid), zap.Error(uErr))
	}
	s.log.Info("contact_conversation_created",
		zap.Uint("user_id", ub.account.UserID),
		zap.String("wxid", wxid),
		zap.Uint("conversation_id", conv.ID))
	return conv.ID, nil
}

func (s *Service) handleIncomingMessage(data json.RawMessage, userID uint) bool {
	if s.convSvc == nil {
		return false
	}

	var msg struct {
		SeqID     int64  `json:"seq_id"`
		Sender    string `json:"sender"`
		Timestamp int64  `json:"timestamp"`
		Items     []struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			FileName string `json:"file_name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return false
	}

	var text string
	for _, item := range msg.Items {
		if item.Type == "text" && item.Text != "" {
			text = item.Text
			break
		}
	}
	if text == "" {
		return true
	}

	s.log.Info("wechat_message", zap.String("text", text), zap.Uint("user_id", userID))

	allowed, err := s.CanUseWeChatBot(context.Background(), userID)
	if err != nil {
		s.log.Error("wechat_access_check", zap.Uint("user_id", userID), zap.Error(err))
		return false
	}
	if !allowed {
		s.sendWeChatText(userID, msg.Sender, "你的微信机器人权限已过期，已停止服务，请联系管理员")
		s.stopUserBot(userID)
		return true
	}

	s.mu.Lock()
	ub, ok := s.bots[userID]
	s.mu.Unlock()
	if !ok {
		// 入口自愈：watcher 缺失时尝试从 DB 恢复
		account, err := s.repo.GetUserBotByUserID(context.Background(), userID)
		if err != nil || account == nil {
			s.log.Warn("inbound_msg_dropped_no_account", zap.Uint("user_id", userID))
			return false
		}
		if account.HubBotID == "" {
			s.log.Warn("inbound_msg_dropped_no_hubbot", zap.Uint("user_id", userID))
			return false
		}
		allowed, _ := s.CanUseWeChatBot(context.Background(), userID)
		if allowed {
			s.log.Info("inbound_self_heal_start_watcher", zap.Uint("user_id", userID))
			s.startMsgWatcher(context.Background(), account)
		} else {
			s.log.Warn("inbound_msg_dropped_no_permission", zap.Uint("user_id", userID))
		}
		return false
	}

	conversationID, err := s.ensureConversation(context.Background(), ub, msg.Sender)
	if err != nil {
		s.log.Error("ensure_conversation_failed", zap.Error(err))
		return false
	}

	done := make(chan struct{})
	ub.setStreamDone(done)
	defer func() {
		close(done)
		ub.clearStreamDone(done)
	}()
	go func() {
		s.sendWeChatTyping(userID, msg.Sender, true)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				s.sendWeChatTyping(userID, msg.Sender, false)
				return
			case <-ticker.C:
				s.sendWeChatTyping(userID, msg.Sender, true)
			}
		}
	}()

	sendAttempt := func(model string) error {
		splitter := newMDSplitter(chunkRuneLimit)
		_, err := s.convSvc.StreamMessage(context.Background(), appconversation.SendMessageInput{
			UserID:             userID,
			ConversationID:     conversationID,
			ContentType:        "text",
			Content:            text,
			PlatformModelName:  model,
			WithYouPoolProfile: model,
			WithYouBotID:       ub.hubBotID,
		}, func(delta string) error {
			splitter.feed(delta, func(chunk string) {
				s.sendWeChatText(userID, msg.Sender, chunk)
			})
			return nil
		})
		if err != nil {
			return err
		}
		splitter.done(func(chunk string) {
			s.sendWeChatText(userID, msg.Sender, chunk)
		})
		return nil
	}

	err = sendAttempt(s.botModel(ub))
	if err == nil {
		return true
	}

	switch {
	case errors.Is(err, appconversation.ErrConversationNotFound):
		// 会话被删：重建会话后重试一次。
		conv, cErr := s.convSvc.CreateConversation(context.Background(), userID, "微信对话", s.botModel(ub), "")
		if cErr != nil {
			s.log.Error("recreate_conversation_failed", zap.Error(cErr))
			s.sendWeChatText(userID, msg.Sender, "服务暂时不可用，请稍后再试")
			return true
		}
		conversationID = conv.ID
		if perContact := s.perContactSessionsEnabled() && strings.TrimSpace(msg.Sender) != ""; perContact {
			// 联系人级模式：更新该联系人的会话映射。
			if _, uErr := s.repo.UpsertContactConversation(context.Background(), &domain.ContactConversation{
				BotAccountID:   ub.account.ID,
				Wxid:           strings.TrimSpace(msg.Sender),
				ConversationID: conv.ID,
			}); uErr != nil {
				s.log.Error("save_contact_conversation_failed", zap.Error(uErr))
			}
		} else {
			ub.account.ConversationID = conv.ID
			if err := s.repo.SaveUserBotAccount(context.Background(), ub.account); err != nil {
				s.log.Error("save_conversation_id_failed", zap.Error(err))
			}
		}
		if retryErr := sendAttempt(s.botModel(ub)); retryErr != nil {
			s.log.Error("stream_message_retry_failed", zap.Error(retryErr))
			s.sendWeChatText(userID, msg.Sender, "服务暂时不可用，请稍后再试")
		}
		return true
	case errors.Is(err, appconversation.ErrModelRouteNotConfigured) ||
		errors.Is(err, appconversation.ErrModelAccessDenied):
		// 默认模型失效/无可用路由：切换到第一个可用模型重试。
		fallback := s.firstAvailableModel(context.Background(), userID)
		if fallback == "" {
			s.log.Error("no_fallback_model", zap.Error(err))
			s.sendWeChatText(userID, msg.Sender, "模型暂时不可用，请稍后再试")
			return true
		}
		s.log.Warn("model_fallback", zap.String("model", fallback), zap.Error(err))
		if retryErr := sendAttempt(fallback); retryErr != nil {
			s.log.Error("stream_message_fallback_failed", zap.Error(retryErr))
			s.sendWeChatText(userID, msg.Sender, "模型暂时不可用，请稍后再试")
		}
		return true
	default:
		s.log.Error("stream_message_failed", zap.Error(err))
		s.sendWeChatText(userID, msg.Sender, "服务暂时不可用，请稍后再试")
		return true
	}
}

// firstAvailableModel 返回当前用户可用的第一个聊天模型名。
func (s *Service) firstAvailableModel(ctx context.Context, userID uint) string {
	if s.channelSvc == nil {
		return ""
	}
	views, err := s.channelSvc.ListActiveModels(ctx, userID)
	if err != nil {
		return ""
	}
	for _, v := range views {
		if name := strings.TrimSpace(v.PlatformModelName); name != "" {
			return name
		}
	}
	return ""
}

const chunkRuneLimit = 200

// typingTicket caches a typing_ticket per bot (channel key).
type typingTicket struct {
	ticket string
	expiry time.Time
}

var (
	typingMu      sync.Mutex
	typingTickets = map[string]*typingTicket{}
)

func (s *Service) ensureTypingTicket(ctx context.Context, channelKey string) (string, error) {
	typingMu.Lock()
	if t, ok := typingTickets[channelKey]; ok && time.Now().Before(t.expiry) {
		typingMu.Unlock()
		return t.ticket, nil
	}
	typingMu.Unlock()

	u := fmt.Sprintf("%s/api/v1/channels/config?key=%s", s.hubBaseURL, channelKey)
	req, _ := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.hubHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return "", ErrInvalidChannelKey
	}

	var r struct {
		TypingTicket string `json:"typing_ticket"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}

	typingMu.Lock()
	typingTickets[channelKey] = &typingTicket{ticket: r.TypingTicket, expiry: time.Now().Add(20 * time.Hour)}
	typingMu.Unlock()
	return r.TypingTicket, nil
}

// refreshChannelKey 从 hub 拉取该 bot 第一个 enabled channel 的 api_key 并回写 DB/内存。
// 用于 DB 里 hub_channel_key 存错或过期时的自愈，避免依赖静态错误值。
func (s *Service) refreshChannelKey(ctx context.Context, userID uint) error {
	if err := s.ensureHubAuth(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	ub, ok := s.bots[userID]
	s.mu.Unlock()
	if !ok || ub.hubBotID == "" {
		return errors.New("no active bot")
	}

	u := fmt.Sprintf("%s/api/bots/%s/channels", s.hubBaseURL, ub.hubBotID)
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	resp, err := s.hubHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var channels []struct {
		APIKey  string `json:"api_key"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		return err
	}
	for _, ch := range channels {
		if ch.Enabled && ch.APIKey != "" {
			if err := s.repo.UpdateBotChannelKey(ctx, userID, ch.APIKey); err != nil {
				s.log.Warn("update_bot_channel_key", zap.Uint("user_id", userID), zap.Error(err))
			}
			ub.account.HubChannelKey = ch.APIKey
			s.log.Info("typing_channel_key_refreshed", zap.Uint("user_id", userID))
			return nil
		}
	}
	return errors.New("no enabled channel")
}

func (s *Service) sendWeChatTyping(userID uint, _ string, typing bool) {
	s.mu.Lock()
	ub, ok := s.bots[userID]
	s.mu.Unlock()
	if !ok || ub.account.HubChannelKey == "" {
		return
	}

	status := "typing"
	if !typing {
		status = "cancel"
	}

	ticket, err := s.ensureTypingTicket(context.Background(), ub.account.HubChannelKey)
	if errors.Is(err, ErrInvalidChannelKey) {
		// 一次自愈：key 无效时从 hub 刷新后重试
		if rerr := s.refreshChannelKey(context.Background(), userID); rerr != nil {
			return
		}
		ticket, err = s.ensureTypingTicket(context.Background(), ub.account.HubChannelKey)
	}
	if err != nil {
		return
	}

	body, _ := json.Marshal(map[string]string{
		"ticket": ticket,
		"status": status,
	})
	req, _ := http.NewRequestWithContext(context.Background(), "POST",
		s.hubBaseURL+"/api/v1/channels/typing?key="+ub.account.HubChannelKey, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.hubHTTP.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func (s *Service) sendWeChatText(userID uint, to, text string) {
	if err := s.ensureHubAuth(context.Background()); err != nil {
		return
	}

	s.mu.Lock()
	ub, ok := s.bots[userID]
	s.mu.Unlock()
	if !ok || ub.hubBotID == "" {
		return
	}

	body, _ := json.Marshal(map[string]string{
		"recipient": to,
		"text":      text,
	})
	req, err := http.NewRequestWithContext(context.Background(), "POST",
		s.hubBaseURL+"/api/bots/"+ub.hubBotID+"/send", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.hubHTTP.Do(req)
	if err != nil {
		s.log.Error("hub_send", zap.Error(err))
		return
	}
	resp.Body.Close()
}

func (s *Service) GetBotConversationPublicID(ctx context.Context, userID uint) string {
	account, err := s.repo.GetUserBotByUserID(ctx, userID)
	if err != nil || account.ConversationID == 0 {
		return ""
	}
	conv, err := s.convSvc.GetConversation(ctx, userID, account.ConversationID)
	if err != nil {
		return ""
	}
	return conv.PublicID
}

// ---- Hub API helpers ----

type hubBotInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	Extra       struct {
		IlinkUserID string `json:"ilink_user_id"`
		BotID       string `json:"bot_id"`
	} `json:"extra"`
}

func (s *Service) listHubBots(ctx context.Context) ([]hubBotInfo, error) {
	if err := s.ensureHubAuth(ctx); err != nil {
		return nil, err
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", s.hubBaseURL+"/api/bots", nil)
	resp, err := s.hubHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var bots []hubBotInfo
	if err := json.NewDecoder(resp.Body).Decode(&bots); err != nil {
		return nil, err
	}
	return bots, nil
}

// reconcileHubState 巡检 DB 中所有 online 记录的 Hub 实际状态，检测漂移并告警。
func (s *Service) reconcileHubState(ctx context.Context) {
	accounts, err := s.repo.GetAllOnlineBots(ctx)
	if err != nil {
		s.log.Error("reconcile_get_online_bots_failed", zap.Error(err))
		return
	}
	hubBots, err := s.listHubBots(ctx)
	if err != nil {
		s.log.Error("reconcile_list_hub_bots_failed", zap.Error(err))
		return
	}
	hubMap := make(map[string]struct{}, len(hubBots))
	for _, b := range hubBots {
		hubMap[b.ID] = struct{}{}
	}
	for _, account := range accounts {
		if account.HubBotID == "" {
			continue
		}
		if _, ok := hubMap[account.HubBotID]; !ok {
			// DB 记录 online，但 Hub 上已不存在 → 漂移告警
			s.log.Warn("hub_bot_drift_detected",
				zap.Uint("user_id", account.UserID),
				zap.String("hub_bot_id", account.HubBotID),
				zap.String("db_status", string(account.Status)),
			)
		}
	}
}

type hubChannelInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	APIKey string `json:"api_key"`
}

func (s *Service) createHubChannel(ctx context.Context, botID, name string) (*hubChannelInfo, error) {
	if err := s.ensureHubAuth(ctx); err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]string{"name": name})
	req, _ := http.NewRequestWithContext(ctx, "POST",
		s.hubBaseURL+"/api/bots/"+botID+"/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.hubHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ch hubChannelInfo
	if err := json.NewDecoder(resp.Body).Decode(&ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}

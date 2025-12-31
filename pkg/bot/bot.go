package bot

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/yingxiaomo/HomeOps/config"
	"github.com/yingxiaomo/HomeOps/pkg/ai"
	"github.com/yingxiaomo/HomeOps/pkg/openclash"
	"github.com/yingxiaomo/HomeOps/pkg/openwrt"
	"github.com/yingxiaomo/HomeOps/pkg/utils"

	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	TeleBot *tele.Bot
	Gemini  *ai.GeminiClient
	Store   *SessionStore
}

type SessionStore struct {
	mu   sync.RWMutex
	Data map[int64]map[string]interface{}
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		Data: make(map[int64]map[string]interface{}),
	}
}

func (s *SessionStore) Get(userID int64, key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if userStore, ok := s.Data[userID]; ok {
		return userStore[key]
	}
	return nil
}

func (s *SessionStore) Set(userID int64, key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Data[userID]; !ok {
		s.Data[userID] = make(map[string]interface{})
	}
	s.Data[userID][key] = value
}

func NewBot() *Bot {
	pref := tele.Settings{
		Token:  config.AppConfig.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	
	if config.AppConfig.TGBaseURL != "" {
		pref.URL = config.AppConfig.TGBaseURL
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	return &Bot{
		TeleBot: b,
		Gemini:  ai.NewGeminiClient(),
		Store:   NewSessionStore(),
	}
}

func (b *Bot) Start() {
	// Middleware
	b.TeleBot.Use(b.AuthMiddleware)

	// Commands
	b.TeleBot.Handle("/start", b.HandleStart)
	b.TeleBot.Handle("/ai", b.HandleAI)
	b.TeleBot.Handle("/wrt", openwrt.HandleWrtMain)
	b.TeleBot.Handle("/sticker", b.HandleStickerMenu)
	b.TeleBot.Handle("/mail", b.HandleMailMenu)
	b.TeleBot.Handle("/info", b.HandleInfo)
	b.TeleBot.Handle("/id", b.HandleInfo) // Alias


	// Callback Queries
	b.TeleBot.Handle(tele.OnCallback, b.HandleCallback)

	// Text & Media
	b.TeleBot.Handle(tele.OnText, b.HandleText)
	b.TeleBot.Handle(tele.OnPhoto, b.HandlePhoto)
	b.TeleBot.Handle(tele.OnSticker, b.HandleSticker)

	log.Printf("Go Bot started on %s", b.TeleBot.Me.Username)
	b.TeleBot.Start()
}

func (b *Bot) AuthMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		user := c.Sender()
		if user == nil {
			return next(c)
		}
		// TODO: Implement proper whitelist check
		if !utils.HasPermission(user.ID, "") {
			log.Printf("Unauthorized access: %d", user.ID)
			return nil
		}
		return next(c)
	}
}

func (b *Bot) HandleStart(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🤖 AI 助手", "ai_toggle"), menu.Data("📡 OpenWrt", "wrt_main")),
		menu.Row(menu.Data("🚀 OpenClash", "clash_main"), menu.Data("📧 临时邮箱", "mail_main")),
		menu.Row(menu.Data("🖼️ 贴纸转换", "sticker_main")),
	)
	return c.Send("👋 欢迎使用 Go 全功能机器人！\n请选择功能：", menu)
}

func (b *Bot) HandleCallback(c tele.Context) error {
	data := c.Callback().Data
	
	// Routing based on prefix
	switch {
	case data == "start_main":
		return b.HandleStart(c)
	case data == "ai_toggle":
		return b.HandleAI(c)
	case strings.HasPrefix(data, "wrt_"):
		return openwrt.HandleCallback(c, data)
	case strings.HasPrefix(data, "clash_"):
		return openclash.HandleCallback(c, data)
	case strings.HasPrefix(data, "sticker_"):
		return b.HandleStickerCallback(c, data)
	case strings.HasPrefix(data, "mail_"):
		return b.HandleMailCallback(c, data)
	}
	return c.Respond()
}

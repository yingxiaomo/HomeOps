package bot

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yingxiaomo/homeops/config"
	"github.com/yingxiaomo/homeops/pkg/ai"
	"github.com/yingxiaomo/homeops/pkg/openclash"
	"github.com/yingxiaomo/homeops/pkg/openwrt"
	"github.com/yingxiaomo/homeops/pkg/session"
	"github.com/yingxiaomo/homeops/pkg/utils"

	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	TeleBot *tele.Bot
	Gemini  *ai.GeminiClient
	Store   *session.SessionStore
}

func NewBot() *Bot {
	pref := tele.Settings{
		Token:   config.AppConfig.BotToken,
		Poller:  &tele.LongPoller{Timeout: 10 * time.Second},
		Verbose: true,
	}

	if config.AppConfig.TGBaseURL != "" {
		pref.URL = config.AppConfig.TGBaseURL
	}

	log.Printf("Bot Config - Token: ...%s", config.AppConfig.BotToken[len(config.AppConfig.BotToken)-5:])
	log.Printf("Bot Config - Proxy: %s", config.AppConfig.TGProxy)

	if config.AppConfig.TGProxy != "" {
		proxyUrl, err := url.Parse(config.AppConfig.TGProxy)
		if err != nil {
			log.Printf("Invalid Proxy URL: %v", err)
		} else {
			pref.Client = &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyUrl),
				},
			}
		}
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	return &Bot{
		TeleBot: b,
		Gemini:  ai.NewGeminiClient(),
		Store:   session.GlobalStore,
	}
}

func (b *Bot) Start() {
	b.TeleBot.Use(b.LogMiddleware)
	b.TeleBot.Use(b.AuthMiddleware)
	b.TeleBot.Handle("/start", b.HandleStart)
	b.TeleBot.Handle("/ai", b.HandleAI)
	b.TeleBot.Handle("/wrt", openwrt.HandleWrtMain)
	b.TeleBot.Handle("/sticker", b.HandleStickerMenu)
	b.TeleBot.Handle("/mail", b.HandleMailMenu)
	b.TeleBot.Handle("/grant", b.HandleGrant)
	b.TeleBot.Handle("/revoke", b.HandleRevoke)
	b.TeleBot.Handle("/users", b.HandleListUsers)
	b.TeleBot.Handle("/info", b.HandleInfo)
	b.TeleBot.Handle("/id", b.HandleInfo)
	b.TeleBot.Handle(tele.OnCallback, b.HandleCallback)
	b.TeleBot.Handle(tele.OnText, b.HandleText)
	b.TeleBot.Handle(tele.OnPhoto, b.HandlePhoto)
	b.TeleBot.Handle(tele.OnSticker, b.HandleSticker)

	openwrt.StartIPMonitor(b.TeleBot)

	log.Printf("Go Bot started on %s", b.TeleBot.Me.Username)
	b.TeleBot.Start()
}

func (b *Bot) LogMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		user := c.Sender()
		updateID := c.Update().ID
		if user != nil {
			log.Printf("[%d] Update from %s (ID: %d): %s", updateID, user.Username, user.ID, c.Text())
			if c.Callback() != nil {
				log.Printf("[%d] Callback data: %s", updateID, c.Callback().Data)
			}
		}
		return next(c)
	}
}

func (b *Bot) AuthMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		user := c.Sender()
		if user == nil {
			return next(c)
		}

		if !utils.HasPermission(user.ID, "") {
			log.Printf("Unauthorized access: %d", user.ID)
			return nil
		}
		return next(c)
	}
}

func (b *Bot) getMainMenu() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🤖 AI 助手", "ai_toggle"), menu.Data("� 批量输入", "batch_start")),
		menu.Row(menu.Data("📡 OpenWrt", "wrt_main"), menu.Data("🚀 OpenClash", "clash_main")),
		menu.Row(menu.Data("📧 临时邮箱", "mail_main"), menu.Data("🖼️ 贴纸转换", "sticker_main")),
	)
	return menu
}

func (b *Bot) HandleStart(c tele.Context) error {
	text := "🤖 **HomeOps 已连接**\n\n请选择功能菜单："
	return c.Send(text, b.getMainMenu(), tele.ModeMarkdown)
}

func (b *Bot) HandleCallback(c tele.Context) error {
	data := strings.TrimSpace(c.Callback().Data)
	data = strings.TrimPrefix(data, "\f")

	switch {
	case data == "start_main":
		return b.HandleStart(c)
	case data == "ai_toggle":
		return b.HandleAI(c)
	case data == "batch_start":
		return b.HandleBatchStart(c)
	case data == "batch_end":
		return b.HandleBatchEnd(c)
	case strings.HasPrefix(data, "wrt_"):
		if err := openwrt.HandleCallback(c, data); err != nil {
			log.Printf("Error handling OpenWrt callback: %v", err)
			return c.Respond(&tele.CallbackResponse{Text: "操作失败", ShowAlert: true})
		}
		return nil
	case strings.HasPrefix(data, "clash_"):
		return openclash.HandleCallback(c, data)
	case strings.HasPrefix(data, "G_") || strings.HasPrefix(data, "S_"):
		return openclash.HandleCallback(c, data)
	case strings.HasPrefix(data, "sticker_"):
		return b.HandleStickerCallback(c, data)
	case strings.HasPrefix(data, "mail_"):
		return b.HandleMailCallback(c, data)
	default:
		log.Printf("Unknown callback data: %s", data)
		return c.Respond()
	}
	return c.Respond()
}

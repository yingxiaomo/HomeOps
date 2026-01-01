package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken           string
	AdminID            int64
	TGBaseURL          string
	TGProxy            string
	GeminiAPIKeys      []string
	OpenWrtHost        string
	OpenWrtPort        int
	OpenWrtUser        string
	OpenWrtPass        string
	OpenWrtKeyFile     string
	OpenClashAPIURL    string
	OpenClashAPISecret string
	AdgURL             string
	AdgUser            string
	AdgPass            string
	AdgToken           string
	AdgLeasesMode      string
}

var AppConfig *Config

func LoadConfig() {
	godotenv.Load(".env")

	// Set timezone to Asia/Shanghai
	if tz := os.Getenv("TZ"); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			time.Local = loc
		}
	} else {
		// Default to Asia/Shanghai if TZ not set
		if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
			time.Local = loc
		}
	}

	AppConfig = &Config{
		BotToken:           os.Getenv("TG_BOT_TOKEN"),
		AdminID:            getEnvAsInt("ADMIN_ID", 0),
		TGBaseURL:          os.Getenv("TG_BASE_URL"),
		TGProxy:            os.Getenv("TG_PROXY"),
		GeminiAPIKeys:      getEnvAsSlice("GEMINI_API_KEY"),
		OpenWrtHost:        os.Getenv("OPENWRT_HOST"),
		OpenWrtPort:        int(getEnvAsInt("OPENWRT_PORT", 22)),
		OpenWrtUser:        getEnvAsIntStr("OPENWRT_USER", "root"),
		OpenWrtPass:        os.Getenv("OPENWRT_PASS"),
		OpenWrtKeyFile:     os.Getenv("OPENWRT_KEY_FILE"),
		OpenClashAPIURL:    getEnvAsIntStr("OPENCLASH_API_URL", "http://127.0.0.1:9090"),
		OpenClashAPISecret: os.Getenv("OPENCLASH_API_SECRET"),
		AdgURL:             os.Getenv("ADG_URL"),
		AdgUser:            os.Getenv("ADG_USER"),
		AdgPass:            os.Getenv("ADG_PASS"),
		AdgToken:           os.Getenv("ADG_TOKEN"),
		AdgLeasesMode:      getEnvAsIntStr("ADG_LEASES_MODE", "auto"),
	}

	if AppConfig.BotToken == "" {
		AppConfig.BotToken = os.Getenv("BOT_TOKEN")
	}
}

func getEnvAsInt(key string, defaultVal int64) int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultVal
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return defaultVal
	}
	return value
}

func getEnvAsIntStr(key string, defaultVal string) string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultVal
	}
	return valueStr
}

func getEnvAsSlice(key string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return []string{}
	}
	parts := strings.Split(valueStr, ",")
	var result []string
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

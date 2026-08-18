package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds all server configuration, mirroring the original AnimeGarden
// server env-var based configuration.
type Config struct {
	// Server
	Port        int
	PublicURL   string // e.g. https://api.animes.garden
	DataDir     string
	DatabaseURL string
	RedisURL    string

	// Cron jobs
	Cron bool

	// Telegram push
	TelegramBotToken  string
	TelegramChannelID string

	// Admin
	AdminToken string

	// Scraping
	ScrapeInterval time.Duration
	ScrapeTimeout  time.Duration

	// CORS origins for the web frontend
	CORSOrigins []string

	// AppHost mirrors APP_HOST: site hostname for feed/detail URLs.
	AppHost string
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

// exeDir returns the directory of the running binary, used to anchor the
// default data directory regardless of the process working directory.
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// Load reads configuration from environment variables.
func Load() *Config {
	cfg := &Config{
		Port:              envInt("PORT", 9701),
		PublicURL:         env("PUBLIC_URL", "http://localhost:9701"),
		DataDir:           env("DATA_DIR", "data"),
		DatabaseURL:       env("DATABASE_URL", ""),
		RedisURL:          env("REDIS_URL", ""),
		Cron:              envBool("CRON", true),
		TelegramBotToken:  env("TELEGRAM_BOT_TOKEN", ""),
		TelegramChannelID: env("TELEGRAM_CHANNEL_ID", ""),
		AdminToken:        env("ADMIN_TOKEN", ""),
		ScrapeInterval:    time.Duration(envInt("SCRAPE_INTERVAL_MIN", 10)) * time.Minute,
		ScrapeTimeout:     time.Duration(envInt("SCRAPE_TIMEOUT_SEC", 60)) * time.Second,
		AppHost:           env("APP_HOST", "animes.garden"),
	}
	// Default CORS: allow the frontend dev origin
	cfg.CORSOrigins = []string{"http://localhost:9700"}
	return cfg
}

// PublicURLHost returns the site host for feed/detail URLs (APP_HOST).
func (c *Config) PublicURLHost() string {
	return c.AppHost
}

// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"net/url"
	"os"
	"time"
)

type Config struct {
	Env             string // local | staging | production
	HTTPAddr        string
	DatabaseDSN     string
	ShutdownTimeout time.Duration

	// AppOrigin is the browser-facing origin (the Next.js Worker), used for
	// CSRF origin checks and links in emails.
	AppOrigin    string
	CookieSecure bool
	MailDriver   string // log
}

func Load() (Config, error) {
	c := Config{
		Env:             getenv("APP_ENV", "local"),
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabaseDSN:     os.Getenv("DATABASE_DSN"),
		ShutdownTimeout: 15 * time.Second,
		AppOrigin:       os.Getenv("APP_ORIGIN"),
		MailDriver:      getenv("MAIL_DRIVER", "log"),
	}
	if c.DatabaseDSN == "" {
		return Config{}, fmt.Errorf("DATABASE_DSN is required")
	}
	u, err := url.Parse(c.AppOrigin)
	if c.AppOrigin == "" || err != nil || u.Scheme == "" || u.Host == "" || u.Path != "" {
		return Config{}, fmt.Errorf("APP_ORIGIN must be an origin like https://example.com, got %q", c.AppOrigin)
	}
	// Only local development may run without Secure cookies.
	c.CookieSecure = c.Env != "local" || os.Getenv("COOKIE_SECURE") != "false"
	if c.Env != "local" && u.Scheme != "https" {
		return Config{}, fmt.Errorf("APP_ORIGIN must be https outside local")
	}
	switch c.MailDriver {
	case "log":
		if c.IsProduction() {
			return Config{}, fmt.Errorf("MAIL_DRIVER=log is not allowed in production: it writes sign-in links to logs")
		}
	default:
		return Config{}, fmt.Errorf("unknown MAIL_DRIVER %q", c.MailDriver)
	}
	return c, nil
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

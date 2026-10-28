// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Env             string // local | staging | production
	HTTPAddr        string
	DatabaseDSN     string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	c := Config{
		Env:             getenv("APP_ENV", "local"),
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabaseDSN:     os.Getenv("DATABASE_DSN"),
		ShutdownTimeout: 15 * time.Second,
	}
	if c.DatabaseDSN == "" {
		return Config{}, fmt.Errorf("DATABASE_DSN is required")
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

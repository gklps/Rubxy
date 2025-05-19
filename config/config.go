package config

import "time"

type Config struct {
	Port          string
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func Load() *Config {
	return &Config{
		Port:          ":8080",
		AccessSecret:  "your-access-secret",
		RefreshSecret: "your-refresh-secret",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
	}
}

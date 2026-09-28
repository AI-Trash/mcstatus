package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Host                 string
	Port                 int
	CacheTTL             time.Duration
	DefaultTimeout       time.Duration
	MaxTimeout           time.Duration
	MojangBlockedRefresh time.Duration
}

func Load() *Config {
	cfg := &Config{
		Host:                 getEnv("HOST", "0.0.0.0"),
		Port:                 getEnvInt("PORT", 3001),
		CacheTTL:             getEnvDuration("CACHE_TTL", 60*time.Second),
		DefaultTimeout:       getEnvDuration("DEFAULT_TIMEOUT", 5*time.Second),
		MaxTimeout:           getEnvDuration("MAX_TIMEOUT", 15*time.Second),
		MojangBlockedRefresh: getEnvDuration("MOJANG_BLOCKED_REFRESH", 1*time.Hour),
	}
	return cfg
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		if s, err := strconv.Atoi(val); err == nil {
			return time.Duration(s) * time.Second
		}
	}
	return defaultVal
}

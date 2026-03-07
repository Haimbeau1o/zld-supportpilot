package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppName        string
	AppEnv         string
	HTTPAddr       string
	AuthSigningKey string
	AuthTokenTTL   time.Duration
}

func Load() Config {
	return Config{
		AppName:  getenv("APP_NAME", "zld-supportpilot"),
		AppEnv:   getenv("APP_ENV", "dev"),
		HTTPAddr: getenv("HTTP_ADDR", ":8080"),
		// 当前默认签名密钥只用于本地开发和学习验证，真实环境必须通过环境变量覆盖。
		AuthSigningKey: getenv("AUTH_SIGNING_KEY", "dev-only-signing-key"),
		AuthTokenTTL:   getenvDurationSeconds("AUTH_TOKEN_TTL_SECONDS", time.Hour),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getenvDurationSeconds(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}

	return time.Duration(seconds) * time.Second
}

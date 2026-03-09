package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName              string
	AppEnv               string
	HTTPAddr             string
	AuthSigningKey       string
	AuthTokenTTL         time.Duration
	HTTPRateLimitEnabled bool
	HTTPRateLimitRPS     float64
	HTTPRateLimitBurst   int
	PersistenceMode      string
	PostgresDSN          string
	DocumentStorageMode  string
	DocumentStorageRoot  string
}

func Load() Config {
	return Config{
		AppName:  getenv("APP_NAME", "zld-supportpilot"),
		AppEnv:   getenv("APP_ENV", "dev"),
		HTTPAddr: getenv("HTTP_ADDR", ":8080"),
		// 当前默认签名密钥只用于本地开发和学习验证，真实环境必须通过环境变量覆盖。
		AuthSigningKey: getenv("AUTH_SIGNING_KEY", "dev-only-signing-key"),
		AuthTokenTTL:   getenvDurationSeconds("AUTH_TOKEN_TTL_SECONDS", time.Hour),
		// 当前项目仍是单进程演示系统，因此默认开启轻量级应用层限流即可；后续多实例部署时再升级为共享存储实现。
		HTTPRateLimitEnabled: getenvBool("HTTP_RATE_LIMIT_ENABLED", true),
		HTTPRateLimitRPS:     getenvFloat64("HTTP_RATE_LIMIT_RPS", 5),
		HTTPRateLimitBurst:   getenvInt("HTTP_RATE_LIMIT_BURST", 10),
		// 持久化模式默认仍走 memory，便于本地无依赖启动；切到 postgres 时由启动阶段强校验关键配置。
		PersistenceMode:     getenvOneOf("APP_PERSISTENCE_MODE", "memory", "memory", "postgres"),
		PostgresDSN:         strings.TrimSpace(os.Getenv("POSTGRES_DSN")),
		DocumentStorageMode: getenvOneOf("DOCUMENT_STORAGE_MODE", "memory", "memory", "filesystem"),
		// 即使当前默认对象存储模式仍是 memory，也预留本地目录配置，便于后续切换到文件系统存储。
		DocumentStorageRoot: getenv("DOCUMENT_STORAGE_ROOT", "data/documents"),
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func getenvOneOf(key string, fallback string, allowedValues ...string) string {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}

	for _, candidate := range allowedValues {
		if value == candidate {
			return value
		}
	}

	return fallback
}

func getenvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func getenvFloat64(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
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

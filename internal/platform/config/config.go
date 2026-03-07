package config

import "os"

type Config struct {
	AppName  string
	AppEnv   string
	HTTPAddr string
}

func Load() Config {
	return Config{
		AppName:  getenv("APP_NAME", "zld-supportpilot"),
		AppEnv:   getenv("APP_ENV", "dev"),
		HTTPAddr: getenv("HTTP_ADDR", ":8080"),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port     int
	LogLevel string
}

func Default() *Config {
	return &Config{Port: 8080, LogLevel: "info"}
}

func (c *Config) LoadFromEnv() error {
	if v := os.Getenv("PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid PORT value '%s': %v", v, err)
		}
		c.Port = port
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	return nil
}

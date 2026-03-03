package config

import (
	"os"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected log level 'info', got '%s'", cfg.LogLevel)
	}
}

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envPort  string
		envLog   string
		wantPort int
		wantLog  string
		wantErr  bool
	}{
		{"valid port", "9090", "debug", 9090, "debug", false},
		{"invalid port", "abc", "", 8080, "info", true},
		{"empty env", "", "", 8080, "info", false},
		{"log only", "", "error", 8080, "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("PORT", tt.envPort)
			os.Setenv("LOG_LEVEL", tt.envLog)
			defer os.Unsetenv("PORT")
			defer os.Unsetenv("LOG_LEVEL")

			cfg := Default()
			err := cfg.LoadFromEnv()

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
			if cfg.Port != tt.wantPort {
				t.Errorf("Expected port %d, got %d", tt.wantPort, cfg.Port)
			}
			if cfg.LogLevel != tt.wantLog {
				t.Errorf("Expected log level '%s', got '%s'", tt.wantLog, cfg.LogLevel)
			}
		})
	}
}

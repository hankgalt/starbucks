package config

import (
	"log"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestGetConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config, _ := GetAppConfig(logger)
	log.Printf("Config: %v\n", config)
}

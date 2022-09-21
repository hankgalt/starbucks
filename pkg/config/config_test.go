package config

import (
	"log"
	"testing"
)

func TestGetConfig(t *testing.T) {
	config, _ := GetConfig()
	log.Printf("Config: %v\n", config)
}

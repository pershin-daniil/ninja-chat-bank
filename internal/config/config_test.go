package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pershin-daniil/ninja-chat-bank/internal/config"
)

func TestGlobalConfig_IsProduction(t *testing.T) {
	assert.True(t, config.Config{Global: config.GlobalConfig{Env: "prod"}}.IsProduction())
	assert.False(t, config.Config{Global: config.GlobalConfig{Env: "neprod"}}.IsProduction())
}

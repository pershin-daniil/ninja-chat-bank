package config

import (
	"fmt"

	"github.com/BurntSushi/toml"

	"github.com/pershin-daniil/ninja-chat-bank/internal/validator"
)

func ParseAndValidate(filename string) (Config, error) {
	var config Config
	if _, err := toml.DecodeFile(filename, &config); err != nil {
		return Config{}, fmt.Errorf("failed to decode file %s: %w", filename, err)
	}
	if err := validator.Validator.Struct(config); err != nil {
		return Config{}, fmt.Errorf("failed to validate config: %w", err)
	}
	return config, nil
}

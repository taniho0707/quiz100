package models

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Event          EventConfig          `toml:"event"`
	TeamSeparation TeamSeparationConfig `toml:"team_separation"`
	Questions      []Question           `toml:"questions"`
}

type EventConfig struct {
	Title    string `toml:"title"`
	TeamMode bool   `toml:"team_mode"`
	TeamSize int    `toml:"team_size"`
	QrCode   string `toml:"qrcode"`
}

type TeamSeparationConfig struct {
	AvoidGroups []string `toml:"avoid_groups"`
}

type Question struct {
	Type    string   `toml:"type"`
	Text    string   `toml:"text"`
	Image   string   `toml:"image"`
	Choices []string `toml:"choices"`
	Correct int      `toml:"correct"`
}

func LoadConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %v", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if c.Event.Title == "" {
		return errors.New("event title is required")
	}

	if c.Event.TeamMode && c.Event.TeamSize <= 0 {
		return errors.New("team_size must be greater than 0 when team_mode is enabled")
	}

	if len(c.Questions) == 0 {
		return errors.New("at least one question is required")
	}

	for i, q := range c.Questions {
		if err := q.Validate(); err != nil {
			return fmt.Errorf("question %d: %v", i+1, err)
		}
	}

	return nil
}

func (q *Question) Validate() error {
	if q.Text == "" {
		return errors.New("question text is required")
	}

	if q.Type != "text" && q.Type != "image" {
		return errors.New("question type must be 'text' or 'image'")
	}

	if q.Type == "image" && q.Image == "" {
		return errors.New("image path is required for image type questions")
	}

	if len(q.Choices) < 2 {
		return errors.New("at least 2 choices are required")
	}

	if q.Correct < 0 || q.Correct >= len(q.Choices) {
		return errors.New("correct answer index is out of range")
	}

	if q.Image != "" {
		imagePath := filepath.Join("static", "images", q.Image)
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			return fmt.Errorf("image file not found: %s", imagePath)
		}
	}

	return nil
}

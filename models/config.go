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
	TeamNames      []string             // Loaded from team.toml
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

type TeamConfig struct {
	TeamNames []string `toml:"team_names"`
}

type Question struct {
	Type    string   `toml:"type" json:"type"`
	Text    string   `toml:"text" json:"text"`
	Image   string   `toml:"image" json:"image"`
	Choices []string `toml:"choices" json:"choices"`
	Correct int      `toml:"correct" json:"correct"`
	Point   int      `toml:"point" json:"point"`
}

func LoadConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %v", err)
	}

	// Load team names from team.toml
	teamConfigPath := filepath.Join(filepath.Dir(configPath), "team.toml")
	teamNames, err := LoadTeamNames(teamConfigPath)
	if err != nil {
		// Team names are optional, use default if not found
		teamNames = []string{}
	}
	config.TeamNames = teamNames

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return &config, nil
}

// LoadTeamNames loads team names from team.toml
func LoadTeamNames(teamConfigPath string) ([]string, error) {
	if _, err := os.Stat(teamConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("team config file not found: %s", teamConfigPath)
	}

	var teamConfig TeamConfig
	if _, err := toml.DecodeFile(teamConfigPath, &teamConfig); err != nil {
		return nil, fmt.Errorf("failed to decode team config file: %v", err)
	}

	if len(teamConfig.TeamNames) == 0 {
		return nil, errors.New("team_names is empty in team config")
	}

	return teamConfig.TeamNames, nil
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

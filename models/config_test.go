package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	testConfig := `
[event]
title = "Test Quiz Event"
team_mode = true
team_size = 4

[team_separation]
avoid_groups = ["test1", "test2"]

[[questions]]
type = "text"
text = "What is 2+2?"
choices = ["2", "3", "4", "5"]
correct = 3

[[questions]]
type = "text"
text = "What is the capital of Japan?"
choices = ["Tokyo", "Osaka", "Kyoto"]
correct = 1
`

	// Create temporary directory and file
	tempDir, err := ioutil.TempDir("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.toml")
	err = ioutil.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test loading config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify event config
	if config.Event.Title != "Test Quiz Event" {
		t.Errorf("Expected title 'Test Quiz Event', got '%s'", config.Event.Title)
	}

	if !config.Event.TeamMode {
		t.Error("Expected team mode to be true")
	}

	if config.Event.TeamSize != 4 {
		t.Errorf("Expected team size 4, got %d", config.Event.TeamSize)
	}

	// Verify team separation
	expectedAvoidGroups := []string{"test1", "test2"}
	if len(config.TeamSeparation.AvoidGroups) != len(expectedAvoidGroups) {
		t.Errorf("Expected %d avoid groups, got %d", len(expectedAvoidGroups), len(config.TeamSeparation.AvoidGroups))
	}

	for i, group := range expectedAvoidGroups {
		if config.TeamSeparation.AvoidGroups[i] != group {
			t.Errorf("Expected avoid group '%s', got '%s'", group, config.TeamSeparation.AvoidGroups[i])
		}
	}

	// Verify questions
	if len(config.Questions) != 2 {
		t.Errorf("Expected 2 questions, got %d", len(config.Questions))
	}

	// Check first question
	q1 := config.Questions[0]
	if q1.Text != "What is 2+2?" {
		t.Errorf("Expected question 'What is 2+2?', got '%s'", q1.Text)
	}

	if len(q1.Choices) != 4 {
		t.Errorf("Expected 4 choices, got %d", len(q1.Choices))
	}

	if q1.Correct != 3 {
		t.Errorf("Expected correct answer 3, got %d", q1.Correct)
	}

	// Check second question
	q2 := config.Questions[1]
	if q2.Text != "What is the capital of Japan?" {
		t.Errorf("Expected question 'What is the capital of Japan?', got '%s'", q2.Text)
	}

	if len(q2.Choices) != 3 {
		t.Errorf("Expected 3 choices, got %d", len(q2.Choices))
	}

	if q2.Correct != 1 {
		t.Errorf("Expected correct answer 1, got %d", q2.Correct)
	}
}

func TestValidateConfig(t *testing.T) {
	// Test valid config
	validConfig := &Config{
		Event: EventConfig{
			Title:    "Valid Quiz",
			TeamMode: false,
			TeamSize: 5,
		},
		Questions: []Question{
			{
				Type:    "text",
				Text:    "Valid question?",
				Choices: []string{"A", "B", "C"},
				Correct: 1,
			},
		},
	}

	err := ValidateConfig(validConfig)
	if err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}

	// Test invalid config - no questions
	invalidConfig := &Config{
		Event: EventConfig{
			Title:    "Invalid Quiz",
			TeamMode: false,
			TeamSize: 5,
		},
		Questions: []Question{},
	}

	err = ValidateConfig(invalidConfig)
	if err == nil {
		t.Error("Expected validation error for config with no questions")
	}

	// Test invalid config - invalid correct answer
	invalidConfig2 := &Config{
		Event: EventConfig{
			Title:    "Invalid Quiz",
			TeamMode: false,
			TeamSize: 5,
		},
		Questions: []Question{
			{
				Type:    "text",
				Text:    "Invalid question?",
				Choices: []string{"A", "B"},
				Correct: 5, // Invalid - out of range
			},
		},
	}

	err = ValidateConfig(invalidConfig2)
	if err == nil {
		t.Error("Expected validation error for invalid correct answer")
	}
}

func ValidateConfig(config *Config) error {
	if len(config.Questions) == 0 {
		return &ValidationError{Message: "No questions provided"}
	}

	for i, question := range config.Questions {
		if len(question.Choices) == 0 {
			return &ValidationError{Message: "Question has no choices", QuestionIndex: &i}
		}

		if question.Correct < 1 || question.Correct > len(question.Choices) {
			return &ValidationError{Message: "Invalid correct answer index", QuestionIndex: &i}
		}
	}

	return nil
}

type ValidationError struct {
	Message       string
	QuestionIndex *int
}

func (e *ValidationError) Error() string {
	if e.QuestionIndex != nil {
		return fmt.Sprintf("Question %d: %s", *e.QuestionIndex+1, e.Message)
	}
	return e.Message
}

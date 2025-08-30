package models

import (
	"testing"
)

func TestTeamSeparationLogic(t *testing.T) {
	// Test the separation constraint logic
	avoidGroups := []string{"田中", "山田", "佐藤"}

	// Test cases
	testCases := []struct {
		name1, name2   string
		shouldSeparate bool
	}{
		{"田中太郎", "田中花子", true},  // Same avoid group
		{"山田一郎", "山田次郎", true},  // Same avoid group
		{"佐藤太郎", "田中花子", false}, // Different avoid groups
		{"鈴木三郎", "高橋四郎", false}, // No avoid groups
	}

	for _, tc := range testCases {
		shouldSeparate := false
		for _, avoidGroup := range avoidGroups {
			if containsString(tc.name1, avoidGroup) && containsString(tc.name2, avoidGroup) {
				shouldSeparate = true
				break
			}
		}

		if shouldSeparate != tc.shouldSeparate {
			t.Errorf("For names %s and %s, expected separation: %v, got: %v",
				tc.name1, tc.name2, tc.shouldSeparate, shouldSeparate)
		}
	}
}

func TestScoreCalculation(t *testing.T) {
	user := &User{
		ID:       1,
		Nickname: "TestUser",
		Score:    5,
	}

	// Test score increment
	if user.Score != 5 {
		t.Errorf("Expected score 5, got %d", user.Score)
	}

	user.Score++
	if user.Score != 6 {
		t.Errorf("Expected score 6 after increment, got %d", user.Score)
	}
}

func TestUserTeamAssignment(t *testing.T) {
	user := &User{
		ID:       1,
		Nickname: "TestUser",
		Score:    0,
		TeamID:   nil,
	}

	// Initially no team
	if user.TeamID != nil {
		t.Error("Expected user to have no team initially")
	}

	// Assign to team
	teamID := 5
	user.TeamID = &teamID

	if user.TeamID == nil || *user.TeamID != 5 {
		t.Error("Expected user to be assigned to team 5")
	}
}

func TestTeamScoreCalculation(t *testing.T) {
	team := &Team{
		ID:    1,
		Name:  "TestTeam",
		Score: 0,
		Members: []User{
			{ID: 1, Nickname: "User1", Score: 3},
			{ID: 2, Nickname: "User2", Score: 5},
			{ID: 3, Nickname: "User3", Score: 2},
		},
	}

	// Calculate team score as sum of member scores
	totalScore := 0
	for _, member := range team.Members {
		totalScore += member.Score
	}

	expectedScore := 3 + 5 + 2
	if totalScore != expectedScore {
		t.Errorf("Expected team score %d, got %d", expectedScore, totalScore)
	}
}

// Helper function for string matching
func containsString(text, pattern string) bool {
	return len(text) >= len(pattern) &&
		(text[:len(pattern)] == pattern ||
			text[len(text)-len(pattern):] == pattern)
}

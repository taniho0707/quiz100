package models

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type TeamAssignmentService struct {
	userRepo *UserRepository
	teamRepo *TeamRepository
	config   *Config
}

func NewTeamAssignmentService(userRepo *UserRepository, teamRepo *TeamRepository, config *Config) *TeamAssignmentService {
	return &TeamAssignmentService{
		userRepo: userRepo,
		teamRepo: teamRepo,
		config:   config,
	}
}

func (s *TeamAssignmentService) CreateTeamsAndAssignUsers() ([]Team, error) {
	// Get all users without team assignment
	users, err := s.userRepo.GetUsersWithoutTeam()
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %v", err)
	}

	if len(users) == 0 {
		return []Team{}, nil
	}

	// Clear existing teams if any
	err = s.teamRepo.DeleteAllTeams()
	if err != nil {
		return nil, fmt.Errorf("failed to clear existing teams: %v", err)
	}

	// Shuffle users for random assignment
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(users), func(i, j int) {
		users[i], users[j] = users[j], users[i]
	})

	// Apply separation constraints
	if len(s.config.TeamSeparation.AvoidGroups) > 0 {
		users = s.applySeparationConstraints(users)
	}

	// Calculate number of teams
	teamSize := s.config.Event.TeamSize
	if teamSize <= 0 {
		teamSize = 5 // Default team size
	}

	numTeams := (len(users) + teamSize - 1) / teamSize // Ceiling division
	if numTeams == 0 {
		numTeams = 1
	}

	// Create teams
	var teams []Team
	for i := 0; i < numTeams; i++ {
		teamName := fmt.Sprintf("チーム%d", i+1)
		team, err := s.teamRepo.CreateTeam(teamName)
		if err != nil {
			return nil, fmt.Errorf("failed to create team %d: %v", i+1, err)
		}
		teams = append(teams, *team)
	}

	// Assign users to teams
	for i, user := range users {
		teamIndex := i % len(teams)
		err = s.userRepo.AssignUserToTeam(user.ID, teams[teamIndex].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to assign user %d to team: %v", user.ID, err)
		}
	}

	// Get teams with members for return
	teamsWithMembers, err := s.teamRepo.GetAllTeamsWithMembers()
	if err != nil {
		return teams, nil // Return teams without member details if query fails
	}

	return teamsWithMembers, nil
}

func (s *TeamAssignmentService) applySeparationConstraints(users []User) []User {
	// Group users by their similarity to avoid groups
	userGroups := make(map[string][]User)
	ungroupedUsers := []User{}

	for _, user := range users {
		matched := false
		for _, avoidGroup := range s.config.TeamSeparation.AvoidGroups {
			if s.isNameMatch(user.Nickname, avoidGroup) {
				if userGroups[avoidGroup] == nil {
					userGroups[avoidGroup] = []User{}
				}
				userGroups[avoidGroup] = append(userGroups[avoidGroup], user)
				matched = true
				break
			}
		}
		if !matched {
			ungroupedUsers = append(ungroupedUsers, user)
		}
	}

	// Distribute grouped users evenly
	result := []User{}
	maxGroupSize := 0
	for _, group := range userGroups {
		if len(group) > maxGroupSize {
			maxGroupSize = len(group)
		}
	}

	// Interleave users from different groups to spread them across teams
	for i := 0; i < maxGroupSize; i++ {
		for groupName, group := range userGroups {
			if i < len(group) {
				result = append(result, group[i])
			}
			_ = groupName // Avoid unused variable warning
		}
	}

	// Add ungrouped users at the end
	result = append(result, ungroupedUsers...)

	return result
}

func (s *TeamAssignmentService) isNameMatch(nickname, avoidGroup string) bool {
	// Convert to lowercase for case-insensitive matching
	nicknameLower := strings.ToLower(nickname)
	avoidGroupLower := strings.ToLower(avoidGroup)

	// Check if nickname contains the avoid group string (partial match)
	return strings.Contains(nicknameLower, avoidGroupLower)
}

func (s *TeamAssignmentService) CalculateTeamScores() ([]Team, error) {
	teams, err := s.teamRepo.GetAllTeamsWithMembers()
	if err != nil {
		return nil, err
	}

	for _, team := range teams {
		totalScore := 0
		for _, member := range team.Members {
			totalScore += member.Score
		}

		err = s.teamRepo.UpdateTeamScore(team.ID, totalScore)
		if err != nil {
			return nil, fmt.Errorf("failed to update team %d score: %v", team.ID, err)
		}
	}

	// Return updated teams with scores
	return s.teamRepo.GetAllTeamsWithMembers()
}

// AssignUserToAvailableTeam assigns a new user to the least filled team
// If all teams are at capacity, it creates a new team
func (s *TeamAssignmentService) AssignUserToAvailableTeam(userID int, nickname string) (*Team, error) {
	// Get all existing teams with their members
	teams, err := s.teamRepo.GetAllTeamsWithMembers()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing teams: %v", err)
	}

	if len(teams) == 0 {
		// No teams exist yet, return without assignment
		return nil, nil
	}

	teamSize := s.config.Event.TeamSize
	if teamSize <= 0 {
		teamSize = 5 // Default team size
	}

	// Find the team with the least members that isn't full
	var targetTeam *Team
	minMembers := teamSize + 1 // Start with max+1 so any team with space will be less

	for i := range teams {
		memberCount := len(teams[i].Members)
		if memberCount < teamSize && memberCount < minMembers {
			targetTeam = &teams[i]
			minMembers = memberCount
		}
	}

	// Check for separation constraints
	if len(s.config.TeamSeparation.AvoidGroups) > 0 && targetTeam != nil {
		// Check if any avoid group member is already in the target team
		for _, avoidGroup := range s.config.TeamSeparation.AvoidGroups {
			if s.isNameMatch(nickname, avoidGroup) {
				// This user matches an avoid group, check if any team member also matches
				for _, member := range targetTeam.Members {
					if s.isNameMatch(member.Nickname, avoidGroup) {
						// Conflict found, try to find another team
						targetTeam = nil
						break
					}
				}
				if targetTeam == nil {
					break
				}
			}
		}

		// If target team has conflict, try to find another team
		if targetTeam == nil {
			for i := range teams {
				if len(teams[i].Members) >= teamSize {
					continue // Team is full
				}

				hasConflict := false
				for _, avoidGroup := range s.config.TeamSeparation.AvoidGroups {
					if s.isNameMatch(nickname, avoidGroup) {
						for _, member := range teams[i].Members {
							if s.isNameMatch(member.Nickname, avoidGroup) {
								hasConflict = true
								break
							}
						}
						if hasConflict {
							break
						}
					}
				}

				if !hasConflict {
					targetTeam = &teams[i]
					break
				}
			}
		}
	}

	// If no team available or all are full, create a new team
	if targetTeam == nil {
		teamName := fmt.Sprintf("チーム%d", len(teams)+1)
		newTeam, err := s.teamRepo.CreateTeam(teamName)
		if err != nil {
			return nil, fmt.Errorf("failed to create new team: %v", err)
		}
		targetTeam = newTeam
	}

	// Assign user to the target team
	err = s.userRepo.AssignUserToTeam(userID, targetTeam.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to assign user to team: %v", err)
	}

	// Return the updated team with members
	return s.teamRepo.GetTeamWithMembers(targetTeam.ID)
}

package models

import (
	"database/sql"
	"fmt"
	"time"
)

type User struct {
	ID        int       `json:"id" db:"id"`
	SessionID string    `json:"session_id" db:"session_id"`
	Nickname  string    `json:"nickname" db:"nickname"`
	TeamID    *int      `json:"team_id" db:"team_id"`
	Score     int       `json:"score" db:"score"`
	Connected bool      `json:"connected" db:"connected"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Team struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Score     int       `json:"score" db:"score"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Members   []User    `json:"members,omitempty"`
}

type Event struct {
	ID              int       `json:"id" db:"id"`
	Title           string    `json:"title" db:"title"`
	Status          string    `json:"status" db:"status"`
	CurrentQuestion int       `json:"current_question" db:"current_question"`
	TeamMode        bool      `json:"team_mode" db:"team_mode"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type Answer struct {
	ID             int       `json:"id" db:"id"`
	UserID         int       `json:"user_id" db:"user_id"`
	QuestionNumber int       `json:"question_number" db:"question_number"`
	AnswerIndex    int       `json:"answer_index" db:"answer_index"`
	IsCorrect      bool      `json:"is_correct" db:"is_correct"`
	AnswerTime     time.Time `json:"answer_time" db:"answer_time"`
}

type EmojiReaction struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Emoji     string    `json:"emoji" db:"emoji"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type UserRepository struct {
	db *sql.DB
}

type TeamRepository struct {
	db *sql.DB
}

type EventRepository struct {
	db *sql.DB
}

type AnswerRepository struct {
	db *sql.DB
}

type EmojiReactionRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

func NewAnswerRepository(db *sql.DB) *AnswerRepository {
	return &AnswerRepository{db: db}
}

func NewEmojiReactionRepository(db *sql.DB) *EmojiReactionRepository {
	return &EmojiReactionRepository{db: db}
}

func (r *UserRepository) CreateUser(sessionID, nickname string) (*User, error) {
	query := `
		INSERT INTO users (session_id, nickname, created_at, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	result, err := r.db.Exec(query, sessionID, nickname)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetUserByID(int(id))
}

func (r *UserRepository) GetUserBySessionID(sessionID string) (*User, error) {
	user := &User{}
	query := `SELECT id, session_id, nickname, team_id, score, connected, created_at, updated_at FROM users WHERE session_id = ?`

	err := r.db.QueryRow(query, sessionID).Scan(
		&user.ID, &user.SessionID, &user.Nickname, &user.TeamID,
		&user.Score, &user.Connected, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return user, err
}

func (r *UserRepository) GetUserByID(id int) (*User, error) {
	user := &User{}
	query := `SELECT id, session_id, nickname, team_id, score, connected, created_at, updated_at FROM users WHERE id = ?`

	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.SessionID, &user.Nickname, &user.TeamID,
		&user.Score, &user.Connected, &user.CreatedAt, &user.UpdatedAt,
	)

	return user, err
}

func (r *UserRepository) GetAllUsers() ([]User, error) {
	query := `SELECT id, session_id, nickname, team_id, score, connected, created_at, updated_at FROM users ORDER BY score DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID, &user.SessionID, &user.Nickname, &user.TeamID,
			&user.Score, &user.Connected, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) UpdateUserConnection(sessionID string, connected bool) error {
	query := `UPDATE users SET connected = ?, updated_at = CURRENT_TIMESTAMP WHERE session_id = ?`
	_, err := r.db.Exec(query, connected, sessionID)
	return err
}

func (r *UserRepository) UpdateUserScore(userID int, score int) error {
	query := `UPDATE users SET score = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, score, userID)
	return err
}

func (r *EventRepository) CreateEvent(title string, teamMode bool, teamSize int, qrcode string) (*Event, error) {
	query := `
		INSERT INTO events (title, status, team_mode, team_size, qrcode, created_at, updated_at)
		VALUES (?, 'waiting', ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	result, err := r.db.Exec(query, title, teamMode, teamSize, qrcode)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetEvent(int(id))
}

func (r *EventRepository) GetCurrentEvent() (*Event, error) {
	event := &Event{}
	query := `SELECT id, title, status, current_question, team_mode, created_at, updated_at FROM events ORDER BY created_at DESC LIMIT 1`

	err := r.db.QueryRow(query).Scan(
		&event.ID, &event.Title, &event.Status, &event.CurrentQuestion,
		&event.TeamMode, &event.CreatedAt, &event.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return event, err
}

func (r *EventRepository) GetEvent(id int) (*Event, error) {
	event := &Event{}
	query := `SELECT id, title, status, current_question, team_mode, created_at, updated_at FROM events WHERE id = ?`

	err := r.db.QueryRow(query, id).Scan(
		&event.ID, &event.Title, &event.Status, &event.CurrentQuestion,
		&event.TeamMode, &event.CreatedAt, &event.UpdatedAt,
	)

	return event, err
}

func (r *EventRepository) UpdateEventStatus(id int, status string) error {
	query := `UPDATE events SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, status, id)
	return err
}

func (r *EventRepository) UpdateCurrentQuestion(id int, questionNumber int) error {
	query := `UPDATE events SET current_question = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, questionNumber, id)
	return err
}

func (r *AnswerRepository) CreateAnswer(userID, questionNumber, answerIndex int, isCorrect bool) error {
	query := `
		INSERT INTO answers (user_id, question_number, answer_index, is_correct, answer_time)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err := r.db.Exec(query, userID, questionNumber, answerIndex, isCorrect)
	return err
}

func (r *AnswerRepository) GetAnswerByUserAndQuestion(userID, questionNumber int) (*Answer, error) {
	answer := &Answer{}
	query := `SELECT id, user_id, question_number, answer_index, is_correct, answer_time FROM answers WHERE user_id = ? AND question_number = ?`

	err := r.db.QueryRow(query, userID, questionNumber).Scan(
		&answer.ID, &answer.UserID, &answer.QuestionNumber,
		&answer.AnswerIndex, &answer.IsCorrect, &answer.AnswerTime,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return answer, err
}

func (r *EmojiReactionRepository) CreateReaction(userID int, emoji string) error {
	query := `INSERT INTO emoji_reactions (user_id, emoji, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)`
	_, err := r.db.Exec(query, userID, emoji)
	return err
}

func (r *EmojiReactionRepository) GetRecentReactions(minutes int) ([]EmojiReaction, error) {
	query := `
		SELECT id, user_id, emoji, created_at 
		FROM emoji_reactions 
		WHERE created_at >= datetime('now', '-' || ? || ' minutes')
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, minutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []EmojiReaction
	for rows.Next() {
		var reaction EmojiReaction
		err := rows.Scan(&reaction.ID, &reaction.UserID, &reaction.Emoji, &reaction.CreatedAt)
		if err != nil {
			return nil, err
		}
		reactions = append(reactions, reaction)
	}

	return reactions, nil
}

// Team Repository Methods
func (r *TeamRepository) CreateTeam(name string) (*Team, error) {
	query := `
		INSERT INTO teams (name, created_at, updated_at)
		VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	result, err := r.db.Exec(query, name)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetTeamByID(int(id))
}

func (r *TeamRepository) GetTeamByID(id int) (*Team, error) {
	team := &Team{}
	query := `SELECT id, name, score, created_at, updated_at FROM teams WHERE id = ?`

	err := r.db.QueryRow(query, id).Scan(
		&team.ID, &team.Name, &team.Score, &team.CreatedAt, &team.UpdatedAt,
	)

	return team, err
}

func (r *TeamRepository) GetAllTeams() ([]Team, error) {
	query := `SELECT id, name, score, created_at, updated_at FROM teams ORDER BY score DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var team Team
		err := rows.Scan(
			&team.ID, &team.Name, &team.Score, &team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}

	return teams, nil
}

func (r *TeamRepository) GetTeamWithMembers(id int) (*Team, error) {
	team, err := r.GetTeamByID(id)
	if err != nil {
		return nil, err
	}

	// Get team members
	query := `SELECT id, session_id, nickname, team_id, score, connected, created_at, updated_at FROM users WHERE team_id = ?`
	rows, err := r.db.Query(query, id)
	if err != nil {
		return team, nil // Return team without members if query fails
	}
	defer rows.Close()

	var members []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID, &user.SessionID, &user.Nickname, &user.TeamID,
			&user.Score, &user.Connected, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			continue
		}
		members = append(members, user)
	}

	team.Members = members
	return team, nil
}

func (r *TeamRepository) GetAllTeamsWithMembers() ([]Team, error) {
	teams, err := r.GetAllTeams()
	if err != nil {
		return nil, err
	}

	for i, team := range teams {
		teamWithMembers, err := r.GetTeamWithMembers(team.ID)
		if err == nil && teamWithMembers != nil {
			teams[i].Members = teamWithMembers.Members
		}
	}

	return teams, nil
}

func (r *TeamRepository) UpdateTeamScore(id int, score int) error {
	query := `UPDATE teams SET score = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, score, id)
	return err
}

func (r *TeamRepository) DeleteAllTeams() error {
	// First remove team associations from users
	_, err := r.db.Exec("UPDATE users SET team_id = NULL WHERE team_id IS NOT NULL")
	if err != nil {
		return err
	}

	// Then delete all teams
	_, err = r.db.Exec("DELETE FROM teams")
	return err
}

// User Repository Methods for Team Assignment
func (r *UserRepository) AssignUserToTeam(userID int, teamID int) error {
	query := `UPDATE users SET team_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, teamID, userID)
	return err
}

func (r *UserRepository) GetUsersWithoutTeam() ([]User, error) {
	query := `SELECT id, session_id, nickname, team_id, score, connected, created_at, updated_at FROM users WHERE team_id IS NULL AND connected = 1`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID, &user.SessionID, &user.Nickname, &user.TeamID,
			&user.Score, &user.Connected, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) DeleteUserBySessionID(sessionID string) error {
	query := `DELETE FROM users WHERE session_id = ?`
	_, err := r.db.Exec(query, sessionID)
	return err
}

func (r *AnswerRepository) DeleteAnswersByUserID(userID int) error {
	query := `DELETE FROM answers WHERE user_id = ?`
	_, err := r.db.Exec(query, userID)
	return err
}

func (r *EmojiReactionRepository) DeleteReactionsByUserID(userID int) error {
	query := `DELETE FROM emoji_reactions WHERE user_id = ?`
	_, err := r.db.Exec(query, userID)
	return err
}

// Event State Management System
// Note: EventState constants are now defined in constants.go

type EventStateManager struct {
	currentState     EventState
	currentQuestion  int
	totalQuestions   int
	teamMode         bool
	validTransitions map[EventState][]EventState
}

func NewEventStateManager(teamMode bool, totalQuestions int) *EventStateManager {
	esm := &EventStateManager{
		currentState:    StateStarted,
		currentQuestion: 0,
		totalQuestions:  totalQuestions,
		teamMode:        teamMode,
	}

	esm.initValidTransitions()
	return esm
}

func (esm *EventStateManager) initValidTransitions() {
	esm.validTransitions = map[EventState][]EventState{
		StateStarted:         {StateTitleDisplay},
		StateTitleDisplay:    {StateTeamAssignment, StateQuestionActive},
		StateTeamAssignment:  {StateQuestionActive},
		StateQuestionActive:  {StateCountdownActive},
		StateCountdownActive: {StateAnswerStats},
		StateAnswerStats:     {StateAnswerReveal},
		StateAnswerReveal:    {StateQuestionActive, StateResults},
		StateResults:         {StateCelebration},
		StateCelebration:     {StateFinished},
		StateFinished:        {},
	}

	// チーム戦でない場合はチーム分け状態をスキップ
	if !esm.teamMode {
		esm.validTransitions[StateTitleDisplay] = []EventState{StateQuestionActive}
	}
}

func (esm *EventStateManager) GetCurrentState() EventState {
	return esm.currentState
}

func (esm *EventStateManager) GetCurrentQuestion() int {
	return esm.currentQuestion
}

func (esm *EventStateManager) SetCurrentQuestion(questionNumber int) error {
	if questionNumber < 0 || questionNumber > esm.totalQuestions {
		return fmt.Errorf("invalid question number: %d (valid range: 0-%d)", questionNumber, esm.totalQuestions)
	}
	esm.currentQuestion = questionNumber
	return nil
}

func (esm *EventStateManager) CanTransitionTo(targetState EventState) bool {
	validStates, exists := esm.validTransitions[esm.currentState]
	if !exists {
		return false
	}

	for _, validState := range validStates {
		if validState == targetState {
			return true
		}
	}

	return false
}

func (esm *EventStateManager) TransitionTo(targetState EventState) error {
	if !esm.CanTransitionTo(targetState) {
		return fmt.Errorf("invalid transition from %s to %s", esm.currentState, targetState)
	}

	esm.currentState = targetState
	return nil
}

// JumpToState allows jumping to any state without transition validation (for admin use)
func (esm *EventStateManager) JumpToState(targetState EventState) error {
	// Validate that the target state exists using the constants
	if !IsValidState(targetState) {
		return fmt.Errorf("invalid state: %s", targetState)
	}

	esm.currentState = targetState
	return nil
}

func (esm *EventStateManager) NextQuestion() error {
	if esm.currentState != StateAnswerReveal && esm.currentState != StateTeamAssignment {
		return fmt.Errorf("cannot advance question from state %s", esm.currentState)
	}

	if esm.currentQuestion >= esm.totalQuestions {
		// 最後の問題なので結果発表へ
		return esm.TransitionTo(StateResults)
	}

	esm.currentQuestion++
	return esm.TransitionTo(StateQuestionActive)
}

func (esm *EventStateManager) GetAvailableActions() []string {
	switch esm.currentState {
	case StateStarted:
		return []string{"show_title"}
	case StateTitleDisplay:
		if esm.teamMode {
			return []string{"assign_teams"}
		}
		return []string{"next_question"}
	case StateTeamAssignment:
		return []string{"next_question"}
	case StateQuestionActive:
		return []string{"countdown_alert"}
	case StateCountdownActive:
		return []string{"show_answer_stats"}
	case StateAnswerStats:
		return []string{"reveal_answer"}
	case StateAnswerReveal:
		if esm.currentQuestion >= esm.totalQuestions {
			return []string{"show_results"}
		}
		return []string{"next_question"}
	case StateResults:
		return []string{"celebration"}
	case StateCelebration:
		return []string{} // 自動遷移
	case StateFinished:
		return []string{}
	default:
		return []string{}
	}
}

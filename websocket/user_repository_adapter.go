package websocket

import "quiz100/models"

// UserRepositoryAdapter adapts models.UserRepository to PingUserRepository
type UserRepositoryAdapter struct {
	userRepo *models.UserRepository
}

// NewUserRepositoryAdapter creates a new adapter
func NewUserRepositoryAdapter(userRepo *models.UserRepository) *UserRepositoryAdapter {
	return &UserRepositoryAdapter{
		userRepo: userRepo,
	}
}

// GetUserByID implements PingUserRepository interface
func (ura *UserRepositoryAdapter) GetUserByID(id int) (*PingUser, error) {
	user, err := ura.userRepo.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, nil
	}

	return &PingUser{
		ID:       user.ID,
		Nickname: user.Nickname,
	}, nil
}

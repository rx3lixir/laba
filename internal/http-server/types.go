package httpserver

import (
	"time"

	"github.com/google/uuid"
)

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateUserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetAllUsersResponse struct {
	Users      []UserResponse `json:"users"`
	TotalCount int            `json:"total_count"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
}

type DeleteUserResponse struct {
	Message string    `json:"message"`
	ID      uuid.UUID `json:"id"`
}

package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/rx3lixir/laba/internal/db"
	"github.com/rx3lixir/laba/pkg/password"
)

func (s *Server) HandleHello(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "hello world",
		"status":  "success",
	}

	s.log.Info("Recieved", "handler", "HandleHello")

	s.respondJSON(w, http.StatusOK, response)
}

// Handles creating a new user
func (s *Server) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	req := new(CreateUserRequest)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	s.log.Info(
		"recieved request",
		"handler", "HandleAddUser",
		"email", req.Email,
	)

	// Request validation
	if err := validateCreateUserRequest(req); err != nil {
		s.handleError(w, err)

		log.Error(
			"User validation failed",
			"user_email", req.Email,
			"error", err,
		)
		return
	}

	// Password hashing
	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		s.log.Error("Failed to hash password", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to proccess password")
		return
	}

	// Creating new user
	newUser := &db.User{
		Username: req.Username,
		Email:    strings.ToLower(strings.TrimSpace(req.Email)),
		Password: string(hashedPassword),
	}

	// Saving user to database
	if err := s.userStore.CreateUser(r.Context(), newUser); err != nil {
		s.log.Error("Failed to create user", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to add user to database")
		return
	}

	// Building a response
	response := CreateUserResponse{
		ID:        newUser.ID,
		Username:  newUser.Username,
		Email:     newUser.Email,
		CreatedAt: newUser.CreatedAt,
	}

	s.log.Info(
		"User created successfully",
		"user_email", newUser.Email,
		"user_id", newUser.Email,
	)

	s.respondJSON(w, http.StatusCreated, response)
}

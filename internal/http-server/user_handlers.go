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

func (s *Server) HandleAddUser(w http.ResponseWriter, r *http.Request) {
	req := new(CreateUserRequest)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	s.log.Info(
		"recieved request",
		"handler",
		"HandleAddUser",
		"request",
		req,
	)

	// Request validation
	if err := validateCreateUserRequest(req); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())

		log.Error(
			"User is invalid",
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

	// Creating new user instance
	newUser := &db.User{
		Name:     req.Name,
		Email:    strings.ToLower(strings.TrimSpace(req.Email)),
		Password: string(hashedPassword),
	}

	// Saving user to database
	if err := s.userStorer.CreateUser(r.Context(), newUser); err != nil {
		s.log.Error(
			"Failed to create user",
			"user_email",
			newUser.Email,
			"error",
			err,
		)
		s.respondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Building a response
	response := CreateUserResponse{
		ID:        newUser.ID,
		Name:      newUser.Name,
		Email:     newUser.Email,
		CreatedAt: newUser.CreatedAt,
	}

	s.log.Info(
		"User created successfully",
		"user_email",
		newUser.Email,
	)

	s.respondJSON(w, http.StatusCreated, response)
}

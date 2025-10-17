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
		"user_id", newUser.ID,
	)

	s.respondJSON(w, http.StatusCreated, response)
}

// Handles getting user using it's ID
func (s *Server) HandleGetUserByID(w http.ResponseWriter, r *http.Request) {
	req := new(GetUserByIDRequest)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	s.log.Info("Received request",
		"handler", "HandleGetUserByID",
		"email", req.Email,
		"id", req.ID,
	)

	// Getting user from database
	user, err := s.userStore.GetUserByID(r.Context(), req.ID)
	if err != nil {
		s.handleError(w, err)
		return
	}

	// Building a response
	response := GetUserByIDResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Password:  user.Password,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	s.log.Info(
		"User created successfully",
		"user_email", user.Email,
		"user_id", user.ID,
	)

	// Writing a response
	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) HandleGetAllUsers(w http.ResponseWriter, r *http.Request) {
	req := new([]GetAllUsersRequest)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid json format")
		return
	}

	s.log.Info("Recieved request",
		"handler", "HandleGetAllUsers",
	)

	users, err := s.userStore.GetUsers(r.Context(), 10, 0)
	if err != nil {
		s.handleError(w, err)
		return
	}

	response := make([]*GetAllUsersResponse, 0, len(users))

	for _, user := range users {
		response = append(response, user)
	}

	// Writing a response
	s.respondJSON(w, http.StatusOK, response)
}

// Handles getting user by it's email
func (s *Server) HandleGetUserByEmail(w http.ResponseWriter, r *http.Request) {
	req := new(GetUserByEmailRequest)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid json format")
		return
	}

	s.log.Info("Recieved request",
		"handler", "HandleGetUsersByEmail",
		"email", req.Email,
		"username", req.Username,
	)

	// Getting user from database
	user, err := s.userStore.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		s.handleError(w, err)
		return
	}

	// Building a response
	response := GetUserByEmailResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Password:  user.Password,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	// Writing a response
	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
}

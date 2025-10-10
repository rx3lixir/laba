package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rx3lixir/laba/internal/db"
	"github.com/rx3lixir/laba/pkg/password"
)

func (s *Server) HandleHello(w http.ResponseWriter, r *http.Request) error {
	response := map[string]string{
		"message": "hello world",
		"status":  "success",
	}

	s.log.Info("Recieved", "handler", "HandleHello")

	return WriteJSON(w, http.StatusOK, response)
}

func handleHello(w http.ResponseWriter, r *http.Request) error {
	response := map[string]string{
		"message": "hello world",
		"status":  "success",
	}

	return WriteJSON(w, http.StatusOK, response)
}

func (s *Server) HandleAddUser(w http.ResponseWriter, r *http.Request) error {
	req := new(CreateUserRequest)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return NewValidationError("Invalid JSON format: " + err.Error())
	}

	s.log.Info("recieved request", "handler", "HandleAddUser", "request", req)

	if err := validateCreateUserRequest(req); err != nil {
		return nil
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return err
	}

	newUser := &db.User{
		Name:     req.Name,
		Email:    strings.ToLower(strings.TrimSpace(req.Email)),
		Password: string(hashedPassword),
	}

	s.userStorer.CreateUser(s.ctx, s.userStorer)

}

package httpserver

import "net/http"

func handleHello(w http.ResponseWriter, r *http.Request) error {
	response := map[string]string{
		"message": "hello world",
		"status":  "success",
	}

	return WriteJSON(w, http.StatusOK, response)
}

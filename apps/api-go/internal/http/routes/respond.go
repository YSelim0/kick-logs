package routes

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Detail string `json:"detail"`
}

type statusResponse struct {
	Status string `json:"status"`
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}

func writeError(response http.ResponseWriter, status int, detail string) {
	writeJSON(response, status, errorResponse{Detail: detail})
}

func decodeJSON(request *http.Request, target any) error {
	defer request.Body.Close()
	return json.NewDecoder(request.Body).Decode(target)
}

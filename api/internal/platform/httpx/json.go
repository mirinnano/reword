package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
)

const maxJSONBody = 16 << 10

// ErrorBody is the shape of every error response.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// DecodeJSON reads a single JSON object with unknown fields rejected. On
// failure it writes a 400/413/415 response and returns false.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mt != "application/json" {
		WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			WriteError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body too large")
			return false
		}
		WriteError(w, http.StatusBadRequest, "invalid_json", fmt.Sprintf("invalid JSON body: %v", err))
		return false
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		WriteError(w, http.StatusBadRequest, "invalid_json", "body must contain a single JSON object")
		return false
	}
	return true
}

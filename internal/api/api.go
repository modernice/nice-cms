package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// FriendlyError is an error with a human-friendly message.
type FriendlyError struct {
	Err     error
	Message string
}

// Friendly returns a FriendlyError that wraps err with the provided message.
func Friendly(err error, format string, v ...any) error {
	return FriendlyError{
		Err:     err,
		Message: fmt.Sprintf(format, v...),
	}
}

func (err FriendlyError) Error() string {
	if err.Err != nil {
		return err.Err.Error()
	}
	return err.Message
}

func (err FriendlyError) Unwrap() error {
	return err.Err
}

func (err FriendlyError) FriendlyError() string {
	return err.Message
}

// Error writes a JSON error response to w with the error message in an "error" field:
//
//	api.Error(w, r, 404, errors.New("entity not found"))
//	// {"error": "entity not found"}
func Error(w http.ResponseWriter, r *http.Request, status int, err error) {
	var msg string
	if err != nil {
		msg = err.Error()
		if err, ok := err.(interface{ FriendlyError() string }); ok {
			msg = err.FriendlyError()
		}
	}

	if status != 0 {
		render.Status(r, status)
	}

	render.JSON(w, r, map[string]any{"error": msg})
}

func JSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	if status != 0 {
		render.Status(r, status)
	}
	render.JSON(w, r, v)
}

func NoContent(w http.ResponseWriter, r *http.Request) {
	render.NoContent(w, r)
}

func ParseUUID(raw, desc string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, Friendly(err, "Invalid UUID for %q: %v", desc, id)
	}
	return id, nil
}

func ExtractUUID(r *http.Request, name string) (uuid.UUID, error) {
	return ParseUUID(chi.URLParam(r, name), name)
}

func Decode(r io.Reader, v any) error {
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return Friendly(err, "Malformed JSON request: %v", err)
	}
	return nil
}

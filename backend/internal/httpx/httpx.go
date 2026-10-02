// Package httpx holds the small HTTP conveniences shared by every handler:
// JSON encoding, a bilingual error envelope, and request decoding with limits.
//
// Errors carry a machine code plus both languages. The frontend shows the
// language the user chose; the code is what tests and logs match on, so
// rewording a message never breaks anything.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// Error is the body of every failed response.
type Error struct {
	Code      string            `json:"code"`
	MessageEN string            `json:"messageEn"`
	MessageHI string            `json:"messageHi"`
	Fields    map[string]string `json:"fields,omitempty"`
}

type errorEnvelope struct {
	Error Error `json:"error"`
}

// APIError is an error that knows which HTTP status and which bilingual message
// it should produce. Handlers return these; one place turns them into responses.
type APIError struct {
	Status    int
	Code      string
	MessageEN string
	MessageHI string
	Fields    map[string]string
	// Internal is logged but never sent to the client, so a database error
	// never leaks a table name to a parent's browser.
	Internal error
}

func (e *APIError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Internal)
	}
	return e.Code
}

func (e *APIError) Unwrap() error { return e.Internal }

// NewError builds an APIError.
func NewError(status int, code, messageEN, messageHI string) *APIError {
	return &APIError{Status: status, Code: code, MessageEN: messageEN, MessageHI: messageHI}
}

// WithField attaches a per-field message so a form can highlight the one input
// that is wrong instead of showing a banner.
func (e *APIError) WithField(name, messageEN string) *APIError {
	if e.Fields == nil {
		e.Fields = map[string]string{}
	}
	e.Fields[name] = messageEN
	return e
}

// WithInternal attaches the underlying cause for the log only.
func (e *APIError) WithInternal(err error) *APIError {
	e.Internal = err
	return e
}

// Common errors, so the same wording appears everywhere.
var (
	ErrUnauthorized = func() *APIError {
		return NewError(http.StatusUnauthorized, "UNAUTHORIZED",
			"Please sign in again.",
			"कृपया दोबारा साइन इन करें।")
	}
	ErrForbidden = func() *APIError {
		return NewError(http.StatusForbidden, "FORBIDDEN",
			"You do not have permission to do this.",
			"आपको यह करने की अनुमति नहीं है।")
	}
	ErrNotFound = func() *APIError {
		return NewError(http.StatusNotFound, "NOT_FOUND",
			"That record was not found.",
			"वह रिकॉर्ड नहीं मिला।")
	}
	ErrBadRequest = func(messageEN, messageHI string) *APIError {
		return NewError(http.StatusBadRequest, "BAD_REQUEST", messageEN, messageHI)
	}
	ErrConflict = func(messageEN, messageHI string) *APIError {
		return NewError(http.StatusConflict, "CONFLICT", messageEN, messageHI)
	}
	ErrTooManyRequests = func() *APIError {
		return NewError(http.StatusTooManyRequests, "TOO_MANY_REQUESTS",
			"Too many attempts. Please wait a minute and try again.",
			"बहुत अधिक प्रयास। कृपया एक मिनट रुककर दोबारा कोशिश करें।")
	}
	ErrInternal = func() *APIError {
		return NewError(http.StatusInternalServerError, "INTERNAL",
			"Something went wrong at our end. Please try again.",
			"हमारी ओर से कुछ गड़बड़ हुई। कृपया दोबारा कोशिश करें।")
	}
)

// JSON writes a successful response.
func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// The status line is already sent, so there is nothing to do but log.
		slog.Error("encode response", "error", err)
	}
}

// NoContent writes a 204.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Fail turns any error into a response. Unknown errors become a generic 500 so
// internal detail never reaches the client.
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		apiErr = ErrInternal().WithInternal(err)
	}

	if apiErr.Status >= http.StatusInternalServerError {
		slog.Error("request failed",
			"method", r.Method, "path", r.URL.Path,
			"code", apiErr.Code, "error", apiErr.Error())
	} else {
		slog.Warn("request rejected",
			"method", r.Method, "path", r.URL.Path,
			"status", apiErr.Status, "code", apiErr.Code)
	}

	JSON(w, apiErr.Status, errorEnvelope{Error: Error{
		Code:      apiErr.Code,
		MessageEN: apiErr.MessageEN,
		MessageHI: apiErr.MessageHI,
		Fields:    apiErr.Fields,
	}})
}

// maxJSONBody caps a request body. A school form is a few kilobytes; anything
// larger is a mistake or an attack.
const maxJSONBody = 1 << 20 // 1 MiB

// DecodeJSON reads and validates a JSON request body.
func DecodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		return ErrBadRequest(
			"Send this request as JSON.",
			"यह अनुरोध JSON में भेजें।")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		var maxBytesErr *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxErr):
			return ErrBadRequest(
				"The request body is not valid JSON.",
				"अनुरोध का प्रारूप मान्य JSON नहीं है।").WithInternal(err)
		case errors.As(err, &typeErr):
			return ErrBadRequest(
				fmt.Sprintf("The field %q has the wrong type.", typeErr.Field),
				"किसी फ़ील्ड का प्रकार ग़लत है।").
				WithField(typeErr.Field, "Wrong type").WithInternal(err)
		case errors.As(err, &maxBytesErr):
			return NewError(http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE",
				"That request was too large.",
				"अनुरोध बहुत बड़ा है।").WithInternal(err)
		case errors.Is(err, io.EOF):
			return ErrBadRequest(
				"The request body was empty.",
				"अनुरोध खाली था।")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			field := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return ErrBadRequest(
				fmt.Sprintf("Unexpected field %s.", field),
				"अनुरोध में एक अज्ञात फ़ील्ड है।").WithInternal(err)
		default:
			return ErrBadRequest(
				"That request could not be read.",
				"अनुरोध पढ़ा नहीं जा सका।").WithInternal(err)
		}
	}

	// Exactly one JSON value per request body.
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ErrBadRequest(
			"Send a single JSON object.",
			"एक ही JSON ऑब्जेक्ट भेजें।")
	}
	return nil
}

// QueryInt reads a positive integer query parameter, falling back when absent
// or unparseable rather than failing the request.
func QueryInt(r *http.Request, key string, fallback, minimum, maximum int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

// Page describes a list response so every list in the API looks the same.
type Page[T any] struct {
	Items   []T  `json:"items"`
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"hasMore"`
}

// NewPage assembles a Page, guaranteeing Items is never nil so the frontend
// never has to guard against null where it expects an array.
func NewPage[T any](items []T, total, limit, offset int) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{
		Items:   items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+len(items) < total,
	}
}

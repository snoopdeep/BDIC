package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
	"bdic/backend/internal/store"
)

// contextWithTimeout derives a bounded context from the request, so a slow
// query cannot hold a connection open indefinitely.
func contextWithTimeout(r *http.Request, limit time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), limit)
}

// pathUUID reads a path parameter and checks it is a well-formed UUID before it
// reaches a query, so a junk id in a URL is a clean 400 rather than a database
// error in the log.
func pathUUID(r *http.Request, name string) (string, error) {
	value := r.PathValue(name)
	if value == "" {
		return "", httpx.ErrBadRequest(
			"That link is missing an identifier.",
			"इस लिंक में पहचानकर्ता नहीं है।")
	}
	if !auth.IsUUID(value) {
		return "", httpx.ErrBadRequest(
			"That identifier is not valid.",
			"यह पहचानकर्ता मान्य नहीं है।")
	}
	return value, nil
}

// queryUUID reads an optional UUID query parameter. An empty value is allowed
// and means "no filter"; a malformed one is rejected rather than ignored,
// because silently ignoring a filter shows the caller more than they asked for.
func queryUUID(r *http.Request, name string) (string, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return "", nil
	}
	if !auth.IsUUID(value) {
		return "", httpx.ErrBadRequest(
			"The "+name+" filter is not a valid identifier.",
			"फ़िल्टर में दिया गया पहचानकर्ता मान्य नहीं है।").
			WithField(name, "Not a valid id")
	}
	return value, nil
}

// storeError turns a store error into the right HTTP response.
func storeError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return httpx.ErrNotFound().WithInternal(err)
	case errors.Is(err, store.ErrConflict):
		return httpx.ErrConflict(
			"That record already exists.",
			"यह रिकॉर्ड पहले से मौजूद है।").WithInternal(err)
	default:
		return httpx.ErrInternal().WithInternal(err)
	}
}

// currentSessionID returns the academic session everything defaults to.
//
// Handlers accept an optional ?sessionId= to look at a past year, and fall back
// to the current one, so no screen has to ask the user which year they mean.
func (s *Server) currentSessionID(r *http.Request) (string, error) {
	requested, err := queryUUID(r, "sessionId")
	if err != nil {
		return "", err
	}
	if requested != "" {
		return requested, nil
	}

	session, err := s.store.CurrentSession(r.Context())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", httpx.ErrBadRequest(
				"No academic session is marked as current. Set one in school settings.",
				"कोई शैक्षणिक सत्र वर्तमान के रूप में चिह्नित नहीं है। सेटिंग्स में एक चुनें।")
		}
		return "", httpx.ErrInternal().WithInternal(err)
	}
	return session.ID, nil
}

// paginate reads the standard limit and offset query parameters.
func paginate(r *http.Request) (limit, offset int) {
	limit = httpx.QueryInt(r, "limit", 50, 1, 200)
	offset = httpx.QueryInt(r, "offset", 0, 0, 100_000)
	return limit, offset
}

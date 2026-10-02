// Package api wires the HTTP surface: the router, the middleware chain, and
// the handlers.
//
// Routing uses the standard library's ServeMux, which since Go 1.22 matches on
// method and path patterns. A third-party router would add a dependency for
// something the standard library now does.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/config"
	"bdic/backend/internal/files"
	"bdic/backend/internal/store"
)

// Server holds everything the handlers need.
type Server struct {
	cfg     config.Config
	store   *store.Store
	signer  *auth.Signer
	authmw  *auth.Middleware
	audit   *audit.Recorder
	storage files.Storage
	logger  *slog.Logger

	// Sign-in is limited per identifier and per address, so guessing one
	// account's password is slow and hammering the endpoint from one machine
	// is slower.
	loginLimiter *auth.Limiter
	// The public admission and enquiry forms need their own, looser limit:
	// a family filling in three children's applications is normal.
	publicLimiter *auth.Limiter
}

// New builds the server.
func New(
	cfg config.Config,
	dataStore *store.Store,
	signer *auth.Signer,
	recorder *audit.Recorder,
	storage files.Storage,
	logger *slog.Logger,
) *Server {
	return &Server{
		cfg:           cfg,
		store:         dataStore,
		signer:        signer,
		authmw:        auth.NewMiddleware(signer, dataStore),
		audit:         recorder,
		storage:       storage,
		logger:        logger,
		loginLimiter:  auth.NewLimiter(8, 10*time.Minute),
		publicLimiter: auth.NewLimiter(20, time.Hour),
	}
}

// signedIn wraps a handler so only an authenticated, active user reaches it.
func (s *Server) signedIn(handler http.HandlerFunc) http.Handler {
	return s.authmw.RequireAuth(handler)
}

// restricted wraps a handler so only the listed roles reach it. The role check
// is the coarse gate; queries still scope rows to the caller.
func (s *Server) restricted(handler http.HandlerFunc, roles ...string) http.Handler {
	return s.authmw.RequireAuth(auth.RequireRole(roles...)(handler))
}

// Handler returns the fully wired HTTP handler.
//
// Each module registers its own routes, in its own file. That keeps this
// function short enough to read, and it means two people adding two modules do
// not edit the same lines.
//
// Every route names its own middleware, so reading a register function tells
// you exactly who can reach what. There is no inherited protection to trace.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	s.registerCoreRoutes(mux)
	s.registerFileRoutes(mux)
	s.registerAdmissionRoutes(mux)
	s.registerPeopleRoutes(mux)
	s.registerAcademicRoutes(mux)
	s.registerAttendanceRoutes(mux)
	s.registerTimetableRoutes(mux)
	s.registerTeachingRoutes(mux)
	s.registerHomeworkRoutes(mux)
	s.registerCertificateRoutes(mux)
	s.registerExamRoutes(mux)
	s.registerFeeRoutes(mux)
	s.registerCommunicationRoutes(mux)
	s.registerSiteRoutes(mux)
	s.registerDashboardRoutes(mux)
	s.registerAuditRoutes(mux)

	// Outermost first: a panic anywhere inside still produces a response, and
	// the access log records every request including the rejected ones.
	return s.recoverPanics(s.logRequests(s.withCORS(s.withSecurityHeaders(mux))))
}

// registerCoreRoutes covers health, sign-in, the signed-in user, and the
// school's own structure.
func (s *Server) registerCoreRoutes(mux *http.ServeMux) {
	// ---- Open: no token required -------------------------------------------
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/public/school", s.handlePublicSchool)
	mux.HandleFunc("GET /api/v1/public/structure", s.handlePublicStructure)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/forgot-password", s.handleForgotPassword)
	mux.HandleFunc("POST /api/v1/auth/reset-password", s.handleResetPassword)

	// ---- Any signed-in user ------------------------------------------------
	mux.Handle("GET /api/v1/me", s.signedIn(s.handleMe))
	mux.Handle("POST /api/v1/me/locale", s.signedIn(s.handleSetLocale))
	mux.Handle("POST /api/v1/me/password", s.signedIn(s.handleChangePassword))
	mux.Handle("POST /api/v1/auth/logout", s.signedIn(s.handleLogout))

	// The school's structure is readable by everyone who is signed in: a
	// parent needs class names to read their child's timetable.
	mux.Handle("GET /api/v1/school", s.signedIn(s.handleGetSchool))
	mux.Handle("GET /api/v1/sessions", s.signedIn(s.handleListSessions))
	mux.Handle("GET /api/v1/classes", s.signedIn(s.handleListClasses))
	mux.Handle("GET /api/v1/subjects", s.signedIn(s.handleListSubjects))
	mux.Handle("GET /api/v1/classes/{classId}/subjects", s.signedIn(s.handleListClassSubjects))
	mux.Handle("GET /api/v1/settings", s.signedIn(s.handleListSettings))

	// ---- Management only ---------------------------------------------------
	mux.Handle("PUT /api/v1/school",
		s.restricted(s.handleUpdateSchool, auth.RoleSuperAdmin, auth.RolePrincipal))
	mux.Handle("PUT /api/v1/settings/{key}",
		s.restricted(s.handleUpdateSetting, auth.RoleSuperAdmin, auth.RolePrincipal))
}

// withCORS answers the browser's preflight and allows only the configured
// origins. A wildcard is never sent, because these responses carry student data.
func (s *Server) withCORS(next http.Handler) http.Handler {
	allowed := make(map[string]bool, len(s.cfg.CORSOrigins))
	for _, origin := range s.cfg.CORSOrigins {
		allowed[origin] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept-Language")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		// This is a JSON API, so nothing it returns should ever be cached by
		// a shared cache: one parent's data must not be served to another.
		w.Header().Set("Cache-Control", "no-store")
		if s.cfg.IsProduction() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the status code so the access log can report it.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

func (rec *statusRecorder) Write(b []byte) (int, error) {
	if rec.status == 0 {
		rec.status = http.StatusOK
	}
	n, err := rec.ResponseWriter.Write(b)
	rec.bytes += n
	return n, err
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		if recorder.status == 0 {
			recorder.status = http.StatusOK
		}

		// The query string is deliberately not logged. It can carry a phone
		// number or an admission number, and logs are the easiest place for
		// personal data to leak out of a system unnoticed.
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"bytes", recorder.bytes,
			"ms", time.Since(start).Milliseconds(),
		)
	})
}

func (s *Server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic recovered",
					"error", recovered,
					"method", r.Method,
					"path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(
					`{"error":{"code":"INTERNAL","messageEn":"Something went wrong at our end. Please try again.","messageHi":"हमारी ओर से कुछ गड़बड़ हुई। कृपया दोबारा कोशिश करें।"}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

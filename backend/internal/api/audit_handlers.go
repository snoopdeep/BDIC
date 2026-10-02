package api

import (
	"net/http"

	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
)

func (s *Server) registerAuditRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/audit-log", s.restricted(s.handleListAuditEvents, auth.RoleSuperAdmin, auth.RolePrincipal))
}

func (s *Server) handleListAuditEvents(w http.ResponseWriter, r *http.Request) {
	limit, offset := paginate(r)
	items, err := s.store.ListAuditEvents(r.Context(), limit, offset)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items), "limit": limit, "offset": offset})
}

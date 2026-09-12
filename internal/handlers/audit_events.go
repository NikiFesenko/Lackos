package handlers

import (
	"net/http"
	"strconv"

	"github.com/mcp-gate/mcp-gate/internal/api"
	"github.com/mcp-gate/mcp-gate/internal/db"
)

// ListAuditEvents handles GET /api/v1/audit-events
//
// Query params:
//   - limit        int    (default 50, max 200)
//   - offset       int    (default 0)
//   - user_id      uuid   (filter by user)
//   - downstream_server_id uuid (filter by server)
//   - outcome      string (allowed|denied|rate_limited|error)
func (h *Handler) ListAuditEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := int32(parseIntParam(q.Get("limit"), 50))
	if limit > 200 {
		limit = 200
	}
	offset := int32(parseIntParam(q.Get("offset"), 0))

	ctx := r.Context()

	// Filter by user_id
	if uid := q.Get("user_id"); uid != "" {
		userID, ok := parseUUID(w, uid)
		if !ok {
			return
		}
		events, err := h.q.ListAuditEventsByUser(ctx, db.ListAuditEventsByUserParams{
			UserID: userID,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to list audit events")
			return
		}
		api.WriteJSON(w, http.StatusOK, events)
		return
	}

	// Filter by downstream_server_id
	if sid := q.Get("downstream_server_id"); sid != "" {
		serverID, ok := parseUUID(w, sid)
		if !ok {
			return
		}
		events, err := h.q.ListAuditEventsByServer(ctx, db.ListAuditEventsByServerParams{
			DownstreamServerID: serverID,
			Limit:              limit,
			Offset:             offset,
		})
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to list audit events")
			return
		}
		api.WriteJSON(w, http.StatusOK, events)
		return
	}

	// Filter by outcome
	if outcome := q.Get("outcome"); outcome != "" {
		valid := map[string]bool{"allowed": true, "denied": true, "rate_limited": true, "error": true}
		if !valid[outcome] {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest,
				"outcome must be one of: allowed, denied, rate_limited, error")
			return
		}
		events, err := h.q.ListAuditEventsByOutcome(ctx, db.ListAuditEventsByOutcomeParams{
			Outcome: outcome,
			Limit:   limit,
			Offset:  offset,
		})
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to list audit events")
			return
		}
		api.WriteJSON(w, http.StatusOK, events)
		return
	}

	// No filter — return paginated full list
	events, err := h.q.ListAuditEvents(ctx, db.ListAuditEventsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to list audit events")
		return
	}
	api.WriteJSON(w, http.StatusOK, events)
}

func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}

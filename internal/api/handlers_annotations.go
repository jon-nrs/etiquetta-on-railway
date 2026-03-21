package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/caioricciuti/etiquetta/internal/auth"
)

var validCategories = map[string]bool{
	"deployment":  true,
	"analytics":   true,
	"performance": true,
	"consent":     true,
	"ads":         true,
	"content":     true,
	"other":       true,
}

var dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ListAnnotations returns annotations for a domain, optionally filtered by date range and category.
func (h *Handlers) ListAnnotations(w http.ResponseWriter, r *http.Request) {
	domainID := r.URL.Query().Get("domain_id")
	if domainID == "" {
		writeError(w, http.StatusBadRequest, "domain_id is required")
		return
	}

	query := "SELECT id, domain_id, date, title, description, category, source, created_by, created_at, updated_at FROM annotations WHERE domain_id = ?"
	args := []interface{}{domainID}

	if start := r.URL.Query().Get("start"); start != "" {
		query += " AND date >= ?"
		args = append(args, start)
	}
	if end := r.URL.Query().Get("end"); end != "" {
		query += " AND date <= ?"
		args = append(args, end)
	}
	if category := r.URL.Query().Get("category"); category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}

	query += " ORDER BY date DESC"

	rows, err := h.db.Conn().QueryContext(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query annotations")
		return
	}
	defer rows.Close()

	type Annotation struct {
		ID          string `json:"id"`
		DomainID    string `json:"domain_id"`
		Date        string `json:"date"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Source      string `json:"source"`
		CreatedBy   string `json:"created_by"`
		CreatedAt   int64  `json:"created_at"`
		UpdatedAt   int64  `json:"updated_at"`
	}

	result := make([]Annotation, 0)
	for rows.Next() {
		var a Annotation
		if err := rows.Scan(&a.ID, &a.DomainID, &a.Date, &a.Title, &a.Description, &a.Category, &a.Source, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan annotation")
			return
		}
		result = append(result, a)
	}

	writeJSON(w, http.StatusOK, result)
}

// CreateAnnotation creates a new annotation.
func (h *Handlers) CreateAnnotation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DomainID    string `json:"domain_id"`
		Date        string `json:"date"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DomainID == "" || req.Title == "" || req.Date == "" {
		writeError(w, http.StatusBadRequest, "domain_id, date, and title are required")
		return
	}

	if !dateRegex.MatchString(req.Date) {
		writeError(w, http.StatusBadRequest, "date must be in YYYY-MM-DD format")
		return
	}

	if req.Category == "" {
		req.Category = "other"
	}
	if !validCategories[req.Category] {
		writeError(w, http.StatusBadRequest, "invalid category")
		return
	}

	claims := auth.GetUserFromContext(r.Context())
	createdBy := ""
	if claims != nil {
		createdBy = claims.UserID
	}

	id := generateID()
	now := time.Now().UnixMilli()

	_, err := h.db.Conn().ExecContext(r.Context(),
		"INSERT INTO annotations (id, domain_id, date, title, description, category, source, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 'manual', ?, ?, ?)",
		id, req.DomainID, req.Date, req.Title, req.Description, req.Category, createdBy, now, now,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create annotation")
		return
	}

	h.logAudit(r, "create", "annotation", id, req.Title)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":          id,
		"domain_id":   req.DomainID,
		"date":        req.Date,
		"title":       req.Title,
		"description": req.Description,
		"category":    req.Category,
		"source":      "manual",
		"created_by":  createdBy,
		"created_at":  now,
		"updated_at":  now,
	})
}

// UpdateAnnotation updates an existing annotation.
func (h *Handlers) UpdateAnnotation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Date        *string `json:"date"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Category    *string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Date != nil && !dateRegex.MatchString(*req.Date) {
		writeError(w, http.StatusBadRequest, "date must be in YYYY-MM-DD format")
		return
	}
	if req.Category != nil && !validCategories[*req.Category] {
		writeError(w, http.StatusBadRequest, "invalid category")
		return
	}

	// Verify exists
	var exists int
	err := h.db.Conn().QueryRowContext(r.Context(), "SELECT 1 FROM annotations WHERE id = ?", id).Scan(&exists)
	if err != nil {
		writeError(w, http.StatusNotFound, "annotation not found")
		return
	}

	now := time.Now().UnixMilli()
	_, err = h.db.Conn().ExecContext(r.Context(),
		`UPDATE annotations SET
			date = COALESCE(?, date),
			title = COALESCE(?, title),
			description = COALESCE(?, description),
			category = COALESCE(?, category),
			updated_at = ?
		WHERE id = ?`,
		req.Date, req.Title, req.Description, req.Category, now, id,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update annotation")
		return
	}

	h.logAudit(r, "update", "annotation", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// DeleteAnnotation deletes an annotation.
func (h *Handlers) DeleteAnnotation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := h.db.Conn().ExecContext(r.Context(), "DELETE FROM annotations WHERE id = ?", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete annotation")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "annotation not found")
		return
	}

	h.logAudit(r, "delete", "annotation", id, "")
	w.WriteHeader(http.StatusNoContent)
}

package api

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/caioricciuti/etiquetta/internal/adfraud"
)

// GetStatsVitals returns web vitals (Pro feature)
func (h *Handlers) GetStatsVitals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	f := parseStatsFilter(r)

	where := "timestamp >= ? AND timestamp <= ? AND bot_category = 'human'"
	args := []interface{}{f.startMs, f.endMs}
	if f.domain != "" {
		where += " AND domain = ?"
		args = append(args, f.domain)
	}

	var lcp, cls, fcp, ttfb, inp float64
	var samples int64
	h.db.Conn().QueryRowContext(ctx, `
		SELECT
			COALESCE(AVG(lcp), 0),
			COALESCE(AVG(cls), 0),
			COALESCE(AVG(fcp), 0),
			COALESCE(AVG(ttfb), 0),
			COALESCE(AVG(inp), 0),
			COUNT(*)
		FROM performance
		WHERE `+where,
		args...).Scan(&lcp, &cls, &fcp, &ttfb, &inp, &samples)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"lcp":     lcp,
		"cls":     cls,
		"fcp":     fcp,
		"ttfb":    ttfb,
		"inp":     inp,
		"samples": samples,
	})
}

// GetStatsErrors returns error summary (Pro feature)
func (h *Handlers) GetStatsErrors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	f := parseStatsFilter(r)

	where := "timestamp >= ? AND timestamp <= ?"
	args := []interface{}{f.startMs, f.endMs}
	if f.domain != "" {
		where += " AND domain = ?"
		args = append(args, f.domain)
	}

	rows, err := h.db.Conn().QueryContext(ctx, `
		SELECT error_hash, error_type, error_message, COUNT(*) as occurrences, COUNT(DISTINCT session_id) as affected_sessions
		FROM errors
		WHERE `+where+`
		GROUP BY error_hash, error_type, error_message
		ORDER BY occurrences DESC
		LIMIT 10
	`, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var hash, errType, message string
		var occurrences, affected int64
		rows.Scan(&hash, &errType, &message, &occurrences, &affected)
		result = append(result, map[string]interface{}{
			"error_hash":        hash,
			"error_type":        errType,
			"error_message":     message,
			"occurrences":       occurrences,
			"affected_sessions": affected,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// ExportEvents exports events as JSON (Pro feature)
func (h *Handlers) ExportEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Get date range from query params
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	query := "SELECT * FROM events"
	var args []interface{}

	if from != "" || to != "" {
		query += " WHERE 1=1"
		if from != "" {
			fromTime, _ := time.Parse(time.RFC3339, from)
			query += " AND timestamp >= ?"
			args = append(args, fromTime.UnixMilli())
		}
		if to != "" {
			toTime, _ := time.Parse(time.RFC3339, to)
			query += " AND timestamp <= ?"
			args = append(args, toTime.UnixMilli())
		}
	}

	query += " ORDER BY timestamp DESC LIMIT 100000"

	rows, err := h.db.Conn().QueryContext(ctx, query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	format := r.URL.Query().Get("format")

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=events.csv")

		cw := csv.NewWriter(w)
		defer cw.Flush()

		cw.Write(cols)

		for rows.Next() {
			values := make([]interface{}, len(cols))
			valuePtrs := make([]interface{}, len(cols))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			rows.Scan(valuePtrs...)

			record := make([]string, len(cols))
			for i, v := range values {
				if v == nil {
					record[i] = ""
				} else if b, ok := v.([]byte); ok {
					record[i] = string(b)
				} else {
					record[i] = fmt.Sprintf("%v", v)
				}
			}
			cw.Write(record)
		}
		return
	}

	// Default: JSON
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=events.json")

	encoder := json.NewEncoder(w)
	w.Write([]byte("["))
	first := true

	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		rows.Scan(valuePtrs...)

		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}

		if !first {
			w.Write([]byte(","))
		}
		first = false
		encoder.Encode(row)
	}

	w.Write([]byte("]"))
}

// GetFraudSummary returns fraud detection summary
func (h *Handlers) GetFraudSummary(w http.ResponseWriter, r *http.Request) {
	days := getDaysParam(r, 7)
	domain := getDomainParam(r)

	detector := adfraud.NewDetector(h.db.Conn())
	summary, err := detector.GetFraudSummary(domain, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// GetSourceQuality returns traffic quality per source
func (h *Handlers) GetSourceQuality(w http.ResponseWriter, r *http.Request) {
	days := getDaysParam(r, 7)
	domain := getDomainParam(r)

	detector := adfraud.NewDetector(h.db.Conn())
	sources, err := detector.GetSourceQuality(domain, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, sources)
}

// ListCampaigns returns all campaigns
func (h *Handlers) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	analyzer := adfraud.NewSpendAnalyzer(h.db.Conn())
	campaigns, err := analyzer.ListCampaigns()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, campaigns)
}

// CreateCampaign creates a new campaign
func (h *Handlers) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string  `json:"name"`
		UTMSource   *string `json:"utm_source,omitempty"`
		UTMMedium   *string `json:"utm_medium,omitempty"`
		UTMCampaign *string `json:"utm_campaign,omitempty"`
		CPC         float64 `json:"cpc"`
		CPM         float64 `json:"cpm"`
		Budget      float64 `json:"budget"`
		StartDate   *int64  `json:"start_date,omitempty"`
		EndDate     *int64  `json:"end_date,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "Campaign name is required")
		return
	}

	campaign := &adfraud.Campaign{
		ID:          generateID(),
		Name:        input.Name,
		UTMSource:   input.UTMSource,
		UTMMedium:   input.UTMMedium,
		UTMCampaign: input.UTMCampaign,
		CPC:         input.CPC,
		CPM:         input.CPM,
		Budget:      input.Budget,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
	}

	analyzer := adfraud.NewSpendAnalyzer(h.db.Conn())
	if err := analyzer.CreateCampaign(campaign); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, campaign)
}

// GetCampaignReport returns fraud report for a campaign
func (h *Handlers) GetCampaignReport(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	domain := getDomainParam(r)

	analyzer := adfraud.NewSpendAnalyzer(h.db.Conn())
	report, err := analyzer.GetCampaignReport(campaignID, domain)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Campaign not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// DeleteCampaign removes a campaign
func (h *Handlers) DeleteCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")

	analyzer := adfraud.NewSpendAnalyzer(h.db.Conn())
	if err := analyzer.DeleteCampaign(campaignID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

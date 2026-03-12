package api

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/caioricciuti/etiquetta/internal/database"
)

// LookupVisitorData returns record counts for a visitor hash (GDPR Art. 15 - right of access)
func (h *Handlers) LookupVisitorData(w http.ResponseWriter, r *http.Request) {
	visitorHash := chi.URLParam(r, "visitorHash")
	if visitorHash == "" {
		writeError(w, http.StatusBadRequest, "Missing visitor hash")
		return
	}

	counts, err := h.db.LookupVisitorData(visitorHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to lookup visitor data")
		return
	}

	var total int64
	for _, c := range counts {
		total += c
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"visitor_hash": visitorHash,
		"tables":       counts,
		"total_records": total,
	})
}

// EraseVisitorData deletes all data for a visitor hash (GDPR Art. 17 - right to erasure)
func (h *Handlers) EraseVisitorData(w http.ResponseWriter, r *http.Request) {
	visitorHash := chi.URLParam(r, "visitorHash")
	if visitorHash == "" {
		writeError(w, http.StatusBadRequest, "Missing visitor hash")
		return
	}

	counts, err := h.db.EraseVisitorData(visitorHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to erase visitor data")
		return
	}

	var total int64
	for _, c := range counts {
		total += c
	}

	log.Printf("[privacy] Erased data for visitor %s: %v (total: %d records)", visitorHash, counts, total)
	h.logAudit(r, "erase", "visitor_data", visitorHash, fmt.Sprintf("Erased %d total records", total))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"visitor_hash":    visitorHash,
		"deleted":         counts,
		"total_deleted":   total,
		"erased_at":       time.Now().UTC().Format(time.RFC3339),
	})
}

// GetPrivacyAudit returns ePrivacy/GDPR compliance status
func (h *Handlers) GetPrivacyAudit(w http.ResponseWriter, r *http.Request) {
	// Check DNT setting
	var respectDNT string
	h.db.Conn().QueryRow("SELECT value FROM settings WHERE key = 'respect_dnt'").Scan(&respectDNT)

	// Check session timeout
	var sessionTimeout string
	h.db.Conn().QueryRow("SELECT value FROM settings WHERE key = 'session_timeout_minutes'").Scan(&sessionTimeout)

	// Get data retention config from license
	retentionDays := h.licenseManager.GetLimit("max_retention_days")

	// Count total records across tables
	var eventCount, perfCount, errorCount, consentCount, sessionCount int64
	h.db.Conn().QueryRow("SELECT COUNT(*) FROM events").Scan(&eventCount)
	h.db.Conn().QueryRow("SELECT COUNT(*) FROM performance").Scan(&perfCount)
	h.db.Conn().QueryRow("SELECT COUNT(*) FROM errors").Scan(&errorCount)
	h.db.Conn().QueryRow("SELECT COUNT(*) FROM consent_records").Scan(&consentCount)
	h.db.Conn().QueryRow("SELECT COUNT(*) FROM visitor_sessions").Scan(&sessionCount)

	// Check consent configs per domain
	type domainConsent struct {
		DomainID   string `json:"domain_id"`
		DomainName string `json:"domain_name"`
		Domain     string `json:"domain"`
		HasConsent bool   `json:"has_consent"`
		Version    int    `json:"version"`
	}

	var domainConsents []domainConsent
	rows, err := h.db.Conn().Query(`
		SELECT d.id, d.name, d.domain,
			COALESCE(cc.version, 0) as version,
			CASE WHEN cc.id IS NOT NULL THEN 1 ELSE 0 END as has_consent
		FROM domains d
		LEFT JOIN consent_configs cc ON cc.domain_id = d.id AND cc.is_active = 1
		WHERE d.is_active = 1
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dc domainConsent
			var hasConsent int
			if rows.Scan(&dc.DomainID, &dc.DomainName, &dc.Domain, &dc.Version, &hasConsent) == nil {
				dc.HasConsent = hasConsent == 1
				domainConsents = append(domainConsents, dc)
			}
		}
	}
	if domainConsents == nil {
		domainConsents = []domainConsent{}
	}

	// Build audit checks
	type auditCheck struct {
		ID          string `json:"id"`
		Category    string `json:"category"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"` // "pass", "warn", "info"
		Detail      string `json:"detail"`
	}

	checks := []auditCheck{
		{
			ID:          "no_cookies",
			Category:    "ePrivacy",
			Name:        "Cookie-free tracking",
			Description: "Analytics tracker does not set any cookies on visitor browsers",
			Status:      "pass",
			Detail:      "tracker.js uses server-side HMAC hashing for visitor identification — zero cookies",
		},
		{
			ID:          "ip_anonymization",
			Category:    "GDPR",
			Name:        "IP anonymization",
			Description: "IP addresses are anonymized before any processing or storage",
			Status:      "pass",
			Detail:      "IPv4 addresses masked to /24 subnet, then HMAC-SHA256 hashed with secret key. Raw IPs are never stored.",
		},
		{
			ID:          "no_pii",
			Category:    "GDPR",
			Name:        "No PII collection",
			Description: "No personally identifiable information is collected by default",
			Status:      "pass",
			Detail:      "No names, emails, or user IDs collected. Visitor identification uses irreversible hashes only.",
		},
		{
			ID:          "server_side_sessions",
			Category:    "ePrivacy",
			Name:        "Server-side sessions",
			Description: "Session tracking is computed server-side without client storage",
			Status:      "pass",
			Detail:      "Sessions derived from HMAC(IP subnet + User-Agent + time window). Timeout: " + sessionTimeout + " minutes.",
		},
		{
			ID:       "data_retention",
			Category: "GDPR",
			Name:     "Data retention policy",
			Description: "Automated data cleanup enforces retention limits",
			Status:   "pass",
			Detail:   func() string {
				if retentionDays > 0 {
					return fmt.Sprintf("Data automatically deleted after %d days (runs every 24h)", retentionDays)
				}
				return "Unlimited retention (Community tier). Consider setting a retention policy."
			}(),
		},
		{
			ID:          "right_to_erasure",
			Category:    "GDPR",
			Name:        "Right to erasure (Art. 17)",
			Description: "Ability to delete all data for a specific visitor on request",
			Status:      "pass",
			Detail:      "Erasure: DELETE /api/privacy/erasure/{visitorHash}. Export: GET /api/privacy/export/{visitorHash}. Covers all tables.",
		},
		{
			ID:          "data_minimization",
			Category:    "GDPR",
			Name:        "Data minimization",
			Description: "Only essential analytics data is collected",
			Status:      "pass",
			Detail:      "Collects: page URL, referrer, UTM params, browser/OS/device type, country/city (from GeoIP), bot signals. No fingerprinting beyond basic screen properties.",
		},
		{
			ID:          "self_hosted",
			Category:    "GDPR",
			Name:        "Self-hosted / data sovereignty",
			Description: "All data stays on your infrastructure — no third-party transfers",
			Status:      "pass",
			Detail:      "Single binary deployment. Data stored in local SQLite database. No external API calls for analytics.",
		},
	}

	// DNT check
	if respectDNT == "true" {
		checks = append(checks, auditCheck{
			ID:          "respect_dnt",
			Category:    "ePrivacy",
			Name:        "Do Not Track / Global Privacy Control",
			Description: "Respects browser DNT and Sec-GPC signals",
			Status:      "pass",
			Detail:      "DNT and Sec-GPC headers are enforced server-side and client-side. Visitors with these signals are not tracked.",
		})
	} else {
		checks = append(checks, auditCheck{
			ID:          "respect_dnt",
			Category:    "ePrivacy",
			Name:        "Do Not Track",
			Description: "Respects browser Do Not Track signal",
			Status:      "info",
			Detail:      "DNT is not enforced. Since Etiquetta is cookie-free and collects no PII, DNT compliance is optional but recommended.",
		})
	}

	// Consent banner checks per domain
	domainsWithConsent := 0
	for _, dc := range domainConsents {
		if dc.HasConsent {
			domainsWithConsent++
		}
	}
	if len(domainConsents) > 0 && domainsWithConsent < len(domainConsents) {
		checks = append(checks, auditCheck{
			ID:          "consent_banner",
			Category:    "ePrivacy",
			Name:        "Consent banner coverage",
			Description: "Consent banners are optional for cookie-free analytics",
			Status:      "info",
			Detail:      fmt.Sprintf("%d of %d domains have consent banners. Since Etiquetta uses cookie-free tracking, consent banners are optional under ePrivacy. Enable them only if you inject third-party scripts via the Tag Manager.", domainsWithConsent, len(domainConsents)),
		})
	} else if len(domainConsents) > 0 {
		checks = append(checks, auditCheck{
			ID:          "consent_banner",
			Category:    "ePrivacy",
			Name:        "Consent banner coverage",
			Description: "All tracked domains should have a consent banner configured",
			Status:      "pass",
			Detail:      fmt.Sprintf("All %d domains have consent banners configured.", len(domainConsents)),
		})
	}

	// Cookies inventory
	cookieInventory := []map[string]interface{}{
		{
			"name":        "etiquetta_session",
			"purpose":     "Admin authentication (JWT)",
			"type":        "strictly_necessary",
			"set_by":      "server",
			"visitor_facing": false,
			"duration":    "7 days",
			"description": "Authenticates admin users to the analytics dashboard. Never set on tracked visitor browsers.",
		},
		{
			"name":        "etiquetta_consent",
			"purpose":     "Stores visitor consent preferences",
			"type":        "strictly_necessary",
			"set_by":      "consent banner (c.js)",
			"visitor_facing": true,
			"duration":    "Configurable (default 365 days)",
			"description": "Only set when consent banner is enabled. Stores accept/reject decision. Required for consent management.",
		},
	}

	// Data inventory
	dataInventory := []map[string]string{
		{"field": "visitor_hash", "purpose": "Unique visitor identification", "pii": "no", "note": "Irreversible HMAC hash of masked IP + User-Agent"},
		{"field": "session_id", "purpose": "Session grouping", "pii": "no", "note": "Time-windowed HMAC hash, rotates every session timeout"},
		{"field": "url / path", "purpose": "Page analytics", "pii": "no", "note": "Which pages are visited"},
		{"field": "referrer_url", "purpose": "Traffic source analysis", "pii": "no", "note": "Where visitors come from"},
		{"field": "utm_*", "purpose": "Campaign tracking", "pii": "no", "note": "UTM parameters from URL"},
		{"field": "geo_country / city / region", "purpose": "Geographic analytics", "pii": "no", "note": "Derived from IP via GeoIP lookup, IP not stored"},
		{"field": "browser_name / os_name / device_type", "purpose": "Technology analytics", "pii": "no", "note": "Parsed from User-Agent header"},
		{"field": "bot_score / bot_signals", "purpose": "Bot detection", "pii": "no", "note": "Automated traffic filtering"},
		{"field": "ip_hash", "purpose": "Bot/fraud analysis", "pii": "no", "note": "SHA-256 hash of IP, not reversible"},
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"checks":           checks,
		"cookie_inventory": cookieInventory,
		"data_inventory":   dataInventory,
		"domain_consents":  domainConsents,
		"storage_summary": map[string]int64{
			"events":           eventCount,
			"performance":      perfCount,
			"errors":           errorCount,
			"consent_records":  consentCount,
			"visitor_sessions": sessionCount,
		},
		"data_retention_days": retentionDays,
		"generated_at":        time.Now().UTC().Format(time.RFC3339),
	})
}

// ExportVisitorData exports all data for a visitor hash (GDPR Art. 15 - right of access / Art. 20 - data portability)
func (h *Handlers) ExportVisitorData(w http.ResponseWriter, r *http.Request) {
	visitorHash := chi.URLParam(r, "visitorHash")
	if visitorHash == "" {
		writeError(w, http.StatusBadRequest, "Missing visitor hash")
		return
	}

	export, err := h.db.ExportVisitorData(visitorHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to export visitor data")
		return
	}

	log.Printf("[privacy] Exported data for visitor %s: %d records", visitorHash, export.TotalRows)
	h.logAudit(r, "export", "visitor_data", visitorHash, fmt.Sprintf("Exported %d total records", export.TotalRows))

	format := r.URL.Query().Get("format")
	shortHash := visitorHash
	if len(shortHash) > 12 {
		shortHash = shortHash[:12]
	}

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=visitor-%s-export.csv", shortHash))
		writeVisitorCSV(w, export)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=visitor-%s-export.json", shortHash))
	writeJSON(w, http.StatusOK, export)
}

// writeVisitorCSV writes a multi-table CSV export
func writeVisitorCSV(w http.ResponseWriter, export *database.VisitorDataExport) {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	tableOrder := []string{"events", "performance", "errors", "consent_records", "visitor_sessions"}
	for _, tableName := range tableOrder {
		td, ok := export.Tables[tableName]
		if !ok || td.Count == 0 {
			continue
		}

		// Section header as a comment row
		cw.Write([]string{fmt.Sprintf("# Table: %s (%d rows)", tableName, td.Count)})

		// Column headers
		cw.Write(td.Columns)

		// Data rows
		for _, row := range td.Rows {
			record := make([]string, len(row))
			for i, v := range row {
				if v == nil {
					record[i] = ""
				} else {
					record[i] = fmt.Sprintf("%v", v)
				}
			}
			cw.Write(record)
		}

		// Blank separator between tables
		cw.Write([]string{})
	}
}

// GetAuditLog returns paginated admin audit log entries
func (h *Handlers) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	action := r.URL.Query().Get("action")
	resourceType := r.URL.Query().Get("resource_type")

	entries, total, err := h.db.QueryAuditLog(page, perPage, action, resourceType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query audit log")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries":  entries,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

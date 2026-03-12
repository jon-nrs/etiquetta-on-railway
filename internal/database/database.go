package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// Event represents a tracking event
type Event struct {
	ID           string          `json:"id"`
	Timestamp    time.Time       `json:"timestamp"`
	EventType    string          `json:"event_type"`
	EventName    *string         `json:"event_name,omitempty"`
	SessionID    string          `json:"session_id"`
	VisitorHash  string          `json:"visitor_hash"`
	Domain       string          `json:"domain"`
	URL          string          `json:"url"`
	Path         string          `json:"path"`
	PageTitle    *string         `json:"page_title,omitempty"`
	ReferrerURL  *string         `json:"referrer_url,omitempty"`
	ReferrerType *string         `json:"referrer_type,omitempty"`
	UTMSource    *string         `json:"utm_source,omitempty"`
	UTMMedium    *string         `json:"utm_medium,omitempty"`
	UTMCampaign  *string         `json:"utm_campaign,omitempty"`
	GeoCountry   *string         `json:"geo_country,omitempty"`
	GeoCity      *string         `json:"geo_city,omitempty"`
	GeoRegion    *string         `json:"geo_region,omitempty"`
	GeoLatitude  *float64        `json:"geo_latitude,omitempty"`
	GeoLongitude *float64        `json:"geo_longitude,omitempty"`
	BrowserName  *string         `json:"browser_name,omitempty"`
	OSName       *string         `json:"os_name,omitempty"`
	DeviceType   *string         `json:"device_type,omitempty"`
	IsBot        bool            `json:"is_bot"`
	Props        json.RawMessage `json:"props,omitempty"`

	// Bot detection fields
	BotScore     int     `json:"bot_score"`
	BotSignals   string  `json:"bot_signals"`
	BotCategory  string  `json:"bot_category"`
	HasScroll    bool    `json:"has_scroll"`
	HasMouseMove bool    `json:"has_mouse_move"`
	HasClick     bool    `json:"has_click"`
	HasTouch     bool    `json:"has_touch"`
	ClickX       *int    `json:"click_x,omitempty"`
	ClickY       *int    `json:"click_y,omitempty"`
	PageDuration *int    `json:"page_duration,omitempty"`
	DatacenterIP bool    `json:"datacenter_ip"`
	IPHash       *string `json:"ip_hash,omitempty"`
}

// Performance represents web vitals
type Performance struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	SessionID      string    `json:"session_id"`
	VisitorHash    string    `json:"visitor_hash"`
	Domain         string    `json:"domain"`
	URL            string    `json:"url"`
	Path           string    `json:"path"`
	LCP            *float64  `json:"lcp,omitempty"`
	CLS            *float64  `json:"cls,omitempty"`
	FCP            *float64  `json:"fcp,omitempty"`
	TTFB           *float64  `json:"ttfb,omitempty"`
	INP            *float64  `json:"inp,omitempty"`
	PageLoadTime   *float64  `json:"page_load_time,omitempty"`
	DeviceType     *string   `json:"device_type,omitempty"`
	ConnectionType *string   `json:"connection_type,omitempty"`
	GeoCountry     *string   `json:"geo_country,omitempty"`
}

// Error represents a JS error
type Error struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	SessionID    string    `json:"session_id"`
	VisitorHash  string    `json:"visitor_hash"`
	Domain       string    `json:"domain"`
	URL          string    `json:"url"`
	Path         string    `json:"path"`
	ErrorType    string    `json:"error_type"`
	ErrorMessage string    `json:"error_message"`
	ErrorStack   *string   `json:"error_stack,omitempty"`
	ErrorHash    string    `json:"error_hash"`
	ScriptURL    *string   `json:"script_url,omitempty"`
	LineNumber   *int      `json:"line_number,omitempty"`
	ColumnNumber *int      `json:"column_number,omitempty"`
	BrowserName  *string   `json:"browser_name,omitempty"`
	GeoCountry   *string   `json:"geo_country,omitempty"`
}

func New(path string) (*DB, error) {
	// Ensure data directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Enable WAL mode and other optimizations via connection string
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=10000&_cache_size=-20000", path)

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Single connection serializes all writes — database/sql blocks goroutines
	// waiting for a connection, preventing SQLITE_BUSY entirely.
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0)

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Ensure PRAGMAs are applied (DSN _-prefixed params may not be parsed by modernc.org/sqlite)
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=10000",
		"PRAGMA cache_size=-20000",
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p); err != nil {
			return nil, fmt.Errorf("failed to set %s: %w", p, err)
		}
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}

// InsertEvent inserts a tracking event
func (db *DB) InsertEvent(e *Event) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	props := "{}"
	if e.Props != nil {
		props = string(e.Props)
	}

	botSignals := "[]"
	if e.BotSignals != "" {
		botSignals = e.BotSignals
	}

	botCategory := "human"
	if e.BotCategory != "" {
		botCategory = e.BotCategory
	}

	_, err := db.conn.Exec(`
		INSERT INTO events (
			id, timestamp, event_type, event_name, session_id, visitor_hash,
			domain, url, path, page_title, referrer_url, referrer_type,
			utm_source, utm_medium, utm_campaign,
			geo_country, geo_city, geo_region, geo_latitude, geo_longitude,
			browser_name, os_name, device_type, is_bot, props,
			bot_score, bot_signals, bot_category,
			has_scroll, has_mouse_move, has_click, has_touch,
			click_x, click_y, page_duration, datacenter_ip, ip_hash
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		e.ID, e.Timestamp.UnixMilli(), e.EventType, e.EventName, e.SessionID, e.VisitorHash,
		e.Domain, e.URL, e.Path, e.PageTitle, e.ReferrerURL, e.ReferrerType,
		e.UTMSource, e.UTMMedium, e.UTMCampaign,
		e.GeoCountry, e.GeoCity, e.GeoRegion, e.GeoLatitude, e.GeoLongitude,
		e.BrowserName, e.OSName, e.DeviceType, e.IsBot, props,
		e.BotScore, botSignals, botCategory,
		e.HasScroll, e.HasMouseMove, e.HasClick, e.HasTouch,
		e.ClickX, e.ClickY, e.PageDuration, e.DatacenterIP, e.IPHash,
	)
	return err
}

// InsertPerformance inserts web vitals data
func (db *DB) InsertPerformance(p *Performance) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		INSERT INTO performance (
			id, timestamp, session_id, visitor_hash, domain, url, path,
			lcp, cls, fcp, ttfb, inp, page_load_time,
			device_type, connection_type, geo_country
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		p.ID, p.Timestamp.UnixMilli(), p.SessionID, p.VisitorHash, p.Domain, p.URL, p.Path,
		p.LCP, p.CLS, p.FCP, p.TTFB, p.INP, p.PageLoadTime,
		p.DeviceType, p.ConnectionType, p.GeoCountry,
	)
	return err
}

// InsertError inserts a JS error
func (db *DB) InsertError(e *Error) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		INSERT INTO errors (
			id, timestamp, session_id, visitor_hash, domain, url, path,
			error_type, error_message, error_stack, error_hash,
			script_url, line_number, column_number, browser_name, geo_country
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		e.ID, e.Timestamp.UnixMilli(), e.SessionID, e.VisitorHash, e.Domain, e.URL, e.Path,
		e.ErrorType, e.ErrorMessage, e.ErrorStack, e.ErrorHash,
		e.ScriptURL, e.LineNumber, e.ColumnNumber, e.BrowserName, e.GeoCountry,
	)
	return err
}

// InsertBatch inserts multiple events in a transaction
func (db *DB) InsertBatch(events []*Event, perfs []*Performance, errs []*Error) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare statements
	eventStmt, err := tx.Prepare(`
		INSERT INTO events (
			id, timestamp, event_type, event_name, session_id, visitor_hash,
			domain, url, path, page_title, referrer_url, referrer_type,
			utm_source, utm_medium, utm_campaign,
			geo_country, geo_city, geo_region, geo_latitude, geo_longitude,
			browser_name, os_name, device_type, is_bot, props,
			bot_score, bot_signals, bot_category,
			has_scroll, has_mouse_move, has_click, has_touch,
			click_x, click_y, page_duration, datacenter_ip, ip_hash
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer eventStmt.Close()

	perfStmt, err := tx.Prepare(`
		INSERT INTO performance (
			id, timestamp, session_id, visitor_hash, domain, url, path,
			lcp, cls, fcp, ttfb, inp, page_load_time,
			device_type, connection_type, geo_country
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer perfStmt.Close()

	errStmt, err := tx.Prepare(`
		INSERT INTO errors (
			id, timestamp, session_id, visitor_hash, domain, url, path,
			error_type, error_message, error_stack, error_hash,
			script_url, line_number, column_number, browser_name, geo_country
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer errStmt.Close()

	// Insert events
	for _, e := range events {
		props := "{}"
		if e.Props != nil {
			props = string(e.Props)
		}
		botSignals := "[]"
		if e.BotSignals != "" {
			botSignals = e.BotSignals
		}
		botCategory := "human"
		if e.BotCategory != "" {
			botCategory = e.BotCategory
		}
		_, err := eventStmt.Exec(
			e.ID, e.Timestamp.UnixMilli(), e.EventType, e.EventName, e.SessionID, e.VisitorHash,
			e.Domain, e.URL, e.Path, e.PageTitle, e.ReferrerURL, e.ReferrerType,
			e.UTMSource, e.UTMMedium, e.UTMCampaign,
			e.GeoCountry, e.GeoCity, e.GeoRegion, e.GeoLatitude, e.GeoLongitude,
			e.BrowserName, e.OSName, e.DeviceType, e.IsBot, props,
			e.BotScore, botSignals, botCategory,
			e.HasScroll, e.HasMouseMove, e.HasClick, e.HasTouch,
			e.ClickX, e.ClickY, e.PageDuration, e.DatacenterIP, e.IPHash,
		)
		if err != nil {
			return err
		}
	}

	// Insert performance
	for _, p := range perfs {
		_, err := perfStmt.Exec(
			p.ID, p.Timestamp.UnixMilli(), p.SessionID, p.VisitorHash, p.Domain, p.URL, p.Path,
			p.LCP, p.CLS, p.FCP, p.TTFB, p.INP, p.PageLoadTime,
			p.DeviceType, p.ConnectionType, p.GeoCountry,
		)
		if err != nil {
			return err
		}
	}

	// Insert errors
	for _, e := range errs {
		_, err := errStmt.Exec(
			e.ID, e.Timestamp.UnixMilli(), e.SessionID, e.VisitorHash, e.Domain, e.URL, e.Path,
			e.ErrorType, e.ErrorMessage, e.ErrorStack, e.ErrorHash,
			e.ScriptURL, e.LineNumber, e.ColumnNumber, e.BrowserName, e.GeoCountry,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetEventCount returns total event count
func (db *DB) GetEventCount() (int64, error) {
	var count int64
	err := db.conn.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	return count, err
}

// LookupVisitorData counts records across all tables for a given visitor hash
func (db *DB) LookupVisitorData(visitorHash string) (map[string]int64, error) {
	counts := map[string]int64{}

	tables := []struct {
		name   string
		column string
	}{
		{"events", "visitor_hash"},
		{"performance", "visitor_hash"},
		{"errors", "visitor_hash"},
		{"consent_records", "visitor_hash"},
		{"visitor_sessions", "visitor_hash"},
	}

	for _, t := range tables {
		var count int64
		err := db.conn.QueryRow(
			fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", t.name, t.column),
			visitorHash,
		).Scan(&count)
		if err != nil {
			// Table may not exist yet — treat as zero
			counts[t.name] = 0
			continue
		}
		counts[t.name] = count
	}

	return counts, nil
}

// EraseVisitorData deletes all data for a visitor hash across all tables (GDPR Art. 17)
func (db *DB) EraseVisitorData(visitorHash string) (map[string]int64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Count first so we can report what was deleted
	counts := map[string]int64{}

	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	tables := []struct {
		name   string
		column string
	}{
		{"events", "visitor_hash"},
		{"performance", "visitor_hash"},
		{"errors", "visitor_hash"},
		{"consent_records", "visitor_hash"},
		{"visitor_sessions", "visitor_hash"},
	}

	for _, t := range tables {
		result, err := tx.Exec(
			fmt.Sprintf("DELETE FROM %s WHERE %s = ?", t.name, t.column),
			visitorHash,
		)
		if err != nil {
			// Table may not exist — skip
			counts[t.name] = 0
			continue
		}
		affected, _ := result.RowsAffected()
		counts[t.name] = affected
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return counts, nil
}

// VisitorTableData holds column names and row data for one table
type VisitorTableData struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Count   int             `json:"count"`
}

// VisitorDataExport holds all data for a single visitor across all tables
type VisitorDataExport struct {
	VisitorHash string                       `json:"visitor_hash"`
	Tables      map[string]*VisitorTableData `json:"tables"`
	TotalRows   int                          `json:"total_rows"`
	ExportedAt  string                       `json:"exported_at"`
}

// ExportVisitorData returns all data for a visitor hash across all tables (GDPR Art. 15)
func (db *DB) ExportVisitorData(visitorHash string) (*VisitorDataExport, error) {
	export := &VisitorDataExport{
		VisitorHash: visitorHash,
		Tables:      make(map[string]*VisitorTableData),
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	tableNames := []string{"events", "performance", "errors", "consent_records", "visitor_sessions"}

	for _, table := range tableNames {
		rows, err := db.conn.Query(
			fmt.Sprintf("SELECT * FROM %s WHERE visitor_hash = ? LIMIT 10000", table),
			visitorHash,
		)
		if err != nil {
			export.Tables[table] = &VisitorTableData{Columns: []string{}, Rows: [][]interface{}{}}
			continue
		}

		cols, _ := rows.Columns()
		td := &VisitorTableData{Columns: cols}

		for rows.Next() {
			values := make([]interface{}, len(cols))
			valuePtrs := make([]interface{}, len(cols))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			if err := rows.Scan(valuePtrs...); err != nil {
				continue
			}
			// Convert []byte to string for JSON compatibility
			row := make([]interface{}, len(cols))
			for i, v := range values {
				if b, ok := v.([]byte); ok {
					row[i] = string(b)
				} else {
					row[i] = v
				}
			}
			td.Rows = append(td.Rows, row)
		}
		rows.Close()

		if td.Rows == nil {
			td.Rows = [][]interface{}{}
		}
		td.Count = len(td.Rows)
		export.Tables[table] = td
		export.TotalRows += td.Count
	}

	return export, nil
}

// AuditLogEntry represents an admin audit log entry
type AuditLogEntry struct {
	ID           string `json:"id"`
	Timestamp    int64  `json:"timestamp"`
	UserID       string `json:"user_id"`
	UserEmail    string `json:"user_email"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Detail       string `json:"detail"`
	IPAddress    string `json:"ip_address"`
}

// InsertAuditLog records an admin action
func (db *DB) InsertAuditLog(entry *AuditLogEntry) error {
	_, err := db.conn.Exec(`
		INSERT INTO audit_log (id, timestamp, user_id, user_email, action, resource_type, resource_id, detail, ip_address)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.Timestamp, entry.UserID, entry.UserEmail, entry.Action, entry.ResourceType, entry.ResourceID, entry.Detail, entry.IPAddress)
	return err
}

// QueryAuditLog retrieves paginated audit log entries with optional filters
func (db *DB) QueryAuditLog(page, perPage int, action, resourceType string) ([]AuditLogEntry, int64, error) {
	where := "1=1"
	var args []interface{}

	if action != "" {
		where += " AND action = ?"
		args = append(args, action)
	}
	if resourceType != "" {
		where += " AND resource_type = ?"
		args = append(args, resourceType)
	}

	var total int64
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	db.conn.QueryRow("SELECT COUNT(*) FROM audit_log WHERE "+where, countArgs...).Scan(&total)

	offset := (page - 1) * perPage
	args = append(args, perPage, offset)
	rows, err := db.conn.Query(
		"SELECT id, timestamp, user_id, user_email, action, resource_type, COALESCE(resource_id, ''), COALESCE(detail, ''), COALESCE(ip_address, '') FROM audit_log WHERE "+where+" ORDER BY timestamp DESC LIMIT ? OFFSET ?",
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.UserID, &e.UserEmail, &e.Action, &e.ResourceType, &e.ResourceID, &e.Detail, &e.IPAddress); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []AuditLogEntry{}
	}

	return entries, total, nil
}

// CleanupOldData removes data older than retentionDays
func (db *DB) CleanupOldData(retentionDays int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -retentionDays).UnixMilli()

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.Exec("DELETE FROM events WHERE timestamp < ?", cutoff)
	tx.Exec("DELETE FROM performance WHERE timestamp < ?", cutoff)
	tx.Exec("DELETE FROM errors WHERE timestamp < ?", cutoff)

	return tx.Commit()
}

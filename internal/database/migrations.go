package database

import (
	"fmt"
)

// Migrate runs database migrations
func (db *DB) Migrate() error {
	// Create migrations table
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY,
			version INTEGER UNIQUE NOT NULL,
			applied_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	var currentVersion int
	row := db.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations")
	row.Scan(&currentVersion)

	// Run migrations
	migrations := []struct {
		version int
		sql     string
	}{
		{
			version: 1,
			sql: `
				-- Events table (pageviews, custom events, clicks, scroll, identify)
				CREATE TABLE IF NOT EXISTS events (
					id TEXT PRIMARY KEY,
					timestamp INTEGER NOT NULL,
					event_type TEXT NOT NULL,
					event_name TEXT,
					session_id TEXT NOT NULL,
					visitor_hash TEXT NOT NULL,
					user_id TEXT,
					domain TEXT NOT NULL,
					url TEXT NOT NULL,
					path TEXT NOT NULL,
					page_title TEXT,
					referrer_url TEXT,
					referrer_type TEXT,
					utm_source TEXT,
					utm_medium TEXT,
					utm_campaign TEXT,
					geo_country TEXT,
					geo_city TEXT,
					geo_region TEXT,
					browser_name TEXT,
					os_name TEXT,
					device_type TEXT,
					is_bot INTEGER DEFAULT 0,
					props TEXT DEFAULT '{}'
				);

				-- Indexes for events
				CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
				CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id);
				CREATE INDEX IF NOT EXISTS idx_events_visitor ON events(visitor_hash);
				CREATE INDEX IF NOT EXISTS idx_events_domain ON events(domain);
				CREATE INDEX IF NOT EXISTS idx_events_path ON events(path);
				CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
				CREATE INDEX IF NOT EXISTS idx_events_country ON events(geo_country);
			`,
		},
		{
			version: 2,
			sql: `
				-- Performance table (Core Web Vitals)
				CREATE TABLE IF NOT EXISTS performance (
					id TEXT PRIMARY KEY,
					timestamp INTEGER NOT NULL,
					session_id TEXT NOT NULL,
					visitor_hash TEXT NOT NULL,
					domain TEXT NOT NULL,
					url TEXT NOT NULL,
					path TEXT NOT NULL,
					lcp REAL,
					cls REAL,
					fcp REAL,
					ttfb REAL,
					inp REAL,
					page_load_time REAL,
					device_type TEXT,
					connection_type TEXT,
					geo_country TEXT
				);

				-- Indexes for performance
				CREATE INDEX IF NOT EXISTS idx_perf_timestamp ON performance(timestamp);
				CREATE INDEX IF NOT EXISTS idx_perf_session ON performance(session_id);
				CREATE INDEX IF NOT EXISTS idx_perf_path ON performance(path);
			`,
		},
		{
			version: 3,
			sql: `
				-- Errors table (JS errors, resource failures)
				CREATE TABLE IF NOT EXISTS errors (
					id TEXT PRIMARY KEY,
					timestamp INTEGER NOT NULL,
					session_id TEXT NOT NULL,
					visitor_hash TEXT NOT NULL,
					domain TEXT NOT NULL,
					url TEXT NOT NULL,
					path TEXT NOT NULL,
					error_type TEXT NOT NULL,
					error_message TEXT NOT NULL,
					error_stack TEXT,
					error_hash TEXT NOT NULL,
					script_url TEXT,
					line_number INTEGER,
					column_number INTEGER,
					browser_name TEXT,
					geo_country TEXT
				);

				-- Indexes for errors
				CREATE INDEX IF NOT EXISTS idx_errors_timestamp ON errors(timestamp);
				CREATE INDEX IF NOT EXISTS idx_errors_hash ON errors(error_hash);
				CREATE INDEX IF NOT EXISTS idx_errors_type ON errors(error_type);
				CREATE INDEX IF NOT EXISTS idx_errors_session ON errors(session_id);
			`,
		},
		{
			version: 4,
			sql: `
				-- Settings table
				CREATE TABLE IF NOT EXISTS settings (
					key TEXT PRIMARY KEY,
					value TEXT NOT NULL,
					updated_at INTEGER NOT NULL
				);

				-- Insert default settings
				INSERT OR IGNORE INTO settings (key, value, updated_at) VALUES
					('track_performance', 'true', strftime('%s', 'now') * 1000),
					('track_errors', 'true', strftime('%s', 'now') * 1000),
					('session_timeout_minutes', '30', strftime('%s', 'now') * 1000),
					('respect_dnt', 'true', strftime('%s', 'now') * 1000);
			`,
		},
		{
			version: 5,
			sql: `
				-- Users table (for auth)
				CREATE TABLE IF NOT EXISTS users (
					id TEXT PRIMARY KEY,
					email TEXT UNIQUE NOT NULL,
					password_hash TEXT NOT NULL,
					name TEXT,
					role TEXT DEFAULT 'viewer',
					created_at INTEGER NOT NULL,
					updated_at INTEGER NOT NULL
				);

				-- Sessions table (for auth)
				CREATE TABLE IF NOT EXISTS sessions (
					id TEXT PRIMARY KEY,
					user_id TEXT NOT NULL,
					expires_at INTEGER NOT NULL,
					created_at INTEGER NOT NULL,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				);

				CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
				CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
			`,
		},
		{
			version: 6,
			sql: `
				-- Domains table (for multi-site tracking)
				CREATE TABLE IF NOT EXISTS domains (
					id TEXT PRIMARY KEY,
					name TEXT NOT NULL,
					domain TEXT UNIQUE NOT NULL,
					created_by TEXT,
					created_at INTEGER NOT NULL,
					is_active INTEGER DEFAULT 1,
					FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
				);

				CREATE INDEX IF NOT EXISTS idx_domains_domain ON domains(domain);
				CREATE INDEX IF NOT EXISTS idx_domains_active ON domains(is_active);

				-- Add setup_complete setting
				INSERT OR IGNORE INTO settings (key, value, updated_at) VALUES
					('setup_complete', 'false', strftime('%s', 'now') * 1000);
			`,
		},
		{
			version: 7,
			sql: `
				-- Add site_id to domains for tracking script authentication
				ALTER TABLE domains ADD COLUMN site_id TEXT;

				-- Generate site_id for existing domains
				UPDATE domains SET site_id = 'site_' || lower(hex(randomblob(8))) WHERE site_id IS NULL;

				-- Create unique index on site_id
				CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_site_id ON domains(site_id);
			`,
		},
		{
			version: 8,
			sql: `
				-- Bot detection columns
				ALTER TABLE events ADD COLUMN bot_score INTEGER DEFAULT 0;
				ALTER TABLE events ADD COLUMN bot_signals TEXT DEFAULT '[]';
				ALTER TABLE events ADD COLUMN bot_category TEXT DEFAULT 'human';

				-- Behavioral flags
				ALTER TABLE events ADD COLUMN has_scroll INTEGER DEFAULT 0;
				ALTER TABLE events ADD COLUMN has_mouse_move INTEGER DEFAULT 0;
				ALTER TABLE events ADD COLUMN has_click INTEGER DEFAULT 0;
				ALTER TABLE events ADD COLUMN has_touch INTEGER DEFAULT 0;

				-- Click tracking for fraud detection
				ALTER TABLE events ADD COLUMN click_x INTEGER;
				ALTER TABLE events ADD COLUMN click_y INTEGER;
				ALTER TABLE events ADD COLUMN page_duration INTEGER;

				-- IP classification
				ALTER TABLE events ADD COLUMN datacenter_ip INTEGER DEFAULT 0;
				ALTER TABLE events ADD COLUMN ip_hash TEXT;

				-- Indexes for bot filtering
				CREATE INDEX IF NOT EXISTS idx_events_bot_category ON events(bot_category);
				CREATE INDEX IF NOT EXISTS idx_events_bot_score ON events(bot_score);
			`,
		},
		{
			version: 9,
			sql: `
				-- Campaigns table for ad fraud detection
				CREATE TABLE IF NOT EXISTS campaigns (
					id TEXT PRIMARY KEY,
					name TEXT NOT NULL,
					utm_source TEXT,
					utm_medium TEXT,
					utm_campaign TEXT,
					cpc REAL DEFAULT 0,
					cpm REAL DEFAULT 0,
					budget REAL DEFAULT 0,
					start_date INTEGER,
					end_date INTEGER,
					created_at INTEGER NOT NULL
				);

				-- Visitor sessions table (materialized)
				CREATE TABLE IF NOT EXISTS visitor_sessions (
					id TEXT PRIMARY KEY,
					session_id TEXT UNIQUE,
					visitor_hash TEXT,
					domain TEXT,
					start_time INTEGER,
					end_time INTEGER,
					duration INTEGER,
					pageviews INTEGER,
					entry_url TEXT,
					exit_url TEXT,
					is_bounce INTEGER,
					bot_score INTEGER,
					bot_category TEXT
				);

				-- Indexes for campaigns and sessions
				CREATE INDEX IF NOT EXISTS idx_campaigns_utm ON campaigns(utm_source, utm_medium, utm_campaign);
				CREATE INDEX IF NOT EXISTS idx_visitor_sessions_session ON visitor_sessions(session_id);
				CREATE INDEX IF NOT EXISTS idx_visitor_sessions_domain ON visitor_sessions(domain);
				CREATE INDEX IF NOT EXISTS idx_visitor_sessions_bot ON visitor_sessions(bot_category);
			`,
		},
		{
			version: 10,
			sql: `
				-- Add geo coordinates for map plotting
				ALTER TABLE events ADD COLUMN geo_latitude REAL;
				ALTER TABLE events ADD COLUMN geo_longitude REAL;

				-- Index for geo queries
				CREATE INDEX IF NOT EXISTS idx_events_geo ON events(geo_latitude, geo_longitude);
			`,
		},
		{
			version: 11,
			sql: `
				-- Add settings for configuration (replacing .env)
				INSERT OR IGNORE INTO settings (key, value, updated_at) VALUES
					('secret_key', '', strftime('%s', 'now') * 1000),
					('maxmind_account_id', '', strftime('%s', 'now') * 1000),
					('maxmind_license_key', '', strftime('%s', 'now') * 1000),
					('geoip_path', './data/GeoLite2-City.mmdb', strftime('%s', 'now') * 1000),
					('geoip_auto_update', 'false', strftime('%s', 'now') * 1000),
					('geoip_last_updated', '', strftime('%s', 'now') * 1000),
					('allowed_origins', '*', strftime('%s', 'now') * 1000),
					('listen_addr', ':3456', strftime('%s', 'now') * 1000);
			`,
		},
		{
			version: 12,
			sql: `
				-- Composite index for the most common stat query pattern:
				-- WHERE timestamp >= ? AND timestamp <= ? AND is_bot = 0 AND domain = ?
				CREATE INDEX IF NOT EXISTS idx_events_ts_domain_bot
					ON events(timestamp, domain, is_bot);

				-- Composite index for pageview stats (adds event_type)
				CREATE INDEX IF NOT EXISTS idx_events_ts_type_bot
					ON events(timestamp, event_type, is_bot);

				-- Composite for visitor_sessions queries
				CREATE INDEX IF NOT EXISTS idx_vsessions_domain_start
					ON visitor_sessions(domain, start_time);
			`,
		},
		{
			version: 13,
			sql: `
				-- Consent tables
				CREATE TABLE IF NOT EXISTS consent_configs (
					id TEXT PRIMARY KEY,
					domain_id TEXT NOT NULL,
					version INTEGER NOT NULL DEFAULT 1,
					is_active INTEGER NOT NULL DEFAULT 1,
					categories TEXT NOT NULL DEFAULT '[]',
					appearance TEXT NOT NULL DEFAULT '{}',
					translations TEXT NOT NULL DEFAULT '{}',
					cookie_name TEXT NOT NULL DEFAULT 'etiquetta_consent',
					cookie_expiry_days INTEGER NOT NULL DEFAULT 365,
					auto_language INTEGER NOT NULL DEFAULT 1,
					geo_targeting TEXT NOT NULL DEFAULT '[]',
					created_at INTEGER NOT NULL,
					updated_at INTEGER NOT NULL,
					FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
				);
				CREATE INDEX idx_consent_configs_domain ON consent_configs(domain_id);
				CREATE UNIQUE INDEX idx_consent_configs_version ON consent_configs(domain_id, version);

				CREATE TABLE IF NOT EXISTS consent_records (
					id TEXT PRIMARY KEY,
					domain_id TEXT NOT NULL,
					visitor_hash TEXT NOT NULL,
					ip_hash TEXT,
					categories TEXT NOT NULL DEFAULT '{}',
					config_version INTEGER NOT NULL,
					action TEXT NOT NULL,
					user_agent TEXT,
					geo_country TEXT,
					timestamp INTEGER NOT NULL,
					FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
				);
				CREATE INDEX idx_consent_records_domain_ts ON consent_records(domain_id, timestamp);
				CREATE INDEX idx_consent_records_visitor ON consent_records(visitor_hash);

				-- Tag Manager tables
				CREATE TABLE IF NOT EXISTS tm_containers (
					id TEXT PRIMARY KEY,
					domain_id TEXT NOT NULL,
					name TEXT NOT NULL,
					published_version INTEGER DEFAULT 0,
					draft_version INTEGER DEFAULT 1,
					published_at INTEGER,
					published_by TEXT,
					created_at INTEGER NOT NULL,
					updated_at INTEGER NOT NULL,
					FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE,
					UNIQUE(domain_id)
				);

				CREATE TABLE IF NOT EXISTS tm_tags (
					id TEXT PRIMARY KEY,
					container_id TEXT NOT NULL,
					name TEXT NOT NULL,
					tag_type TEXT NOT NULL,
					config TEXT NOT NULL DEFAULT '{}',
					consent_category TEXT NOT NULL DEFAULT 'marketing',
					priority INTEGER DEFAULT 0,
					is_enabled INTEGER DEFAULT 1,
					version INTEGER DEFAULT 1,
					created_at INTEGER NOT NULL,
					updated_at INTEGER NOT NULL,
					FOREIGN KEY (container_id) REFERENCES tm_containers(id) ON DELETE CASCADE
				);
				CREATE INDEX idx_tm_tags_container ON tm_tags(container_id);

				CREATE TABLE IF NOT EXISTS tm_triggers (
					id TEXT PRIMARY KEY,
					container_id TEXT NOT NULL,
					name TEXT NOT NULL,
					trigger_type TEXT NOT NULL,
					config TEXT NOT NULL DEFAULT '{}',
					created_at INTEGER NOT NULL,
					updated_at INTEGER NOT NULL,
					FOREIGN KEY (container_id) REFERENCES tm_containers(id) ON DELETE CASCADE
				);
				CREATE INDEX idx_tm_triggers_container ON tm_triggers(container_id);

				CREATE TABLE IF NOT EXISTS tm_tag_triggers (
					tag_id TEXT NOT NULL,
					trigger_id TEXT NOT NULL,
					is_exception INTEGER DEFAULT 0,
					PRIMARY KEY (tag_id, trigger_id),
					FOREIGN KEY (tag_id) REFERENCES tm_tags(id) ON DELETE CASCADE,
					FOREIGN KEY (trigger_id) REFERENCES tm_triggers(id) ON DELETE CASCADE
				);

				CREATE TABLE IF NOT EXISTS tm_variables (
					id TEXT PRIMARY KEY,
					container_id TEXT NOT NULL,
					name TEXT NOT NULL,
					variable_type TEXT NOT NULL,
					config TEXT NOT NULL DEFAULT '{}',
					created_at INTEGER NOT NULL,
					updated_at INTEGER NOT NULL,
					FOREIGN KEY (container_id) REFERENCES tm_containers(id) ON DELETE CASCADE
				);
				CREATE INDEX idx_tm_variables_container ON tm_variables(container_id);

				CREATE TABLE IF NOT EXISTS tm_snapshots (
					id TEXT PRIMARY KEY,
					container_id TEXT NOT NULL,
					version INTEGER NOT NULL,
					snapshot TEXT NOT NULL,
					published_by TEXT,
					published_at INTEGER NOT NULL,
					FOREIGN KEY (container_id) REFERENCES tm_containers(id) ON DELETE CASCADE,
					UNIQUE(container_id, version)
				);
			`,
		},
		{
			version: 14,
			sql: `
				-- Admin audit log for GDPR compliance
				CREATE TABLE IF NOT EXISTS audit_log (
					id TEXT PRIMARY KEY,
					timestamp INTEGER NOT NULL,
					user_id TEXT NOT NULL,
					user_email TEXT NOT NULL,
					action TEXT NOT NULL,
					resource_type TEXT NOT NULL,
					resource_id TEXT,
					detail TEXT,
					ip_address TEXT
				);

				CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp);
				CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_id);
				CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
				CREATE INDEX IF NOT EXISTS idx_audit_log_resource ON audit_log(resource_type, resource_id);
			`,
		},
	}

	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}

		tx, err := db.conn.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", m.version, err)
		}

		_, err = tx.Exec(m.sql)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to run migration %d: %w", m.version, err)
		}

		_, err = tx.Exec("INSERT INTO migrations (version, applied_at) VALUES (?, strftime('%s', 'now') * 1000)", m.version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", m.version, err)
		}
	}

	return nil
}

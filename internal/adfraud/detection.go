package adfraud

import (
	"database/sql"
	"fmt"
)

// FraudSignal represents a detected fraud indicator
type FraudSignal struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Count       int64  `json:"count"`
	Severity    string `json:"severity"` // low, medium, high
}

// FraudSummary contains overall fraud statistics
type FraudSummary struct {
	TotalClicks      int64         `json:"total_clicks"`
	BotClicks        int64         `json:"bot_clicks"`
	SuspiciousClicks int64         `json:"suspicious_clicks"`
	HumanClicks      int64         `json:"human_clicks"`
	InvalidRate      float64       `json:"invalid_rate"`       // (bot+suspicious)/total, 0-1
	DatacenterClicks int64         `json:"datacenter_clicks"`
	TotalSpend       float64       `json:"total_spend"`
	WastedSpend      float64       `json:"wasted_spend"`
	RealCPC          *float64      `json:"real_cpc"`
	RealCPA          *float64      `json:"real_cpa"`
	Conversions      int64         `json:"conversions"`
	ConversionEvent  string        `json:"conversion_event"`
	Signals          []FraudSignal `json:"signals"`
}

// Detector handles fraud detection operations
type Detector struct {
	db *sql.DB
}

// NewDetector creates a new fraud detector
func NewDetector(db *sql.DB) *Detector {
	return &Detector{db: db}
}

// helper to build domain condition
func domainClause(alias string, domain string, args *[]interface{}) string {
	if domain == "" {
		return ""
	}
	*args = append(*args, domain)
	if alias != "" {
		return fmt.Sprintf(" AND %s.domain = ?", alias)
	}
	return " AND domain = ?"
}

// GetFraudSummary returns an overview of detected fraud
func (d *Detector) GetFraudSummary(domain string, startMs, endMs int64, conversionEvent string) (*FraudSummary, error) {
	summary := &FraudSummary{
		Signals:         make([]FraudSignal, 0),
		ConversionEvent: conversionEvent,
	}

	// Get campaign traffic counts by bot category (pageviews with UTM = ad clicks)
	query := `
		SELECT
			bot_category,
			COUNT(*) as clicks
		FROM events
		WHERE timestamp >= ? AND timestamp <= ?
			AND event_type = 'pageview'
			AND utm_source IS NOT NULL
	`
	args := []interface{}{startMs, endMs}
	query += domainClause("", domain, &args)
	query += " GROUP BY bot_category"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var clicks int64
		rows.Scan(&category, &clicks)

		summary.TotalClicks += clicks
		switch category {
		case "human":
			summary.HumanClicks = clicks
		case "bad_bot":
			summary.BotClicks += clicks
		case "suspicious":
			summary.SuspiciousClicks = clicks
		case "good_bot":
			summary.BotClicks += clicks
		}
	}

	if summary.TotalClicks > 0 {
		summary.InvalidRate = float64(summary.BotClicks+summary.SuspiciousClicks) / float64(summary.TotalClicks)
	}

	// Datacenter clicks (campaign traffic from datacenter IPs)
	dcQuery := `SELECT COUNT(*) FROM events
		WHERE timestamp >= ? AND timestamp <= ?
			AND datacenter_ip = 1 AND utm_source IS NOT NULL AND event_type = 'pageview'`
	dcArgs := []interface{}{startMs, endMs}
	dcQuery += domainClause("", domain, &dcArgs)
	d.db.QueryRow(dcQuery, dcArgs...).Scan(&summary.DatacenterClicks)

	// Detect fraud patterns (existing 3 + new 4)
	summary.Signals = append(summary.Signals, d.detectClickWithoutImpression(domain, startMs, endMs)...)
	summary.Signals = append(summary.Signals, d.detectCoordinateClustering(domain, startMs, endMs)...)
	summary.Signals = append(summary.Signals, d.detectEngagementMismatch(domain, startMs, endMs)...)
	summary.Signals = append(summary.Signals, d.detectDatacenterCampaignCorrelation(domain, startMs, endMs)...)
	summary.Signals = append(summary.Signals, d.detectRepeatClicks(domain, startMs, endMs)...)
	summary.Signals = append(summary.Signals, d.detectTemporalBursts(domain, startMs, endMs)...)
	if conversionEvent != "" {
		summary.Signals = append(summary.Signals, d.detectConversionRatioAnomaly(domain, startMs, endMs, conversionEvent)...)
	}

	// Calculate spend and waste
	summary.TotalSpend = d.calculateTotalSpend(domain, startMs, endMs)
	summary.WastedSpend = d.calculateWastedSpend(domain, startMs, endMs)

	// Real CPC = total_spend / human_clicks
	if summary.HumanClicks > 0 && summary.TotalSpend > 0 {
		cpc := summary.TotalSpend / float64(summary.HumanClicks)
		summary.RealCPC = &cpc
	}

	// Real CPA = total_spend / conversions (only if conversion event is configured)
	if conversionEvent != "" {
		convQuery := `SELECT COUNT(*) FROM events
			WHERE timestamp >= ? AND timestamp <= ?
				AND event_type = 'custom' AND event_name = ?
				AND bot_category = 'human'`
		convArgs := []interface{}{startMs, endMs, conversionEvent}
		convQuery += domainClause("", domain, &convArgs)
		d.db.QueryRow(convQuery, convArgs...).Scan(&summary.Conversions)

		if summary.Conversions > 0 && summary.TotalSpend > 0 {
			cpa := summary.TotalSpend / float64(summary.Conversions)
			summary.RealCPA = &cpa
		}
	}

	return summary, nil
}

// detectClickWithoutImpression finds clicks that don't have a prior pageview in the session
func (d *Detector) detectClickWithoutImpression(domain string, startMs, endMs int64) []FraudSignal {
	query := `
		SELECT COUNT(DISTINCT e.session_id) as orphan_clicks
		FROM events e
		WHERE e.timestamp >= ? AND e.timestamp <= ?
			AND e.event_type = 'click'
			AND e.utm_source IS NOT NULL
	`
	args := []interface{}{startMs, endMs}
	query += domainClause("e", domain, &args)
	query += `
			AND NOT EXISTS (
				SELECT 1 FROM events e2
				WHERE e2.session_id = e.session_id
					AND e2.event_type = 'pageview'
					AND e2.timestamp <= e.timestamp
			)`

	var count int64
	d.db.QueryRow(query, args...).Scan(&count)

	if count > 0 {
		return []FraudSignal{{
			Type:        "click_without_impression",
			Description: "Clicks from campaign traffic without prior page impression",
			Count:       count,
			Severity:    "high",
		}}
	}
	return nil
}

// detectCoordinateClustering finds suspiciously clustered click coordinates
func (d *Detector) detectCoordinateClustering(domain string, startMs, endMs int64) []FraudSignal {
	// Subquery for total count
	subArgs := []interface{}{startMs, endMs}
	subQuery := "SELECT COUNT(*) FROM events WHERE timestamp >= ? AND timestamp <= ? AND event_type = 'click' AND click_x IS NOT NULL"
	subQuery += domainClause("", domain, &subArgs)

	query := fmt.Sprintf(`
		SELECT click_x, click_y, COUNT(*) as click_count,
			CAST(COUNT(*) AS DOUBLE) / (%s) * 100 as pct
		FROM events
		WHERE timestamp >= ? AND timestamp <= ?
			AND event_type = 'click'
			AND click_x IS NOT NULL
	`, subQuery)

	args := append(subArgs, startMs, endMs)
	query += domainClause("", domain, &args)
	query += " GROUP BY click_x, click_y HAVING pct > 10 LIMIT 5"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var signals []FraudSignal
	for rows.Next() {
		var x, y int
		var count int64
		var pct float64
		rows.Scan(&x, &y, &count, &pct)

		signals = append(signals, FraudSignal{
			Type:        "coordinate_clustering",
			Description: "High concentration of clicks at specific coordinates",
			Count:       count,
			Severity:    "medium",
		})
	}
	return signals
}

// detectEngagementMismatch finds sessions with clicks but no engagement
func (d *Detector) detectEngagementMismatch(domain string, startMs, endMs int64) []FraudSignal {
	// Build subqueries for sessions with engagement
	subArgs1 := []interface{}{startMs, endMs}
	sub1 := "SELECT session_id FROM events WHERE timestamp >= ? AND timestamp <= ? AND has_scroll = 1"
	sub1 += domainClause("", domain, &subArgs1)

	subArgs2 := []interface{}{startMs, endMs}
	sub2 := "SELECT session_id FROM events WHERE timestamp >= ? AND timestamp <= ? AND page_duration > 5000"
	sub2 += domainClause("", domain, &subArgs2)

	query := fmt.Sprintf(`
		SELECT COUNT(DISTINCT session_id) as count
		FROM events
		WHERE timestamp >= ? AND timestamp <= ?
			AND event_type = 'click'
			AND utm_source IS NOT NULL
			AND session_id NOT IN (%s UNION %s)
	`, sub1, sub2)

	args := []interface{}{startMs, endMs}
	query += domainClause("", domain, &args)
	args = append(args, subArgs1...)
	args = append(args, subArgs2...)

	// DuckDB needs positional params in order, so rebuild properly
	// Simpler approach: single query
	simpleQuery := `
		SELECT COUNT(DISTINCT session_id) as count
		FROM events
		WHERE timestamp >= ? AND timestamp <= ?
			AND event_type = 'click'
			AND utm_source IS NOT NULL
	`
	simpleArgs := []interface{}{startMs, endMs}
	simpleQuery += domainClause("", domain, &simpleArgs)
	simpleQuery += `
			AND session_id NOT IN (
				SELECT DISTINCT session_id FROM events
				WHERE timestamp >= ? AND timestamp <= ?
					AND (has_scroll = 1 OR page_duration > 5000)
	`
	simpleArgs = append(simpleArgs, startMs, endMs)
	simpleQuery += domainClause("", domain, &simpleArgs)
	simpleQuery += ")"

	var count int64
	d.db.QueryRow(simpleQuery, simpleArgs...).Scan(&count)

	if count > 0 {
		return []FraudSignal{{
			Type:        "engagement_mismatch",
			Description: "Campaign clicks with no scroll or meaningful time on site",
			Count:       count,
			Severity:    "medium",
		}}
	}
	return nil
}

// detectDatacenterCampaignCorrelation finds datacenter IPs in campaign traffic
func (d *Detector) detectDatacenterCampaignCorrelation(domain string, startMs, endMs int64) []FraudSignal {
	query := `SELECT COUNT(*) FROM events
		WHERE timestamp >= ? AND timestamp <= ?
			AND datacenter_ip = 1 AND utm_source IS NOT NULL AND event_type = 'pageview'`
	args := []interface{}{startMs, endMs}
	query += domainClause("", domain, &args)

	var count int64
	d.db.QueryRow(query, args...).Scan(&count)

	if count > 0 {
		return []FraudSignal{{
			Type:        "datacenter_campaign",
			Description: "Campaign traffic originating from datacenter IPs (likely bots)",
			Count:       count,
			Severity:    "high",
		}}
	}
	return nil
}

// detectRepeatClicks finds visitors hitting the same campaign repeatedly
func (d *Detector) detectRepeatClicks(domain string, startMs, endMs int64) []FraudSignal {
	query := `SELECT COUNT(*) FROM (
		SELECT visitor_hash, utm_campaign, COUNT(*) as visits
		FROM events
		WHERE timestamp >= ? AND timestamp <= ?
			AND utm_source IS NOT NULL AND event_type = 'pageview'
	`
	args := []interface{}{startMs, endMs}
	query += domainClause("", domain, &args)
	query += " GROUP BY visitor_hash, utm_campaign HAVING visits > 5)"

	var count int64
	d.db.QueryRow(query, args...).Scan(&count)

	if count > 0 {
		return []FraudSignal{{
			Type:        "repeat_clicks",
			Description: "Same visitor clicking the same campaign more than 5 times",
			Count:       count,
			Severity:    "medium",
		}}
	}
	return nil
}

// detectTemporalBursts finds campaign traffic spikes from a single source
func (d *Detector) detectTemporalBursts(domain string, startMs, endMs int64) []FraudSignal {
	query := `SELECT COUNT(*) FROM (
		SELECT utm_source,
			strftime(to_timestamp(timestamp / 1000)::TIMESTAMP, '%Y-%m-%d %H') as hour_window,
			COUNT(*) as visits
		FROM events
		WHERE timestamp >= ? AND timestamp <= ?
			AND utm_source IS NOT NULL AND event_type = 'pageview'
	`
	args := []interface{}{startMs, endMs}
	query += domainClause("", domain, &args)
	query += " GROUP BY utm_source, hour_window HAVING visits > 100)"

	var count int64
	d.db.QueryRow(query, args...).Scan(&count)

	if count > 0 {
		return []FraudSignal{{
			Type:        "temporal_burst",
			Description: "Single traffic source sending 100+ visits in one hour",
			Count:       count,
			Severity:    "high",
		}}
	}
	return nil
}

// detectConversionRatioAnomaly finds sources with high traffic but zero conversions
func (d *Detector) detectConversionRatioAnomaly(domain string, startMs, endMs int64, conversionEvent string) []FraudSignal {
	query := `SELECT COUNT(*) FROM (
		SELECT utm_source,
			SUM(CASE WHEN event_type = 'pageview' THEN 1 ELSE 0 END) as campaign_visits,
			SUM(CASE WHEN event_type = 'custom' AND event_name = ? THEN 1 ELSE 0 END) as conversion_count
		FROM events
		WHERE timestamp >= ? AND timestamp <= ?
			AND bot_category = 'human' AND utm_source IS NOT NULL
	`
	args := []interface{}{conversionEvent, startMs, endMs}
	query += domainClause("", domain, &args)
	query += " GROUP BY utm_source HAVING campaign_visits > 50) WHERE conversion_count = 0"

	var count int64
	d.db.QueryRow(query, args...).Scan(&count)

	if count > 0 {
		return []FraudSignal{{
			Type:        "conversion_ratio_anomaly",
			Description: "Traffic sources with 50+ visits but zero conversions",
			Count:       count,
			Severity:    "medium",
		}}
	}
	return nil
}

// calculateTotalSpend computes total ad spend from campaigns + ad_spend_daily
func (d *Detector) calculateTotalSpend(domain string, startMs, endMs int64) float64 {
	var total float64

	// From ad_spend_daily (synced from ad platforms)
	startDate := fmt.Sprintf("%d", startMs/1000) // seconds for date comparison
	endDate := fmt.Sprintf("%d", endMs/1000)
	_ = startDate
	_ = endDate

	// ad_spend_daily stores date as string 'YYYY-MM-DD', query by converting timestamps
	adQuery := `SELECT COALESCE(SUM(cost_micros), 0) / 1000000.0 FROM ad_spend_daily
		WHERE date >= strftime(to_timestamp(? / 1000)::TIMESTAMP, '%Y-%m-%d')
			AND date <= strftime(to_timestamp(? / 1000)::TIMESTAMP, '%Y-%m-%d')`
	adArgs := []interface{}{startMs, endMs}

	var adSpend float64
	err := d.db.QueryRow(adQuery, adArgs...).Scan(&adSpend)
	if err == nil {
		total += adSpend
	}

	// From manually created campaigns (CPC × matching clicks)
	campQuery := `
		SELECT COALESCE(SUM(c.cpc), 0) as total_cpc_spend
		FROM events e
		JOIN campaigns c ON (
			(c.utm_source IS NULL OR c.utm_source = e.utm_source)
			AND (c.utm_medium IS NULL OR c.utm_medium = e.utm_medium)
			AND (c.utm_campaign IS NULL OR c.utm_campaign = e.utm_campaign)
		)
		WHERE e.timestamp >= ? AND e.timestamp <= ?
			AND e.event_type = 'pageview'
			AND e.utm_source IS NOT NULL
	`
	campArgs := []interface{}{startMs, endMs}
	campQuery += domainClause("e", domain, &campArgs)

	var campSpend float64
	err = d.db.QueryRow(campQuery, campArgs...).Scan(&campSpend)
	if err == nil {
		total += campSpend / 100 // cents to dollars
	}

	return total
}

// calculateWastedSpend estimates money wasted on bot/fraudulent clicks
func (d *Detector) calculateWastedSpend(domain string, startMs, endMs int64) float64 {
	query := `
		SELECT COALESCE(SUM(c.cpc), 0) as waste
		FROM events e
		JOIN campaigns c ON (
			(c.utm_source IS NULL OR c.utm_source = e.utm_source)
			AND (c.utm_medium IS NULL OR c.utm_medium = e.utm_medium)
			AND (c.utm_campaign IS NULL OR c.utm_campaign = e.utm_campaign)
		)
		WHERE e.timestamp >= ? AND e.timestamp <= ?
			AND e.event_type = 'pageview'
			AND e.bot_score >= 50
			AND e.utm_source IS NOT NULL
	`
	args := []interface{}{startMs, endMs}
	query += domainClause("e", domain, &args)

	var waste float64
	d.db.QueryRow(query, args...).Scan(&waste)
	return waste / 100 // Convert cents to dollars
}

# Etiquetta Documentation

Complete reference for integrating Etiquetta analytics into your website.

## Table of Contents

- [Getting Started](#getting-started)
- [Automatic Tracking](#automatic-tracking)
- [Custom Events](#custom-events)
- [JavaScript API Reference](#javascript-api-reference)
- [Configuration](#configuration)
- [Privacy & Consent](#privacy--consent)
- [Analytics API](#analytics-api)
- [Event Types Reference](#event-types-reference)
- [Rate Limits & Batching](#rate-limits--batching)

---

## Getting Started

### 1. Add Your Domain

1. Log in to your Etiquetta instance
2. Go to **Settings > Domains**
3. Click **Add Domain** and enter your site name and domain (e.g. `example.com`)
4. Copy the tracking snippet

### 2. Install the Tracking Script

Add the snippet to your website's `<head>`:

```html
<script defer data-site="YOUR_SITE_ID" src="https://your-etiquetta-instance.com/s.js"></script>
```

- `defer` ensures the script loads without blocking page rendering
- `data-site` identifies your domain — get it from **Settings > Domains > Snippet**
- The script automatically loads the consent banner and tag manager if configured

That's it. Pageviews, scroll depth, outbound clicks, engagement time, and bot detection are all tracked automatically.

### Multi-domain Setup

You can track multiple websites from a single Etiquetta instance. Each domain gets its own `data-site` ID. Install the snippet on each site with its corresponding ID.

---

## Automatic Tracking

The tracker collects the following data out of the box — no code required.

### Pageviews

Every page load fires a `pageview` event with:

| Field | Description |
|-------|-------------|
| `url` | Full page URL |
| `path` | URL pathname (e.g. `/blog/my-post`) |
| `referrer_url` | Where the visitor came from |
| `page_title` | Document title |
| `utm_source` | UTM source parameter |
| `utm_medium` | UTM medium parameter |
| `utm_campaign` | UTM campaign parameter |

**SPA support**: The tracker automatically detects client-side navigation via `history.pushState`, `history.replaceState`, and `popstate` events. No extra configuration needed for React, Vue, Next.js, or any SPA framework.

### Scroll Depth

Tracks how far visitors scroll. Events fire at four milestones:

| Milestone | Event Name |
|-----------|------------|
| 25% | `scroll_25` |
| 50% | `scroll_50` |
| 75% | `scroll_75` |
| 100% | `scroll_100` |

Each milestone fires only once per page. Milestones reset on SPA navigation.

### Outbound Link Clicks

When a visitor clicks a link to an external domain, an event is recorded with:

- **Event type**: `click`
- **Event name**: `outbound`
- **Props**: `{ "target": "https://external-site.com/page" }`

This works via event delegation — no need to annotate links.

### Engagement Time

When a visitor leaves the page, an engagement event is sent with:

| Property | Description |
|----------|-------------|
| `total_time_ms` | Total time on page (ms) |
| `visible_time_ms` | Time the page was actually visible (ms) |
| `scroll_depth` | Maximum scroll depth percentage |

This accounts for background tabs — `visible_time_ms` only counts time the tab was in the foreground.

### Behavioral Signals

The tracker detects visitor behavior to help distinguish humans from bots:

- **Scroll**: Whether the visitor scrolled
- **Mouse movement**: Whether mouse movement was detected
- **Click**: Whether and where the visitor clicked
- **Touch**: Whether touch events occurred

These signals are included with every event and feed into Etiquetta's bot detection system.

### Core Web Vitals (Pro)

With a Pro or Enterprise license, the tracker automatically collects Core Web Vitals:

| Metric | Description |
|--------|-------------|
| **LCP** | Largest Contentful Paint — loading performance |
| **FCP** | First Contentful Paint — time to first content |
| **CLS** | Cumulative Layout Shift — visual stability |
| **INP** | Interaction to Next Paint — responsiveness |
| **TTFB** | Time to First Byte — server response time |
| `page_load_time` | Total page load time |
| `connection_type` | Network connection type (4g, 3g, etc.) |

Performance data is sent once when the visitor leaves the page.

### JavaScript Error Tracking (Pro)

With a Pro or Enterprise license, the tracker captures:

- **JavaScript errors** (`window.onerror`) — message, file, line, column, stack trace
- **Unhandled promise rejections** — message, stack trace

Errors are deduplicated by fingerprint (hash of message + filename + line + column), so the same error doesn't flood your analytics.

---

## Custom Events

Custom events let you track specific user interactions beyond pageviews.

### Basic Usage

```javascript
// Track a button click
etiquetta.track('button_click', { button: 'signup' })

// Track a form submission
etiquetta.track('form_submit', { form: 'contact', source: 'footer' })

// Track a purchase
etiquetta.track('purchase', { plan: 'pro', amount: 99, currency: 'EUR' })

// Track a file download
etiquetta.track('download', { file: 'whitepaper.pdf', format: 'pdf' })

// Track with no properties
etiquetta.track('cta_viewed')
```

### API

```javascript
etiquetta.track(name, props)
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | `string` | Yes | Event name (e.g. `"signup"`, `"purchase"`) |
| `props` | `object` | No | Key-value properties (values can be strings, numbers, or booleans) |

The `name` must be a non-empty string. The `props` object is serialized to JSON and stored with the event. Keep property names and values concise.

### Practical Examples

#### E-commerce

```javascript
// Product viewed
etiquetta.track('product_view', {
  product_id: 'SKU-123',
  category: 'shoes',
  price: 79.99
})

// Add to cart
etiquetta.track('add_to_cart', {
  product_id: 'SKU-123',
  quantity: 1
})

// Checkout completed
etiquetta.track('checkout', {
  order_id: 'ORD-456',
  total: 79.99,
  items: 1
})
```

#### SaaS

```javascript
// Feature usage
etiquetta.track('feature_used', { feature: 'export', format: 'csv' })

// Onboarding step
etiquetta.track('onboarding_step', { step: 3, step_name: 'invite_team' })

// Upgrade prompt shown
etiquetta.track('upgrade_prompt', { trigger: 'feature_gate', feature: 'api_access' })
```

#### Content

```javascript
// Article read
etiquetta.track('article_read', { slug: 'getting-started', category: 'tutorial' })

// Video play
etiquetta.track('video_play', { video_id: 'intro-tour', duration: 120 })

// Search performed
etiquetta.track('search', { query: 'analytics setup', results: 5 })
```

#### Forms

```javascript
// Attach to a form
document.querySelector('#signup-form').addEventListener('submit', () => {
  etiquetta.track('signup_form_submit', { source: 'homepage' })
})

// Attach to a button
document.querySelector('#cta-button').addEventListener('click', () => {
  etiquetta.track('cta_click', { variant: 'blue', position: 'hero' })
})
```

#### React

```tsx
function SignupButton() {
  return (
    <button onClick={() => etiquetta.track('signup_click', { source: 'nav' })}>
      Sign Up
    </button>
  )
}
```

### Viewing Custom Events

Custom events appear in your dashboard under the **Custom Events** section. Each event name is listed with its total count and unique visitor count for the selected date range.

To filter by custom events, use the stats API:

```
GET /api/stats/events?domain=example.com&days=30
```

---

## JavaScript API Reference

The tracker exposes four methods on `window.etiquetta`:

### `etiquetta.track(name, props?)`

Send a custom event.

```javascript
etiquetta.track('signup', { plan: 'pro' })
```

- `name` (string, required) — Event name
- `props` (object, optional) — Arbitrary key-value properties

### `etiquetta.pageview(opts?)`

Manually trigger a pageview. Useful when you need to override automatic SPA detection.

```javascript
// Track current page
etiquetta.pageview()

// Track a specific URL
etiquetta.pageview({ url: 'https://example.com/virtual-page' })

// Force re-track the same page
etiquetta.pageview({ force: true })
```

- `opts.url` (string, optional) — Override the tracked URL
- `opts.force` (boolean, optional) — Send even if it's the same URL as last pageview

### `etiquetta.flush()`

Force-send all queued events immediately. Events are normally batched and sent every second. Call this if you need to ensure events are sent before navigating away.

```javascript
etiquetta.track('checkout', { order: '123' })
etiquetta.flush()
window.location.href = '/thank-you'
```

### `etiquetta.getVisitorHash()`

Returns the current visitor's fingerprint hash (string). This is a deterministic, cookie-free identifier based on device characteristics (screen size, timezone, language, etc.).

```javascript
const hash = etiquetta.getVisitorHash()
// e.g. "a1b2c3d4e5f6g7h8"
```

---

## Configuration

### Server-Injected Config

The Etiquetta server automatically injects configuration before the tracker script. This controls Pro features based on your license:

```javascript
window.__ETIQUETTA_CONFIG__ = {
  endpoint: "/i",
  trackPerformance: true,  // Core Web Vitals (requires Pro license)
  trackErrors: true,       // Error tracking (requires Pro license)
  respectDNT: true         // Honor Do-Not-Track headers
}
```

You generally don't need to set this manually.

### Custom Configuration

To override defaults, set `window.__ETIQUETTA_CONFIG__` **before** the tracker script loads:

```html
<script>
  window.__ETIQUETTA_CONFIG__ = {
    debug: true  // Enable console logging
  };
</script>
<script defer data-site="YOUR_SITE_ID" src="https://your-instance.com/s.js"></script>
```

### Config Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `baseUrl` | string | Script origin | Base URL for the Etiquetta server |
| `endpoint` | string | `"/i"` | Ingest endpoint path |
| `siteId` | string | From `data-site` | Site identifier (usually set via HTML attribute) |
| `trackPerformance` | boolean | `true` | Enable Core Web Vitals collection |
| `trackErrors` | boolean | `true` | Enable JavaScript error tracking |
| `respectDNT` | boolean | `true` | Honor browser Do-Not-Track / Global Privacy Control |
| `debug` | boolean | `false` | Log tracker activity to browser console |

---

## Privacy & Consent

### Cookie-Free Tracking

Etiquetta does not use cookies. Visitor identification uses a deterministic fingerprint based on:

- Screen dimensions and color depth
- Timezone offset
- Browser language
- Hardware concurrency
- Platform

This fingerprint is computed client-side and is not personally identifiable. It changes when any of these device characteristics change.

### Do-Not-Track / Global Privacy Control

By default, Etiquetta respects the browser's Do-Not-Track (`DNT`) and Global Privacy Control (`GPC`) signals. When either is enabled, the tracker stops completely — no events are sent.

To disable this behavior:

```javascript
window.__ETIQUETTA_CONFIG__ = { respectDNT: false };
```

### Consent Banner Integration

If you use Etiquetta's consent management feature (Pro), the tracker integrates automatically:

1. The consent banner script (`c.js`) is auto-loaded by the tracker
2. If consent is configured and analytics consent is denied, tracking pauses
3. When the visitor grants analytics consent, tracking starts
4. The tracker listens for `etiquetta:consent` events to react to consent changes

The consent system stores preferences in a first-party cookie (`etiquetta_consent_{siteId}`) with configurable expiry.

### IP Handling

- Client IP addresses are never stored in raw form
- IPs are salted and hashed server-side for session computation
- GeoIP enrichment (country, city, region) happens server-side; the raw IP is discarded after processing

---

## Analytics API

All stats endpoints require authentication (session cookie). Responses are JSON.

### Query Parameters

These parameters work across all `/api/stats/*` endpoints:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `start` | ISO 8601 | — | Start date (e.g. `2024-01-01T00:00:00Z`) |
| `end` | ISO 8601 | — | End date |
| `days` | integer | `7` | Days back from today (alternative to start/end) |
| `domain` | string | — | Filter by domain |
| `country` | string | — | Filter by country code |
| `browser` | string | — | Filter by browser name |
| `device` | string | — | Filter by device type (`mobile`, `desktop`, `tablet`) |
| `page` | string | — | Filter by page path |
| `referrer` | string | — | Filter by referrer (partial match) |
| `bot_filter` | string | Exclude bots | Bot traffic filter (see below) |

### Bot Filter Values

| Value | Description |
|-------|-------------|
| *(default)* | Exclude all bots (is_bot = 0) |
| `all` | Include all traffic |
| `humans` | Only human visitors |
| `bots` | Only bot traffic |
| `good_bots` | Only good bots (search engines, etc.) |
| `bad_bots` | Only malicious/suspicious bots |
| `suspicious` | Only suspicious traffic |

### Endpoints

#### Overview

```
GET /api/stats/overview
```

Returns summary statistics: `total_events`, `unique_visitors`, `sessions`, `pageviews`, `bounce_rate`, `avg_session_seconds`, `live_visitors`.

#### Timeseries

```
GET /api/stats/timeseries
```

Returns pageviews over time: `period`, `pageviews`, `visitors`.

#### Pages

```
GET /api/stats/pages
```

Returns top pages: `path`, `views`, `visitors`.

#### Referrers

```
GET /api/stats/referrers
```

Returns traffic sources: `source`, `referrer_type`, `visits`, `visitors`.

#### Geographic

```
GET /api/stats/geo
```

Returns geographic breakdown: `country`, `visitors`.

```
GET /api/stats/map
```

Returns map data with coordinates: `city`, `country`, `lat`, `lng`, `visitors`, `pageviews`.

#### Devices & Browsers

```
GET /api/stats/devices
```

Returns device breakdown: `device`, `visitors`.

```
GET /api/stats/browsers
```

Returns browser breakdown: `browser`, `visitors`.

#### Campaigns

```
GET /api/stats/campaigns
```

Returns UTM campaign data: `utm_source`, `utm_medium`, `utm_campaign`, `sessions`, `visitors`.

#### Custom Events

```
GET /api/stats/events
```

Returns custom event counts: `event_name`, `count`, `unique_visitors`.

#### Outbound Links

```
GET /api/stats/outbound
```

Returns outbound click data: `url`, `clicks`, `unique_visitors`.

#### Bots

```
GET /api/stats/bots
```

Returns bot traffic breakdown including categories, score distribution, timeseries, and top bots.

#### Core Web Vitals (Pro)

```
GET /api/stats/vitals
```

Returns Web Vitals distributions: LCP, FCP, CLS, INP, TTFB.

#### Errors (Pro)

```
GET /api/stats/errors
```

Returns JavaScript error data: `error_hash`, `error_message`, `count`, `affected_visitors`.

#### Fraud Detection (Enterprise)

```
GET /api/stats/fraud
```

Returns fraud analysis summary.

### Event Ingestion

```
POST /i
```

The ingest endpoint accepts NDJSON (newline-delimited JSON). Each line is one event:

```
{"type":"events","event_id":"abc123","timestamp":1710000000000,"site_id":"site_xxx","domain":"example.com","visitor_hash":"a1b2c3d4","event_type":"pageview","event_name":"pv","url":"https://example.com/","path":"/","page_title":"Home"}
{"type":"events","event_id":"def456","timestamp":1710000001000,"site_id":"site_xxx","domain":"example.com","visitor_hash":"a1b2c3d4","event_type":"custom","event_name":"signup","props":"{\"plan\":\"pro\"}"}
```

This endpoint is used by the tracker script. You generally don't need to call it directly.

---

## Event Types Reference

| `event_type` | `event_name` | Trigger | Key Fields |
|-------------|-------------|---------|------------|
| `pageview` | `pv` | Page load / SPA navigation | `url`, `path`, `referrer_url`, `page_title`, `utm_*` |
| `scroll` | `scroll_25`, `scroll_50`, `scroll_75`, `scroll_100` | Scroll milestone reached | `url`, `path` |
| `click` | `outbound` | External link clicked | `props.target` (destination URL) |
| `engagement` | `page_exit` | Visitor leaves page | `page_duration`, `props.total_time_ms`, `props.visible_time_ms`, `props.scroll_depth` |
| `custom` | *(user-defined)* | `etiquetta.track()` call | `props` (user-defined JSON) |
| `performance` | — | Page unload (Pro) | `lcp`, `fcp`, `cls`, `inp`, `ttfb`, `page_load_time` |
| `error` | — | JS error / rejection (Pro) | `error_message`, `error_stack`, `script_url`, `line_number` |

### Fields Included With Every Event

| Field | Description |
|-------|-------------|
| `event_id` | Unique event identifier |
| `timestamp` | Unix timestamp in milliseconds |
| `site_id` | Domain identifier |
| `domain` | Hostname |
| `visitor_hash` | Deterministic visitor fingerprint |
| `has_scroll` | Whether visitor scrolled (0/1) |
| `has_mouse_move` | Whether mouse movement detected (0/1) |
| `has_click` | Whether visitor clicked (0/1) |
| `has_touch` | Whether touch events detected (0/1) |
| `bot_signals` | Client-side bot detection signals |

### Server-Side Enrichment

These fields are added server-side during ingestion:

| Field | Description |
|-------|-------------|
| `geo_country` | Country (from GeoIP) |
| `geo_city` | City |
| `geo_region` | Region/state |
| `geo_latitude` | Latitude |
| `geo_longitude` | Longitude |
| `browser_name` | Browser (from User-Agent) |
| `os_name` | Operating system |
| `device_type` | `mobile`, `desktop`, or `tablet` |
| `is_bot` | Bot classification (score > 50) |
| `bot_score` | Bot probability score (0–100) |
| `bot_category` | `human`, `good_bot`, `bad_bot`, `suspicious` |
| `session_id` | Server-computed session identifier |
| `ip_hash` | Salted hash of client IP |

---

## Rate Limits & Batching

### Client-Side

| Setting | Value |
|---------|-------|
| Rate limit | 100 events per 60 seconds |
| Batch size | Up to 10 events per request |
| Flush interval | Every 1 second |
| Transport | `navigator.sendBeacon` (preferred) or `fetch` with `keepalive` |
| Payload format | NDJSON (newline-delimited JSON) |

Events are queued in memory and flushed either when the batch reaches 10 events or every second, whichever comes first. The queue is also flushed on `beforeunload` and `pagehide` to capture exit events.

If the rate limit is exceeded, additional events are silently dropped.

### Server-Side

| Setting | Value |
|---------|-------|
| Ingest rate limit | 100 requests per minute per IP |
| Max payload size | 1 MB |
| Write buffer flush | Configurable (default: 50,000 rows or 30 seconds) |

The server buffers incoming events in memory, periodically writes them to Parquet files, then bulk-loads into DuckDB for efficient columnar storage.

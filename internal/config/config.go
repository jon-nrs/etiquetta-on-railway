package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	ListenAddr string `json:"listen_addr"`
	DataDir    string `json:"data_dir"`
	GeoIPPath  string `json:"geoip_path"`

	// Tracker settings
	SessionTimeoutMinutes int  `json:"session_timeout_minutes"`
	TrackPerformance      bool `json:"track_performance"`
	TrackErrors           bool `json:"track_errors"`
	RespectDNT            bool `json:"respect_dnt"`

	// CORS
	AllowedOrigins []string `json:"allowed_origins"`

	// Secret for session HMAC
	SecretKey string `json:"secret_key"`
}

func Load(path string) *Config {
	cfg := &Config{
		ListenAddr:            ":3456",
		DataDir:               "./data",
		GeoIPPath:             "./data/GeoLite2-City.mmdb",
		SessionTimeoutMinutes: 30,
		TrackPerformance:      true,
		TrackErrors:           true,
		RespectDNT:            false,
		AllowedOrigins:        []string{"*"},
		SecretKey:             "change-me-in-production",
	}

	if path == "" {
		path = "config.json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Use defaults if no config file
		return cfg
	}

	json.Unmarshal(data, cfg)
	return cfg
}

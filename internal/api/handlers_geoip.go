package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/caioricciuti/etiquetta/internal/geoip"
	"github.com/caioricciuti/etiquetta/internal/settings"
)

// GeoIPSettings represents the GeoIP configuration
type GeoIPSettings struct {
	AccountID   string `json:"account_id"`
	LicenseKey  string `json:"license_key"`
	GeoIPPath   string `json:"geoip_path"`
	AutoUpdate  bool   `json:"auto_update"`
	LastUpdated string `json:"last_updated"`
}

// GeoIPStatus represents the status of the GeoIP database
type GeoIPStatus struct {
	Exists       bool   `json:"exists"`
	Path         string `json:"path"`
	FileSize     int64  `json:"file_size"`
	LastModified string `json:"last_modified,omitempty"`
	Configured   bool   `json:"configured"`
}

// GetGeoIPSettings returns the current GeoIP settings (with masked credentials)
func (h *Handlers) GetGeoIPSettings(w http.ResponseWriter, r *http.Request) {
	settingsSvc := settings.New(h.db.Conn())

	// Get secret key for decryption
	secretKey, _ := settingsSvc.Get("secret_key")
	if secretKey != "" {
		settingsSvc.SetMasterKey(secretKey)
	}

	accountID, _ := settingsSvc.Get("maxmind_account_id")
	licenseKey, _ := settingsSvc.Get("maxmind_license_key")
	geoipPath := settingsSvc.GetWithDefault("geoip_path", h.cfg.DataDir+"/GeoLite2-City.mmdb")
	autoUpdate := settingsSvc.GetBool("geoip_auto_update", false)
	lastUpdated, _ := settingsSvc.Get("geoip_last_updated")

	// Mask credentials for display
	maskedAccountID := ""
	if accountID != "" {
		maskedAccountID = maskValue(accountID)
	}
	maskedLicenseKey := ""
	if licenseKey != "" {
		maskedLicenseKey = maskValue(licenseKey)
	}

	settings := GeoIPSettings{
		AccountID:   maskedAccountID,
		LicenseKey:  maskedLicenseKey,
		GeoIPPath:   geoipPath,
		AutoUpdate:  autoUpdate,
		LastUpdated: lastUpdated,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// UpdateGeoIPSettings updates the GeoIP settings
func (h *Handlers) UpdateGeoIPSettings(w http.ResponseWriter, r *http.Request) {
	var input struct {
		AccountID  *string `json:"account_id"`
		LicenseKey *string `json:"license_key"`
		AutoUpdate *bool   `json:"auto_update"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	settingsSvc := settings.New(h.db.Conn())

	// Get secret key for encryption
	secretKey, _ := settingsSvc.Get("secret_key")
	if secretKey != "" {
		settingsSvc.SetMasterKey(secretKey)
	}

	// Update only provided fields
	if input.AccountID != nil {
		settingsSvc.Set("maxmind_account_id", *input.AccountID)
	}
	if input.LicenseKey != nil {
		settingsSvc.Set("maxmind_license_key", *input.LicenseKey)
	}
	if input.AutoUpdate != nil {
		if *input.AutoUpdate {
			settingsSvc.Set("geoip_auto_update", "true")
		} else {
			settingsSvc.Set("geoip_auto_update", "false")
		}
	}

	h.logAudit(r, "update", "settings", "geoip", "GeoIP settings updated")
	w.WriteHeader(http.StatusNoContent)
}

// GetGeoIPStatus returns the status of the GeoIP database file
func (h *Handlers) GetGeoIPStatus(w http.ResponseWriter, r *http.Request) {
	settingsSvc := settings.New(h.db.Conn())

	// Get secret key for decryption
	secretKey, _ := settingsSvc.Get("secret_key")
	if secretKey != "" {
		settingsSvc.SetMasterKey(secretKey)
	}

	geoipPath := settingsSvc.GetWithDefault("geoip_path", h.cfg.DataDir+"/GeoLite2-City.mmdb")
	accountID, _ := settingsSvc.Get("maxmind_account_id")
	licenseKey, _ := settingsSvc.Get("maxmind_license_key")

	status := GeoIPStatus{
		Path:       geoipPath,
		Configured: accountID != "" && licenseKey != "",
	}

	if info, err := os.Stat(geoipPath); err == nil {
		status.Exists = true
		status.FileSize = info.Size()
		status.LastModified = info.ModTime().Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// DownloadGeoIPDatabase triggers a download of the GeoIP database
func (h *Handlers) DownloadGeoIPDatabase(w http.ResponseWriter, r *http.Request) {
	settingsSvc := settings.New(h.db.Conn())

	// Get secret key for decryption
	secretKey, _ := settingsSvc.Get("secret_key")
	if secretKey != "" {
		settingsSvc.SetMasterKey(secretKey)
	}

	accountID, _ := settingsSvc.Get("maxmind_account_id")
	licenseKey, _ := settingsSvc.Get("maxmind_license_key")

	if accountID == "" || licenseKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "MaxMind credentials not configured",
		})
		return
	}

	downloader := geoip.NewDownloader(accountID, licenseKey, h.cfg.DataDir)

	if err := downloader.Download(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Update last downloaded timestamp
	settingsSvc.Set("geoip_last_updated", time.Now().Format(time.RFC3339))

	// Reload enricher with new database
	if h.enricher != nil {
		h.enricher.ReloadGeoIP(h.cfg.DataDir + "/GeoLite2-City.mmdb")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "GeoIP database downloaded successfully",
	})
}

// maskValue masks a sensitive value for display
func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}

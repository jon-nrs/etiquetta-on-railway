package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"strconv"

	"github.com/caioricciuti/etiquetta/internal/settings"
)

// EmailSettings represents the email configuration
type EmailSettings struct {
	Provider    string `json:"email_provider"`
	FromAddress string `json:"email_from_address"`
	BaseURL     string `json:"email_base_url"`
	SMTPHost    string `json:"smtp_host"`
	SMTPPort    int    `json:"smtp_port"`
	SMTPUser    string `json:"smtp_username"`
	SMTPPass    string `json:"smtp_password"`
	SMTPUseTLS  bool   `json:"smtp_use_tls"`
	ResendKey   string `json:"resend_api_key"`
}

func newSettingsService(h *Handlers) *settings.Service {
	svc := settings.New(h.db.Conn())
	secretKey, _ := svc.Get("secret_key")
	if secretKey != "" {
		svc.SetMasterKey(secretKey)
	}
	return svc
}

// GetEmailSettings returns the current email settings (with masked credentials)
func (h *Handlers) GetEmailSettings(w http.ResponseWriter, r *http.Request) {
	svc := newSettingsService(h)

	provider := svc.GetWithDefault("email_provider", "disabled")
	fromAddress, _ := svc.Get("email_from_address")
	baseURL, _ := svc.Get("email_base_url")
	smtpHost, _ := svc.Get("smtp_host")
	smtpPort := svc.GetInt("smtp_port", 587)
	smtpUser, _ := svc.Get("smtp_username")
	smtpPass, _ := svc.Get("smtp_password")
	smtpTLS := svc.GetBool("smtp_use_tls", true)
	resendKey, _ := svc.Get("resend_api_key")

	// Mask sensitive values
	maskedPass := ""
	if smtpPass != "" {
		maskedPass = "••••••••"
	}
	maskedResend := ""
	if resendKey != "" {
		maskedResend = "••••••••"
	}

	writeJSON(w, http.StatusOK, EmailSettings{
		Provider:    provider,
		FromAddress: fromAddress,
		BaseURL:     baseURL,
		SMTPHost:    smtpHost,
		SMTPPort:    smtpPort,
		SMTPUser:    smtpUser,
		SMTPPass:    maskedPass,
		SMTPUseTLS:  smtpTLS,
		ResendKey:   maskedResend,
	})
}

// UpdateEmailSettings updates the email settings
func (h *Handlers) UpdateEmailSettings(w http.ResponseWriter, r *http.Request) {
	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	svc := newSettingsService(h)

	keyMap := map[string]string{
		"email_provider":     "email_provider",
		"email_from_address": "email_from_address",
		"email_base_url":     "email_base_url",
		"smtp_host":          "smtp_host",
		"smtp_port":          "smtp_port",
		"smtp_username":      "smtp_username",
		"smtp_password":      "smtp_password",
		"smtp_use_tls":       "smtp_use_tls",
		"resend_api_key":     "resend_api_key",
	}

	for jsonKey, settingKey := range keyMap {
		val, ok := input[jsonKey]
		if !ok {
			continue
		}

		var strVal string
		switch v := val.(type) {
		case string:
			strVal = v
		case float64:
			strVal = strconv.Itoa(int(v))
		case bool:
			if v {
				strVal = "true"
			} else {
				strVal = "false"
			}
		default:
			strVal = fmt.Sprintf("%v", v)
		}

		svc.Set(settingKey, strVal)
	}

	h.logAudit(r, "update", "settings", "email", "Email settings updated")
	w.WriteHeader(http.StatusNoContent)
}

// TestEmailSettings tests the email configuration by attempting a connection
func (h *Handlers) TestEmailSettings(w http.ResponseWriter, r *http.Request) {
	svc := newSettingsService(h)

	provider := svc.GetWithDefault("email_provider", "disabled")

	if provider == "disabled" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": "Email provider is disabled",
		})
		return
	}

	if provider == "smtp" {
		host, _ := svc.Get("smtp_host")
		port := svc.GetInt("smtp_port", 587)

		if host == "" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"message": "SMTP host is not configured",
			})
			return
		}

		addr := net.JoinHostPort(host, strconv.Itoa(port))
		client, err := smtp.Dial(addr)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Failed to connect to SMTP server: %s", err.Error()),
			})
			return
		}
		client.Close()

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Successfully connected to %s", addr),
		})
		return
	}

	if provider == "resend" {
		apiKey, _ := svc.Get("resend_api_key")
		if apiKey == "" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"message": "Resend API key is not configured",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Resend API key is configured",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": false,
		"message": fmt.Sprintf("Unknown email provider: %s", provider),
	})
}

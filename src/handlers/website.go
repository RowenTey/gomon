//go:build js && wasm

package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/RowenTey/gomon/src/models"
	"github.com/RowenTey/gomon/src/storage"
)

const (
	GREEN_HEX = "33D058"
)

type WebsiteHandler struct {
	minFrequency int
	storage      storage.Storage
}

func NewWebsiteHandler(minFrequency int, appStorage storage.Storage) *WebsiteHandler {
	return &WebsiteHandler{
		minFrequency: minFrequency,
		storage:      appStorage,
	}
}

// CreateWebsite handles POST requests to create a new website to monitor
func (h *WebsiteHandler) CreateWebsite(w http.ResponseWriter, r *http.Request) {
	var req models.CreateWebsiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Validate the request
	if req.URL == "" {
		SendJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "URL is required",
		})
		return
	}

	if req.Frequency <= h.minFrequency {
		req.Frequency = h.minFrequency
	}

	if err := validateWebhookRequest(req.WebhookEnabled, req.WebhookURL); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Check if website already exists
	_, err := h.storage.GetWebsite(req.URL)
	if err == nil {
		SendJSONResponse(w, http.StatusConflict, models.APIResponse{
			Success: false,
			Error:   "Website already exists",
		})
		return
	}
	if err != sql.ErrNoRows {
		SendJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to check existing website",
		})
		return
	}

	// Create new website
	now := time.Now().Unix()
	website := models.Website{
		URL:                    req.URL,
		Frequency:              req.Frequency,
		CreatedAt:              now,
		Status:                 models.StatusUnknown,
		LastCheckedAt:          0,
		ResponseTime:           -1,
		StatusCode:             -1,
		Error:                  "",
		WebhookEnabled:         req.WebhookEnabled,
		WebhookURL:             req.WebhookURL,
		WebhookPayloadTemplate: req.WebhookPayloadTemplate,
	}

	if err := h.storage.CreateWebsite(website); err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to save website",
		})
		return
	}

	SendJSONResponse(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Website added to monitoring",
		Data:    website,
	})
}

// GetWebsite handles GET requests to retrieve a website configuration
func (h *WebsiteHandler) GetWebsite(w http.ResponseWriter, r *http.Request, url string) {
	website, err := h.storage.GetWebsite(url)
	if err != nil {
		SendJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Website not found",
		})
		return
	}

	// Return website without status
	SendJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    website,
	})
}

func (h *WebsiteHandler) UpdateWebsite(w http.ResponseWriter, r *http.Request, url string) {
	website, err := h.storage.GetWebsite(url)
	if err != nil {
		SendJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Website not found",
		})
		return
	}

	var req models.UpdateWebsiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.Frequency != nil {
		frequency := *req.Frequency
		if frequency <= h.minFrequency {
			frequency = h.minFrequency
		}
		website.Frequency = frequency
	}

	if req.WebhookEnabled != nil {
		website.WebhookEnabled = *req.WebhookEnabled
	}
	if req.WebhookURL != nil {
		website.WebhookURL = *req.WebhookURL
	}
	if req.WebhookPayloadTemplate != nil {
		website.WebhookPayloadTemplate = *req.WebhookPayloadTemplate
	}
	if err := validateWebhookRequest(website.WebhookEnabled, website.WebhookURL); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.storage.UpdateWebsite(*website); err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to update website",
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Website updated",
		Data:    website,
	})
}
func (h *WebsiteHandler) DeleteWebsite(w http.ResponseWriter, r *http.Request, url string) {
	if err := h.storage.DeleteWebsite(url); err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to delete website",
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Website deleted",
	})
}

// ListWebsites handles GET requests to retrieve all websites
func (h *WebsiteHandler) ListWebsites(w http.ResponseWriter, r *http.Request) {
	websites, err := h.storage.ListWebsites(1000)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to list websites",
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    websites,
	})
}

func (h *WebsiteHandler) GetShieldsIoBadge(w http.ResponseWriter, r *http.Request) {
	// Extract URL from query parameters
	url := r.URL.Query().Get("websiteUrl")
	if url == "" {
		SendJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "URL parameter is required",
		})
		return
	}

	// Prepare badge data
	badge := models.ShieldsIoBadge{
		SchemaVersion: 1,
		Label:         "STATUS",
	}

	website, err := h.storage.GetWebsite(url)
	if err != nil {
		badge.Message = "UNKNOWN"
		badge.Color = "red"
		SendJSONResponse(w, http.StatusNotFound, badge)
		return
	}

	badge.Message = strings.ToUpper(string(website.Status))

	// Set color based on status
	switch website.Status {
	case models.StatusUp:
		badge.Color = GREEN_HEX
	case models.StatusUnknown:
		badge.Color = "red"
	case models.StatusDown:
		badge.Color = "red"
	case models.StatusDegraded:
		badge.Color = "orange"
	}

	// Directly send the badge as JSON response so Shields.io can use it
	SendJSONResponse(w, http.StatusOK, badge)
}

func validateWebhookRequest(enabled bool, webhookURL string) error {
	if !enabled {
		return nil
	}
	if webhookURL == "" {
		return errors.New("webhookUrl is required when webhookEnabled is true")
	}
	parsed, err := url.ParseRequestURI(webhookURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("webhookUrl must be a valid http/https URL")
	}
	return nil
}

// Helper function to send JSON responses
func SendJSONResponse(w http.ResponseWriter, statusCode int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

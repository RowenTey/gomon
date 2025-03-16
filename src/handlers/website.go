//go:build js && wasm

package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/RowenTey/gomon/src/models"
	"github.com/RowenTey/gomon/src/storage"
)

type WebsiteHandler struct {
	storage storage.KVStorage
}

func NewWebsiteHandler(kvStorage storage.KVStorage) *WebsiteHandler {
	return &WebsiteHandler{
		storage: kvStorage,
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

	if req.Frequency <= 60 {
		req.Frequency = 60 // Minimum 60 seconds
	}

	// Check if website already exists
	var existingWebsite models.Website
	key := models.GetKey(req.URL)
	err := h.storage.GetJSON(key, &existingWebsite)
	if err == nil {
		SendJSONResponse(w, http.StatusConflict, models.APIResponse{
			Success: false,
			Error:   "Website already exists",
		})
		return
	}

	// Create new website
	now := time.Now().Unix()
	website := models.Website{
		URL:           req.URL,
		Frequency:     req.Frequency,
		CreatedAt:     now,
		Status:        models.StatusUnknown,
		LastCheckedAt: 0,  // Never checked
		ResponseTime:  -1, // No response time yet
		StatusCode:    -1, // No status code yet
		Error:         "", // No error yet
	}

	// Save to KV
	if err := h.storage.PutJSON(key, website); err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to save website configuration",
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
	// Get website config from KV
	var website models.Website
	key := models.GetKey(url)
	if err := h.storage.GetJSON(key, &website); err != nil {
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
	// Get website config from KV
	var website models.Website
	key := models.GetKey(url)
	if err := h.storage.GetJSON(key, &website); err != nil {
		SendJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Website not found",
		})
		return
	}

	var req models.CreateWebsiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Update website
	website.Frequency = req.Frequency
	if req.Frequency <= 60 {
		req.Frequency = 60 // Minimum 60 seconds
	}

	// Save to KV
	if err := h.storage.PutJSON(key, website); err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to update website configuration",
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
	// Get website config from KV
	key := models.GetKey(url)
	if err := h.storage.Delete(key); err != nil {
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
	// Get all website keys from KV
	keys, err := h.storage.List("websites_", 1000) // Limit to 1000 websites
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to list websites",
		})
		return
	}

	websites := []models.Website{}

	// Get each website
	for _, key := range keys {
		var website models.Website
		if err := h.storage.GetJSON(key, &website); err != nil {
			// Skip if we can't retrieve this website
			continue
		}
		websites = append(websites, website)
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

	// Get website config from KV
	var website models.Website
	key := models.GetKey(url)
	if err := h.storage.GetJSON(key, &website); err != nil {
		badge.Message = "UNKNOWN"
		badge.Color = "RED"
		SendJSONResponse(w, http.StatusNotFound, badge)
		return
	}

	badge.Message = strings.ToUpper(string(website.Status))

	// Set color based on status
	switch website.Status {
	case models.StatusUp:
		badge.Color = "GREEN"
	case models.StatusUnknown:
		badge.Color = "RED"
	case models.StatusDown:
		badge.Color = "RED"
	case models.StatusDegraded:
		badge.Color = "ORANGE"
	}

	// Directly send the badge as JSON response so Shields.io can use it
	SendJSONResponse(w, http.StatusOK, badge)
}

// Helper function to send JSON responses
func SendJSONResponse(w http.ResponseWriter, statusCode int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

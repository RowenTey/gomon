package models

import "fmt"

// Website represents a monitored website configuration
type Website struct {
	URL           string     `json:"url"`
	Frequency     int        `json:"frequency"` // in seconds
	LastCheckedAt int64      `json:"lastCheckedAt,omitempty"`
	CreatedAt     int64      `json:"createdAt"`
	Status        StatusType `json:"status"`
	ResponseTime  int        `json:"responseTime,omitempty"` // in milliseconds
	StatusCode    int        `json:"statusCode"`
	Error         string     `json:"error,omitempty"`
}

// StatusType defines the monitoring status type
type StatusType string

const (
	StatusUp       StatusType = "up"
	StatusDown     StatusType = "down"
	StatusDegraded StatusType = "degraded"
	StatusUnknown  StatusType = "unknown"
)

// CreateWebsiteRequest represents the payload for creating a new website to monitor
type CreateWebsiteRequest struct {
	URL       string `json:"url"`
	Frequency int    `json:"frequency"` // in seconds
}

// APIResponse represents a standard API response format
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ShieldsIoBadge represents a JSON response for Shields.io badges
type ShieldsIoBadge struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

// Helper function to format KV storage key
func GetKey(url string) string {
	return fmt.Sprintf("websites_%s", url)
}

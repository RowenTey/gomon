package models

import (
	"fmt"
	"hash/fnv"
	"time"
)

// Website represents a monitored website configuration
type Website struct {
	URL                           string            `json:"url"`
	Frequency                     int               `json:"frequency"` // in seconds
	LastCheckedAt                 int64             `json:"lastCheckedAt,omitempty"`
	CreatedAt                     int64             `json:"createdAt"`
	Status                        StatusType        `json:"status"`
	ResponseTime                  int               `json:"responseTime,omitempty"` // in milliseconds
	StatusCode                    int               `json:"statusCode"`
	Error                         string            `json:"error,omitempty"`
	CustomHeaders                 map[string]string `json:"customHeaders,omitempty"`
	LastUnhealthyNotificationAt   int64             `json:"lastUnhealthyNotificationAt,omitempty"`
	LastUnhealthyNotificationType StatusType        `json:"lastUnhealthyNotificationType,omitempty"`

	WebhookEnabled         bool   `json:"webhookEnabled"`
	WebhookURL             string `json:"webhookUrl,omitempty"`
	WebhookPayloadTemplate string `json:"webhookPayloadTemplate,omitempty"`
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
	URL                    string            `json:"url"`
	Frequency              int               `json:"frequency"` // in seconds
	CustomHeaders          map[string]string `json:"customHeaders"`
	WebhookEnabled         bool              `json:"webhookEnabled"`
	WebhookURL             string            `json:"webhookUrl"`
	WebhookPayloadTemplate string            `json:"webhookPayloadTemplate"`
}

// UpdateWebsiteRequest represents editable website fields.
type UpdateWebsiteRequest struct {
	Frequency              *int               `json:"frequency,omitempty"`
	CustomHeaders          *map[string]string `json:"customHeaders,omitempty"`
	WebhookEnabled         *bool              `json:"webhookEnabled,omitempty"`
	WebhookURL             *string            `json:"webhookUrl,omitempty"`
	WebhookPayloadTemplate *string            `json:"webhookPayloadTemplate,omitempty"`
}

// WebhookRuntimeConfig is global webhook behavior shared by all websites.
// It is intended to be loaded from environment variables.
type WebhookRuntimeConfig struct {
	NotifyOnRecovery           bool    `json:"notifyOnRecovery"`
	MaxAttempts                int     `json:"maxAttempts"`
	InitialDelaySec            int     `json:"initialDelaySec"`
	MaxDelaySec                int     `json:"maxDelaySec"`
	BackoffFactor              float64 `json:"backoffFactor"`
	RepeatUnhealthyCooldownSec int     `json:"repeatUnhealthyCooldownSec"`
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

// WebhookEvent is the callback payload sent to external webhook endpoints.
type WebhookEvent struct {
	EventID        string     `json:"eventId"`
	WebsiteURL     string     `json:"websiteUrl"`
	Timestamp      int64      `json:"timestamp"`
	PreviousStatus StatusType `json:"previousStatus"`
	CurrentStatus  StatusType `json:"currentStatus"`
	ResponseTime   int        `json:"responseTime"`
	StatusCode     int        `json:"statusCode"`
	Error          string     `json:"error,omitempty"`
}

// WebhookDelivery tracks webhook queue/retry state.
type WebhookDelivery struct {
	EventID         string  `json:"eventId"`
	WebsiteURL      string  `json:"websiteUrl"`
	WebhookURL      string  `json:"webhookUrl"`
	Payload         string  `json:"payload"`
	AttemptCount    int     `json:"attemptCount"`
	MaxAttempts     int     `json:"maxAttempts"`
	InitialDelaySec int     `json:"initialDelaySec"`
	MaxDelaySec     int     `json:"maxDelaySec"`
	BackoffFactor   float64 `json:"backoffFactor"`
	NextAttemptAt   int64   `json:"nextAttemptAt"`
	DeliveredAt     int64   `json:"deliveredAt,omitempty"`
	LastError       string  `json:"lastError,omitempty"`
	CreatedAt       int64   `json:"createdAt"`
	UpdatedAt       int64   `json:"updatedAt"`
}

// ApplyDefaults ensures webhook config values are always valid.
func (c *WebhookRuntimeConfig) ApplyDefaults() {
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 3
	}
	if c.InitialDelaySec <= 0 {
		c.InitialDelaySec = 30
	}
	if c.MaxDelaySec <= 0 {
		c.MaxDelaySec = 300
	}
	if c.BackoffFactor < 1.0 {
		c.BackoffFactor = 2.0
	}
	if c.MaxDelaySec < c.InitialDelaySec {
		c.MaxDelaySec = c.InitialDelaySec
	}
	if c.RepeatUnhealthyCooldownSec <= 0 {
		c.RepeatUnhealthyCooldownSec = 600
	}
}

// NewWebhookDelivery creates a queued delivery from an event and website config.
func NewWebhookDelivery(eventID string, website Website, config WebhookRuntimeConfig, payload string, now int64) WebhookDelivery {
	config.ApplyDefaults()
	return WebhookDelivery{
		EventID:         eventID,
		WebsiteURL:      website.URL,
		WebhookURL:      website.WebhookURL,
		Payload:         payload,
		AttemptCount:    0,
		MaxAttempts:     config.MaxAttempts,
		InitialDelaySec: config.InitialDelaySec,
		MaxDelaySec:     config.MaxDelaySec,
		BackoffFactor:   config.BackoffFactor,
		NextAttemptAt:   now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// NewEventID creates a simple unique event id for transition notifications.
func NewEventID(url string, now time.Time) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(url))
	return fmt.Sprintf("%s-%08x", now.UTC().Format("20060102150405.000000000"), h.Sum32())
}

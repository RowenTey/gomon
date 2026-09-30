//go:build js && wasm

package monitoring

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/RowenTey/gomon/src/httpclient"
	"github.com/RowenTey/gomon/src/models"
	"github.com/RowenTey/gomon/src/storage"
)

// userAgent identifies gomon's checks to monitored origins.
const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/111.0"

// webhookTimeout bounds a single webhook delivery attempt.
const webhookTimeout = 15 * time.Second

// Monitor manages website monitoring operations
type Monitor struct {
	storage       storage.Storage
	runtimeConfig models.WebhookRuntimeConfig
	isRunning     bool
	mu            sync.Mutex
	timeoutSec    int
}

// NewMonitor creates a new monitor instance
func NewMonitor(appStorage storage.Storage, config models.WebhookRuntimeConfig, timeoutSec int) *Monitor {
	config.ApplyDefaults()
	if timeoutSec <= 0 {
		timeoutSec = 3
	}
	return &Monitor{
		storage:       appStorage,
		runtimeConfig: config,
		isRunning:     false,
		timeoutSec:    timeoutSec,
	}
}

// StartMonitoring begins the monitoring routine
func (m *Monitor) StartMonitoring() {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.isRunning = false
		m.mu.Unlock()
	}()

	log.Println("Starting monitoring routine...")

	// Run immediately first, then on interval
	log.Println("Checking websites...")
	m.checkWebsites(m.runtimeConfig)
	m.processWebhookDeliveries()
}

// checkWebsites checks which websites need to be monitored
func (m *Monitor) checkWebsites(config models.WebhookRuntimeConfig) {
	now := time.Now().Unix()
	websites, err := m.storage.ListWebsitesDueForCheck(now, 1000)
	if err != nil {
		log.Printf("Error listing due websites: %v\n", err)
		return
	}

	if len(websites) == 0 {
		log.Println("No websites to monitor!")
		return
	}

	// Create a waitgroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	for _, website := range websites {
		log.Printf("Checking website %s...\n", website.URL)
		wg.Add(1)
		go func(site models.Website) {
			defer wg.Done()
			m.checkWebsite(&site, config)
		}(website)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	log.Println("Websites check complete!")
}

// checkWebsite checks a single website and updates its status
func (m *Monitor) checkWebsite(website *models.Website, config models.WebhookRuntimeConfig) {
	startTime := time.Now()
	previousStatus := website.Status

	// Set User-Agent to identify
	headers := map[string]string{"User-Agent": userAgent}
	for key, value := range website.CustomHeaders {
		headers[key] = value
	}

	// The body is never read, so it is cancelled rather than buffered.
	resp, err := httpclient.Do(httpclient.Request{
		Method:  http.MethodGet,
		URL:     website.URL,
		Headers: headers,
		Timeout: time.Duration(m.timeoutSec) * time.Second,
	}, false)
	responseTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		log.Printf("Error checking website %s: %v\n", website.URL, err)
		m.updateStatus(website, previousStatus, 0, responseTime, err.Error(), config)
		return
	}

	// Update status based on response
	log.Printf("Website %s check complete, status - %v", website.URL, resp.StatusCode)
	m.updateStatus(website, previousStatus, resp.StatusCode, responseTime, "", config)
}

// updateStatus updates the status of a monitored website
func (m *Monitor) updateStatus(website *models.Website, previousStatus models.StatusType, statusCode, responseTime int, errorMsg string, config models.WebhookRuntimeConfig) {
	now := time.Now().Unix()
	website.LastCheckedAt = now
	website.ResponseTime = responseTime
	website.StatusCode = statusCode
	website.Error = errorMsg

	// Determine status type
	if errorMsg != "" {
		website.Status = models.StatusDown
	} else if statusCode >= 500 {
		website.Status = models.StatusDown
	} else if statusCode >= 400 {
		website.Status = models.StatusDegraded
	} else if statusCode >= 200 && statusCode < 300 {
		website.Status = models.StatusUp
	} else {
		website.Status = models.StatusUnknown
	}

	if website.Status == models.StatusUp && previousStatus != models.StatusUp {
		website.LastUnhealthyNotificationAt = 0
		website.LastUnhealthyNotificationType = ""
	}

	shouldNotify, remainingCooldownSec := m.shouldNotify(previousStatus, website.Status, website, config, now)

	// Render the payload before persisting so the notification cooldown fields
	// can be written together with the status in a single D1 round-trip.
	var delivery *models.WebhookDelivery
	if shouldNotify {
		event := models.WebhookEvent{
			EventID:        models.NewEventID(website.URL, time.Now()),
			WebsiteURL:     website.URL,
			Timestamp:      now,
			PreviousStatus: previousStatus,
			CurrentStatus:  website.Status,
			ResponseTime:   website.ResponseTime,
			StatusCode:     website.StatusCode,
			Error:          website.Error,
		}
		payload, err := renderPayload(website.WebhookPayloadTemplate, event)
		if err != nil {
			log.Printf("Error rendering webhook payload for %s: %v\n", website.URL, err)
		} else {
			log.Printf("Rendered webhook payload for %s: %s\n", website.URL, payload)
			queued := models.NewWebhookDelivery(event.EventID, *website, config, payload, now)
			delivery = &queued
			if isUnhealthyStatus(website.Status) {
				website.LastUnhealthyNotificationAt = now
				website.LastUnhealthyNotificationType = website.Status
			}
		}
	}

	if err := m.storage.UpdateWebsite(*website); err != nil {
		log.Printf("Error updating status for %s: %v\n", website.URL, err)
		return
	}

	if !shouldNotify {
		if remainingCooldownSec > 0 {
			log.Printf("Skipping webhook notification for %s due to cooldown (%ds remaining)\n", website.URL, remainingCooldownSec)
			return
		}
		log.Printf("No notification needed for %s status change from %s to %s\n", website.URL, previousStatus, website.Status)
		return
	}

	if delivery == nil {
		return
	}

	if err := m.storage.EnqueueWebhookDelivery(*delivery); err != nil {
		log.Printf("Error enqueueing webhook delivery for %s: %v\n", website.URL, err)
		return
	}

	log.Printf("Enqueued webhook delivery for %s status change from %s to %s with event ID %s\n", website.URL, previousStatus, website.Status, delivery.EventID)
}

func (m *Monitor) shouldNotify(previousStatus, currentStatus models.StatusType, website *models.Website, config models.WebhookRuntimeConfig, now int64) (bool, int64) {
	if !website.WebhookEnabled || website.WebhookURL == "" {
		return false, 0
	}

	if isUnhealthyStatus(currentStatus) {
		cooldownSec := int64(config.RepeatUnhealthyCooldownSec)
		if website.LastUnhealthyNotificationAt > 0 {
			elapsed := now - website.LastUnhealthyNotificationAt
			if elapsed < cooldownSec {
				return false, cooldownSec - elapsed
			}
		}

		return true, 0
	}

	if config.NotifyOnRecovery && previousStatus != models.StatusUp && currentStatus == models.StatusUp {
		return true, 0
	}

	return false, 0
}

func isUnhealthyStatus(status models.StatusType) bool {
	return status == models.StatusDegraded || status == models.StatusDown
}

func (m *Monitor) processWebhookDeliveries() {
	now := time.Now().Unix()
	deliveries, err := m.storage.ListDueWebhookDeliveries(now, 200)
	if err != nil {
		log.Printf("Error listing due webhook deliveries: %v\n", err)
		return
	}

	if len(deliveries) == 0 {
		log.Printf("No deliveries to be processed...")
		return
	}

	for _, delivery := range deliveries {
		if err := validateDeliveryTarget(delivery.WebhookURL); err != nil {
			log.Printf("Skipping invalid webhook URL %s: %v\n", delivery.WebhookURL, err)
			m.scheduleFailureRetry(delivery, err.Error())
			continue
		}

		resp, err := httpclient.Do(httpclient.Request{
			Method:  http.MethodPost,
			URL:     delivery.WebhookURL,
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    delivery.Payload,
			Timeout: webhookTimeout,
		}, true)
		if err != nil {
			log.Printf("Error sending webhook request for %s: %v\n", delivery.WebhookURL, err)
			m.scheduleFailureRetry(delivery, err.Error())
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if err := m.storage.MarkWebhookDeliverySuccess(delivery.EventID, time.Now().Unix()); err != nil {
				log.Printf("Error marking webhook delivery success for %s: %v\n", delivery.EventID, err)
			}
			continue
		}

		log.Printf("Received non-2xx response for webhook %s: %d, body: %s\n", delivery.WebhookURL, resp.StatusCode, resp.Body)
		m.scheduleFailureRetry(delivery, fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
	}
}

func (m *Monitor) scheduleFailureRetry(delivery models.WebhookDelivery, errMsg string) {
	nextAttemptCount := delivery.AttemptCount + 1
	exhausted := nextAttemptCount >= delivery.MaxAttempts
	nextAttemptAt := int64(0)
	if !exhausted {
		delay := calculateBackoffDelay(delivery, nextAttemptCount)
		nextAttemptAt = time.Now().Unix() + int64(delay)
	}
	if err := m.storage.MarkWebhookDeliveryFailure(delivery.EventID, nextAttemptAt, nextAttemptCount, exhausted, errMsg); err != nil {
		log.Printf("Error marking webhook delivery failure for %s: %v\n", delivery.EventID, err)
	}
}

func calculateBackoffDelay(delivery models.WebhookDelivery, attempt int) int {
	initial := delivery.InitialDelaySec
	maxDelay := delivery.MaxDelaySec
	factor := delivery.BackoffFactor

	if initial <= 0 {
		initial = 30
	}
	if maxDelay <= 0 {
		maxDelay = 300
	}
	if maxDelay < initial {
		maxDelay = initial
	}
	if factor < 1 {
		factor = 2
	}

	if attempt <= 0 {
		return initial
	}
	delay := float64(initial)
	for i := 1; i < attempt; i++ {
		delay *= factor
		if int(delay) >= maxDelay {
			return maxDelay
		}
	}
	if int(delay) < initial {
		return initial
	}
	if int(delay) > maxDelay {
		return maxDelay
	}
	return int(delay)
}

func validateDeliveryTarget(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported webhook scheme: %s", parsed.Scheme)
	}
	return nil
}

func renderPayload(payloadTemplate string, event models.WebhookEvent) (string, error) {
	if strings.TrimSpace(payloadTemplate) == "" {
		payloadBytes, err := json.Marshal(event)
		if err != nil {
			return "", err
		}
		return string(payloadBytes), nil
	}

	replacer := strings.NewReplacer(
		"{{eventId}}", escapeJSONStringValue(event.EventID),
		"{{websiteUrl}}", escapeJSONStringValue(event.WebsiteURL),
		"{{timestamp}}", time.Unix(event.Timestamp, 0).UTC().Format(time.RFC3339),
		"{{previousStatus}}", escapeJSONStringValue(string(event.PreviousStatus)),
		"{{currentStatus}}", escapeJSONStringValue(strings.ToUpper(string(event.CurrentStatus))),
		"{{responseTime}}", strconv.Itoa(event.ResponseTime),
		"{{statusCode}}", strconv.Itoa(event.StatusCode),
		"{{error}}", escapeJSONStringValue(event.Error),
	)

	rendered := replacer.Replace(payloadTemplate)
	if !json.Valid([]byte(rendered)) {
		return "", fmt.Errorf("rendered webhook payload is not valid JSON")
	}

	return rendered, nil
}

func escapeJSONStringValue(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return value
	}
	if len(encoded) < 2 {
		return value
	}
	return string(encoded[1 : len(encoded)-1])
}

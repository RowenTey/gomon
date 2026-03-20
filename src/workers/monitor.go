//go:build js && wasm

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/RowenTey/gomon/src/models"
	"github.com/RowenTey/gomon/src/storage"
	"github.com/syumai/workers/cloudflare/fetch"
)

// Monitor manages website monitoring operations
type Monitor struct {
	storage       storage.Storage
	runtimeConfig models.WebhookRuntimeConfig
	httpClient    *http.Client
	isRunning     bool
	mu            sync.Mutex
}

// NewMonitor creates a new monitor instance
func NewMonitor(appStorage storage.Storage, config models.WebhookRuntimeConfig) *Monitor {
	config.ApplyDefaults()
	return &Monitor{
		storage:       appStorage,
		runtimeConfig: config,
		isRunning:     false,
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

	// Create fetch client
	cli := fetch.NewClient()

	// Make HTTP request to check status
	req, err := fetch.NewRequest(context.Background(), http.MethodGet, website.URL, nil)
	if err != nil {
		m.updateStatus(website, previousStatus, 0, -1, err.Error(), config)
		return
	}

	// Set User-Agent to identify
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/111.0")
	for key, value := range website.CustomHeaders {
		req.Header.Set(key, value)
	}

	resp, err := cli.Do(req, nil)
	responseTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		m.updateStatus(website, previousStatus, 0, responseTime, err.Error(), config)
		return
	}
	defer resp.Body.Close()

	// Update status based on response
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

	if err := m.storage.UpdateWebsite(*website); err != nil {
		log.Printf("Error updating status for %s: %v\n", website.URL, err)
		return
	}

	if !m.shouldNotify(previousStatus, website.Status, website, config) {
		return
	}

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
		return
	}
	delivery := models.NewWebhookDelivery(event.EventID, *website, config, payload, now)
	if err := m.storage.EnqueueWebhookDelivery(delivery); err != nil {
		log.Printf("Error enqueueing webhook delivery for %s: %v\n", website.URL, err)
	}
}

func (m *Monitor) shouldNotify(previousStatus, currentStatus models.StatusType, website *models.Website, config models.WebhookRuntimeConfig) bool {
	if !website.WebhookEnabled || website.WebhookURL == "" {
		return false
	}
	if previousStatus == currentStatus {
		return false
	}
	if currentStatus == models.StatusDegraded || currentStatus == models.StatusDown {
		return true
	}
	if config.NotifyOnRecovery && previousStatus != models.StatusUp && currentStatus == models.StatusUp {
		return true
	}
	return false
}

func (m *Monitor) processWebhookDeliveries() {
	now := time.Now().Unix()
	deliveries, err := m.storage.ListDueWebhookDeliveries(now, 200)
	if err != nil {
		log.Printf("Error listing due webhook deliveries: %v\n", err)
		return
	}

	if len(deliveries) == 0 {
		return
	}

	cli := fetch.NewClient()
	for _, delivery := range deliveries {
		if err := validateDeliveryTarget(delivery.WebhookURL); err != nil {
			log.Printf("Skipping invalid webhook URL %s: %v\n", delivery.WebhookURL, err)
			m.scheduleFailureRetry(delivery, err.Error())
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		req, err := fetch.NewRequest(ctx, http.MethodPost, delivery.WebhookURL, strings.NewReader(delivery.Payload))
		if err != nil {
			log.Printf("Error creating webhook request for %s: %v\n", delivery.WebhookURL, err)
			cancel()
			m.scheduleFailureRetry(delivery, err.Error())
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := cli.Do(req, nil)
		cancel()
		if err != nil {
			log.Printf("Error sending webhook request for %s: %v\n", delivery.WebhookURL, err)
			m.scheduleFailureRetry(delivery, err.Error())
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if err := m.storage.MarkWebhookDeliverySuccess(delivery.EventID, time.Now().Unix()); err != nil {
				log.Printf("Error marking webhook delivery success for %s: %v\n", delivery.EventID, err)
			}
			continue
		}

		log.Printf("Received non-2xx response for webhook %s: %d\n", delivery.WebhookURL, resp.StatusCode)
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
		"{{eventId}}", event.EventID,
		"{{websiteUrl}}", event.WebsiteURL,
		"{{timestamp}}", strconv.FormatInt(event.Timestamp, 10),
		"{{previousStatus}}", string(event.PreviousStatus),
		"{{currentStatus}}", string(event.CurrentStatus),
		"{{responseTime}}", strconv.Itoa(event.ResponseTime),
		"{{statusCode}}", strconv.Itoa(event.StatusCode),
		"{{error}}", event.Error,
	)

	return replacer.Replace(payloadTemplate), nil
}

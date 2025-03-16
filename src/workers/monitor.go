//go:build js && wasm

package monitoring

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/RowenTey/gomon/src/models"
	"github.com/RowenTey/gomon/src/storage"
	"github.com/syumai/workers/cloudflare/fetch"
)

// Monitor manages website monitoring operations
type Monitor struct {
	storage    storage.KVStorage
	httpClient *http.Client
	isRunning  bool
}

// NewMonitor creates a new monitor instance
func NewMonitor(kvStorage storage.KVStorage) *Monitor {
	return &Monitor{
		storage:   kvStorage,
		isRunning: false,
	}
}

// StartMonitoring begins the monitoring routine
func (m *Monitor) StartMonitoring() {
	if m.isRunning {
		return
	}

	log.Println("Starting monitoring routine...")
	m.isRunning = true

	// Run immediately first, then on interval
	log.Println("Checking websites...")
	m.checkWebsites()
}

// checkWebsites checks which websites need to be monitored
func (m *Monitor) checkWebsites() {
	// Get all website keys
	keys, err := m.storage.List("websites_", 1000)
	if err != nil {
		log.Printf("Error listing websites: %v\n", err)
		return
	}

	if len(keys) == 0 {
		log.Println("No websites to monitor!")
		return
	}

	// Create a waitgroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	now := time.Now().Unix()
	for _, key := range keys {
		var website models.Website
		if err := m.storage.GetJSON(key, &website); err != nil {
			log.Printf("Error getting website %s: %v\n", key, err)
			continue
		}

		// Check if it's time to monitor this website
		if website.LastCheckedAt == 0 || now-website.LastCheckedAt >= int64(website.Frequency) {
			log.Printf("Checking website %s...\n", website.URL)
			// Add to waitgroup before starting the goroutine
			wg.Add(1)
			go func(site models.Website) {
				defer wg.Done()
				m.checkWebsite(&site)
			}(website)
		}
	}

	// Wait for all goroutines to finish
	wg.Wait()

	log.Println("Websites check complete!")
}

// checkWebsite checks a single website and updates its status
func (m *Monitor) checkWebsite(website *models.Website) {
	startTime := time.Now()

	// Create fetch client
	cli := fetch.NewClient()

	// Make HTTP request to check status
	req, err := fetch.NewRequest(context.Background(), http.MethodGet, website.URL, nil)
	if err != nil {
		m.updateStatus(website, 0, -1, err.Error())
		return
	}

	// Set User-Agent to identify our monitoring tool
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/111.0")

	resp, err := cli.Do(req, nil)
	responseTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		m.updateStatus(website, 0, responseTime, err.Error())
		return
	}
	defer resp.Body.Close()

	// Update status based on response
	m.updateStatus(website, resp.StatusCode, responseTime, "")
}

// updateStatus updates the status of a monitored website
func (m *Monitor) updateStatus(website *models.Website, statusCode, responseTime int, errorMsg string) {
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

	// Save status to KV
	key := models.GetKey(website.URL)
	if err := m.storage.PutJSON(key, website); err != nil {
		log.Printf("Error updating status for %s: %v\n", website.URL, err)
	}
}

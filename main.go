//go:build js && wasm

package main

import (
	"context"
	"log"
	"net/http"

	"github.com/RowenTey/gomon/src/handlers"
	"github.com/RowenTey/gomon/src/models"
	"github.com/RowenTey/gomon/src/storage"
	monitoring "github.com/RowenTey/gomon/src/workers"
	"github.com/syumai/workers"
	"github.com/syumai/workers/cloudflare"
	"github.com/syumai/workers/cloudflare/cron"
)

func main() {
	log.Println("Starting gomon...")

	// Initialize KV storage for websites and monitoring results
	kvStorage, err := storage.NewKVStorage(cloudflare.Getenv("KV_NAMESPACE"))
	if err != nil {
		panic(err)
	}

	// Initialize website handler
	websiteHandler := handlers.NewWebsiteHandler(kvStorage)

	// Register API routes
	// Create a single handler function for all routes
	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set common headers
		w.Header().Set("Content-Type", "application/json")

		// Simple router based on path
		path := r.URL.Path

		switch {
		// Handle API endpoints
		case path == "/api/websites":
			if r.Method == http.MethodPost {
				websiteHandler.CreateWebsite(w, r)
			} else if r.Method == http.MethodGet {
				// Extract URL from query parameters
				url := r.URL.Query().Get("websiteUrl")
				if url == "" {
					websiteHandler.ListWebsites(w, r)
					break
				}
				websiteHandler.GetWebsite(w, r, url)
			} else if r.Method == http.MethodPut {
				url := r.URL.Query().Get("websiteUrl")
				if url == "" {
					handlers.SendJSONResponse(w, http.StatusBadRequest, models.APIResponse{
						Success: false,
						Error:   "URL parameter is required",
					})
					break
				}
				websiteHandler.UpdateWebsite(w, r, url)
			} else if r.Method == http.MethodDelete {
				url := r.URL.Query().Get("websiteUrl")
				if url == "" {
					handlers.SendJSONResponse(w, http.StatusBadRequest, models.APIResponse{
						Success: false,
						Error:   "URL parameter is required",
					})
					break
				}
				websiteHandler.DeleteWebsite(w, r, url)
			} else {
				handlers.SendJSONResponse(w, http.StatusMethodNotAllowed, models.APIResponse{
					Success: false,
					Error:   "Method not allowed",
				})
			}

		// Shields.io badge endpoint
		case path == "/api/websites/badge":
			if r.Method == http.MethodGet {
				websiteHandler.GetShieldsIoBadge(w, r)
			} else {
				handlers.SendJSONResponse(w, http.StatusMethodNotAllowed, models.APIResponse{
					Success: false,
					Error:   "Method not allowed",
				})
			}

		// Health check endpoint
		case path == "/health":
			handlers.SendJSONResponse(w, http.StatusOK, models.APIResponse{
				Success: true,
				Error:   "",
				Data:    map[string]string{"status": "ok"},
			})

		// Root endpoint
		case path == "/":
			handlers.SendJSONResponse(w, http.StatusOK, models.APIResponse{
				Success: true,
				Error:   "",
				Data:    map[string]string{"message": "Hello from gomon :)"},
			})

		// Handle 404
		default:
			// 404
			handlers.SendJSONResponse(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Error:   "Endpoint not found",
			})
		}
	})

	// Set up worker
	workers.ServeNonBlock(mainHandler)

	// Set up cron job to run monitoring routine
	task := func(ctx context.Context) error {
		e, err := cron.NewEvent(ctx)
		if err != nil {
			return err
		}
		log.Println("CRON ran at:", e.ScheduledTime.Format("02-01-2006 15:04:05"))

		// Initialize monitor and start monitoring routine
		monitor := monitoring.NewMonitor(kvStorage)
		monitor.StartMonitoring()
		return nil
	}
	cron.ScheduleTaskNonBlock(task)

	// Send a ready signal to the runtime
	workers.Ready()
	log.Println("HTTP Server started!")

	// Block until the handler or task is done
	select {
	case <-workers.Done():
	case <-cron.Done():
		log.Println("Shutting down...")
	}
}

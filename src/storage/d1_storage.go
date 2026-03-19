//go:build js && wasm

package storage

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/RowenTey/gomon/src/models"
	_ "github.com/syumai/workers/cloudflare/d1"
)

// Storage defines D1-backed persistence operations.
type Storage interface {
	CreateWebsite(website models.Website) error
	GetWebsite(url string) (*models.Website, error)
	UpdateWebsite(website models.Website) error
	DeleteWebsite(url string) error
	ListWebsites(limit int) ([]models.Website, error)
	ListWebsitesDueForCheck(now int64, limit int) ([]models.Website, error)
	EnqueueWebhookDelivery(delivery models.WebhookDelivery) error
	ListDueWebhookDeliveries(now int64, limit int) ([]models.WebhookDelivery, error)
	MarkWebhookDeliverySuccess(eventID string, deliveredAt int64) error
	MarkWebhookDeliveryFailure(eventID string, nextAttemptAt int64, attemptCount int, exhausted bool, lastErr string) error
}

// D1Storage implements Storage using Cloudflare D1.
type D1Storage struct {
	db *sql.DB
}

// NewD1Storage initializes D1 storage with the specified binding name.
func NewD1Storage(bindingName string) (*D1Storage, error) {
	if bindingName == "" {
		return nil, errors.New("D1 binding name cannot be empty")
	}

	db, err := sql.Open("d1", bindingName)
	if err != nil {
		return nil, err
	}
	log.Println("D1 storage initialized!")

	return &D1Storage{
		db: db,
	}, nil
}

func (d *D1Storage) CreateWebsite(website models.Website) error {
	_, err := d.db.ExecContext(
		context.Background(),
		`INSERT INTO websites (
			url, frequency, last_checked_at, created_at, updated_at, status, response_time, status_code, error,
			webhook_enabled, webhook_url, webhook_payload_template
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		website.URL,
		website.Frequency,
		website.LastCheckedAt,
		website.CreatedAt,
		website.CreatedAt,
		website.Status,
		website.ResponseTime,
		website.StatusCode,
		website.Error,
		boolToInt(website.WebhookEnabled),
		website.WebhookURL,
		website.WebhookPayloadTemplate,
	)
	return err
}

func (d *D1Storage) GetWebsite(url string) (*models.Website, error) {
	website := models.Website{}
	err := d.db.QueryRowContext(
		context.Background(),
		`SELECT
			url, frequency, last_checked_at, created_at, status, response_time, status_code, error,
			webhook_enabled, webhook_url, webhook_payload_template
		FROM websites
		WHERE url = ?`,
		url,
	).Scan(
		&website.URL,
		&website.Frequency,
		&website.LastCheckedAt,
		&website.CreatedAt,
		&website.Status,
		&website.ResponseTime,
		&website.StatusCode,
		&website.Error,
		(*intBool)(&website.WebhookEnabled),
		&website.WebhookURL,
		&website.WebhookPayloadTemplate,
	)
	if err != nil {
		return nil, err
	}
	return &website, nil
}

func (d *D1Storage) UpdateWebsite(website models.Website) error {
	updatedAt := time.Now().Unix()
	_, err := d.db.ExecContext(
		context.Background(),
		`UPDATE websites
		SET frequency = ?, last_checked_at = ?, updated_at = ?, status = ?, response_time = ?, status_code = ?, error = ?,
			webhook_enabled = ?, webhook_url = ?, webhook_payload_template = ?
		WHERE url = ?`,
		website.Frequency,
		website.LastCheckedAt,
		updatedAt,
		website.Status,
		website.ResponseTime,
		website.StatusCode,
		website.Error,
		boolToInt(website.WebhookEnabled),
		website.WebhookURL,
		website.WebhookPayloadTemplate,
		website.URL,
	)
	return err
}

func (d *D1Storage) DeleteWebsite(url string) error {
	_, err := d.db.ExecContext(context.Background(), `DELETE FROM websites WHERE url = ?`, url)
	return err
}

func (d *D1Storage) ListWebsites(limit int) ([]models.Website, error) {
	rows, err := d.db.QueryContext(
		context.Background(),
		`SELECT
			url, frequency, last_checked_at, created_at, status, response_time, status_code, error,
			webhook_enabled, webhook_url, webhook_payload_template
		FROM websites
		ORDER BY created_at DESC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	websites := make([]models.Website, 0)
	for rows.Next() {
		website := models.Website{}
		if err := rows.Scan(
			&website.URL,
			&website.Frequency,
			&website.LastCheckedAt,
			&website.CreatedAt,
			&website.Status,
			&website.ResponseTime,
			&website.StatusCode,
			&website.Error,
			(*intBool)(&website.WebhookEnabled),
			&website.WebhookURL,
			&website.WebhookPayloadTemplate,
		); err != nil {
			return nil, err
		}
		websites = append(websites, website)
	}

	return websites, rows.Err()
}

func (d *D1Storage) ListWebsitesDueForCheck(now int64, limit int) ([]models.Website, error) {
	rows, err := d.db.QueryContext(
		context.Background(),
		`SELECT
			url, frequency, last_checked_at, created_at, status, response_time, status_code, error,
			webhook_enabled, webhook_url, webhook_payload_template
		FROM websites
		WHERE last_checked_at = 0 OR (? - last_checked_at) >= frequency
		ORDER BY last_checked_at ASC
		LIMIT ?`,
		now,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	websites := make([]models.Website, 0)
	for rows.Next() {
		website := models.Website{}
		if err := rows.Scan(
			&website.URL,
			&website.Frequency,
			&website.LastCheckedAt,
			&website.CreatedAt,
			&website.Status,
			&website.ResponseTime,
			&website.StatusCode,
			&website.Error,
			(*intBool)(&website.WebhookEnabled),
			&website.WebhookURL,
			&website.WebhookPayloadTemplate,
		); err != nil {
			return nil, err
		}
		websites = append(websites, website)
	}

	return websites, rows.Err()
}

func (d *D1Storage) EnqueueWebhookDelivery(delivery models.WebhookDelivery) error {
	_, err := d.db.ExecContext(
		context.Background(),
		`INSERT OR IGNORE INTO webhook_deliveries (
			event_id, website_url, webhook_url, payload, attempt_count, max_attempts,
			initial_delay_sec, max_delay_sec, backoff_factor, next_attempt_at,
			delivered_at, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		delivery.EventID,
		delivery.WebsiteURL,
		delivery.WebhookURL,
		delivery.Payload,
		delivery.AttemptCount,
		delivery.MaxAttempts,
		delivery.InitialDelaySec,
		delivery.MaxDelaySec,
		delivery.BackoffFactor,
		delivery.NextAttemptAt,
		delivery.DeliveredAt,
		delivery.LastError,
		delivery.CreatedAt,
		delivery.UpdatedAt,
	)
	return err
}

func (d *D1Storage) ListDueWebhookDeliveries(now int64, limit int) ([]models.WebhookDelivery, error) {
	rows, err := d.db.QueryContext(
		context.Background(),
		`SELECT event_id, website_url, webhook_url, payload, attempt_count, max_attempts,
			initial_delay_sec, max_delay_sec, backoff_factor, next_attempt_at,
			delivered_at, last_error, created_at, updated_at
		FROM webhook_deliveries
		WHERE delivered_at = 0 AND next_attempt_at <= ? AND attempt_count < max_attempts
		ORDER BY next_attempt_at ASC
		LIMIT ?`,
		now,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deliveries := make([]models.WebhookDelivery, 0)
	for rows.Next() {
		delivery := models.WebhookDelivery{}
		if err := rows.Scan(
			&delivery.EventID,
			&delivery.WebsiteURL,
			&delivery.WebhookURL,
			&delivery.Payload,
			&delivery.AttemptCount,
			&delivery.MaxAttempts,
			&delivery.InitialDelaySec,
			&delivery.MaxDelaySec,
			&delivery.BackoffFactor,
			&delivery.NextAttemptAt,
			&delivery.DeliveredAt,
			&delivery.LastError,
			&delivery.CreatedAt,
			&delivery.UpdatedAt,
		); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, delivery)
	}

	return deliveries, rows.Err()
}

func (d *D1Storage) MarkWebhookDeliverySuccess(eventID string, deliveredAt int64) error {
	_, err := d.db.ExecContext(
		context.Background(),
		`UPDATE webhook_deliveries
		SET delivered_at = ?, updated_at = ?, last_error = ''
		WHERE event_id = ?`,
		deliveredAt,
		deliveredAt,
		eventID,
	)
	return err
}

func (d *D1Storage) MarkWebhookDeliveryFailure(eventID string, nextAttemptAt int64, attemptCount int, exhausted bool, lastErr string) error {
	updatedAt := time.Now().Unix()
	deliveredAt := int64(0)
	if exhausted {
		nextAttemptAt = 0
		deliveredAt = -1
	}
	_, err := d.db.ExecContext(
		context.Background(),
		`UPDATE webhook_deliveries
		SET attempt_count = ?, next_attempt_at = ?, delivered_at = ?, last_error = ?, updated_at = ?
		WHERE event_id = ?`,
		attemptCount,
		nextAttemptAt,
		deliveredAt,
		lastErr,
		updatedAt,
		eventID,
	)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

type intBool bool

func (b *intBool) Scan(src any) error {
	if src == nil {
		*b = false
		return nil
	}

	switch value := src.(type) {
	case int64:
		*b = value != 0
		return nil
	case int:
		*b = value != 0
		return nil
	default:
		return errors.New("invalid bool value type")
	}
}

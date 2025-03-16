//go:build js && wasm

package storage

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/syumai/workers/cloudflare/kv"
)

// KVStorage defines the interface for KV storage operations
type KVStorage interface {
	// String operations
	GetString(key string) (string, error)
	PutString(key string, value string) error

	// JSON operations
	GetJSON(key string, value interface{}) error
	PutJSON(key string, value interface{}) error

	// Delete operation
	Delete(key string) error

	// List operations
	List(prefix string, limit int) ([]string, error)
}

// CloudflareKV implements KVStorage interface using Cloudflare KV
type CloudflareKV struct {
	namespace *kv.Namespace
}

// NewKVStorage initializes a new KV storage with the specified namespace
func NewKVStorage(namespaceName string) (*CloudflareKV, error) {
	if namespaceName == "" {
		return nil, errors.New("namespace name cannot be empty")
	}

	namespace, err := kv.NewNamespace(namespaceName)
	if err != nil {
		return nil, err
	}
	log.Println("KV storage initialized!")

	return &CloudflareKV{
		namespace: namespace,
	}, nil
}

// GetString retrieves a string value from KV storage
func (c *CloudflareKV) GetString(key string) (string, error) {
	return c.namespace.GetString(key, nil)
}

// PutString stores a string value in KV storage
func (c *CloudflareKV) PutString(key string, value string) error {
	return c.namespace.PutString(key, value, nil)
}

// GetJSON retrieves a JSON value from KV storage and unmarshals it
func (c *CloudflareKV) GetJSON(key string, value any) error {
	jsonStr, err := c.GetString(key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(jsonStr), value)
}

// PutJSON marshals and stores a JSON value in KV storage
func (c *CloudflareKV) PutJSON(key string, value any) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.PutString(key, string(jsonData))
}

// Delete removes a value from KV storage
func (c *CloudflareKV) Delete(key string) error {
	return c.namespace.Delete(key)
}

// List gets keys with the given prefix
func (c *CloudflareKV) List(prefix string, limit int) ([]string, error) {
	result, err := c.namespace.List(&kv.ListOptions{
		Prefix: prefix,
		Limit:  limit,
	})

	if err != nil {
		return nil, err
	}

	keys := make([]string, len(result.Keys))
	for i, key := range result.Keys {
		keys[i] = key.Name
	}

	return keys, nil
}

package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResellerStructure(t *testing.T) {
	now := time.Now()
	reseller := Reseller{
		ID:        "reseller-123",
		Name:      "Test Reseller",
		Email:     "reseller@example.com",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "admin-456",
	}

	assert.Equal(t, "reseller-123", reseller.ID)
	assert.Equal(t, "Test Reseller", reseller.Name)
	assert.Equal(t, "reseller@example.com", reseller.Email)
	assert.Equal(t, "active", reseller.Status)
	assert.Equal(t, now, reseller.CreatedAt)
	assert.Equal(t, now, reseller.UpdatedAt)
	assert.Equal(t, "admin-456", reseller.CreatedBy)
}

func TestResellerJSONSerialization(t *testing.T) {
	now := time.Now()
	reseller := Reseller{
		ID:        "json-reseller-456",
		Name:      "JSON Test Reseller",
		Email:     "json@example.com",
		Status:    "suspended",
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "json-admin-123",
	}

	jsonData, err := json.Marshal(reseller)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledReseller Reseller
	err = json.Unmarshal(jsonData, &unmarshaledReseller)
	assert.NoError(t, err)

	assert.Equal(t, reseller.ID, unmarshaledReseller.ID)
	assert.Equal(t, reseller.Name, unmarshaledReseller.Name)
	assert.Equal(t, reseller.Email, unmarshaledReseller.Email)
	assert.Equal(t, reseller.Status, unmarshaledReseller.Status)
	assert.Equal(t, reseller.CreatedAt.Unix(), unmarshaledReseller.CreatedAt.Unix())
	assert.Equal(t, reseller.UpdatedAt.Unix(), unmarshaledReseller.UpdatedAt.Unix())
	assert.Equal(t, reseller.CreatedBy, unmarshaledReseller.CreatedBy)
}

package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDistributorStructure(t *testing.T) {
	now := time.Now()
	distributor := Distributor{
		ID:        "distributor-123",
		Name:      "Test Distributor",
		Email:     "distributor@example.com",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "admin-456",
	}

	assert.Equal(t, "distributor-123", distributor.ID)
	assert.Equal(t, "Test Distributor", distributor.Name)
	assert.Equal(t, "distributor@example.com", distributor.Email)
	assert.Equal(t, "active", distributor.Status)
	assert.Equal(t, now, distributor.CreatedAt)
	assert.Equal(t, now, distributor.UpdatedAt)
	assert.Equal(t, "admin-456", distributor.CreatedBy)
}

func TestDistributorJSONSerialization(t *testing.T) {
	now := time.Now()
	distributor := Distributor{
		ID:        "json-distributor-456",
		Name:      "JSON Test Distributor",
		Email:     "json@example.com",
		Status:    "suspended",
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "json-admin-123",
	}

	jsonData, err := json.Marshal(distributor)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledDistributor Distributor
	err = json.Unmarshal(jsonData, &unmarshaledDistributor)
	assert.NoError(t, err)

	assert.Equal(t, distributor.ID, unmarshaledDistributor.ID)
	assert.Equal(t, distributor.Name, unmarshaledDistributor.Name)
	assert.Equal(t, distributor.Email, unmarshaledDistributor.Email)
	assert.Equal(t, distributor.Status, unmarshaledDistributor.Status)
	assert.Equal(t, distributor.CreatedAt.Unix(), unmarshaledDistributor.CreatedAt.Unix())
	assert.Equal(t, distributor.UpdatedAt.Unix(), unmarshaledDistributor.UpdatedAt.Unix())
	assert.Equal(t, distributor.CreatedBy, unmarshaledDistributor.CreatedBy)
}

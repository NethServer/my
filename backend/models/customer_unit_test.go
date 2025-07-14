package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCustomerStructure(t *testing.T) {
	now := time.Now()
	customer := Customer{
		ID:        "customer-123",
		Name:      "Test Customer",
		Email:     "customer@example.com",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "admin-789",
	}

	assert.Equal(t, "customer-123", customer.ID)
	assert.Equal(t, "Test Customer", customer.Name)
	assert.Equal(t, "customer@example.com", customer.Email)
	assert.Equal(t, "active", customer.Status)
	assert.Equal(t, now, customer.CreatedAt)
	assert.Equal(t, now, customer.UpdatedAt)
	assert.Equal(t, "admin-789", customer.CreatedBy)
}

func TestCustomerJSONSerialization(t *testing.T) {
	now := time.Now()
	customer := Customer{
		ID:        "json-customer-456",
		Name:      "JSON Test Customer",
		Email:     "json@example.com",
		Status:    "suspended",
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "json-admin-123",
	}

	jsonData, err := json.Marshal(customer)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledCustomer Customer
	err = json.Unmarshal(jsonData, &unmarshaledCustomer)
	assert.NoError(t, err)

	assert.Equal(t, customer.ID, unmarshaledCustomer.ID)
	assert.Equal(t, customer.Name, unmarshaledCustomer.Name)
	assert.Equal(t, customer.Email, unmarshaledCustomer.Email)
	assert.Equal(t, customer.Status, unmarshaledCustomer.Status)
	assert.Equal(t, customer.CreatedAt.Unix(), unmarshaledCustomer.CreatedAt.Unix())
	assert.Equal(t, customer.UpdatedAt.Unix(), unmarshaledCustomer.UpdatedAt.Unix())
	assert.Equal(t, customer.CreatedBy, unmarshaledCustomer.CreatedBy)
}

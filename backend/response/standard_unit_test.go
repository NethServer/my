package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	tests := []struct {
		name            string
		createFunc      func() Response
		expectedCode    int
		expectedMessage string
		expectedData    interface{}
	}{
		{
			name: "Success response",
			createFunc: func() Response {
				return Success(200, "Operation successful", map[string]string{"result": "success"})
			},
			expectedCode:    200,
			expectedMessage: "Operation successful",
			expectedData:    map[string]string{"result": "success"},
		},
		{
			name: "Error response",
			createFunc: func() Response {
				return Error(500, "Internal error occurred", map[string]string{"error": "details"})
			},
			expectedCode:    500,
			expectedMessage: "Internal error occurred",
			expectedData:    map[string]string{"error": "details"},
		},
		{
			name: "OK response",
			createFunc: func() Response {
				return OK("Request processed successfully", "test data")
			},
			expectedCode:    200,
			expectedMessage: "Request processed successfully",
			expectedData:    "test data",
		},
		{
			name: "Created response",
			createFunc: func() Response {
				return Created("Resource created successfully", map[string]interface{}{"id": 123})
			},
			expectedCode:    201,
			expectedMessage: "Resource created successfully",
			expectedData:    map[string]interface{}{"id": 123},
		},
		{
			name: "BadRequest response",
			createFunc: func() Response {
				return BadRequest("Invalid request parameters", nil)
			},
			expectedCode:    400,
			expectedMessage: "Invalid request parameters",
			expectedData:    nil,
		},
		{
			name: "Unauthorized response",
			createFunc: func() Response {
				return Unauthorized("Authentication required", nil)
			},
			expectedCode:    401,
			expectedMessage: "Authentication required",
			expectedData:    nil,
		},
		{
			name: "Forbidden response",
			createFunc: func() Response {
				return Forbidden("Access denied", map[string]string{"reason": "insufficient permissions"})
			},
			expectedCode:    403,
			expectedMessage: "Access denied",
			expectedData:    map[string]string{"reason": "insufficient permissions"},
		},
		{
			name: "NotFound response",
			createFunc: func() Response {
				return NotFound("Resource not found", nil)
			},
			expectedCode:    404,
			expectedMessage: "Resource not found",
			expectedData:    nil,
		},
		{
			name: "Conflict response",
			createFunc: func() Response {
				return Conflict("Resource already exists", map[string]string{"field": "username"})
			},
			expectedCode:    409,
			expectedMessage: "Resource already exists",
			expectedData:    map[string]string{"field": "username"},
		},
		{
			name: "UnprocessableEntity response",
			createFunc: func() Response {
				return UnprocessableEntity("Validation failed", []string{"field1", "field2"})
			},
			expectedCode:    422,
			expectedMessage: "Validation failed",
			expectedData:    []string{"field1", "field2"},
		},
		{
			name: "InternalServerError response",
			createFunc: func() Response {
				return InternalServerError("System error occurred", nil)
			},
			expectedCode:    500,
			expectedMessage: "System error occurred",
			expectedData:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := tt.createFunc()

			assert.Equal(t, tt.expectedCode, response.Code)
			assert.Equal(t, tt.expectedMessage, response.Message)
			assert.Equal(t, tt.expectedData, response.Data)
		})
	}
}

func TestResponseStructure(t *testing.T) {
	// Test that Response struct has correct field types and tags
	response := Response{
		Code:    200,
		Message: "test message",
		Data:    "test data",
	}

	assert.Equal(t, 200, response.Code)
	assert.Equal(t, "test message", response.Message)
	assert.Equal(t, "test data", response.Data)
	assert.IsType(t, int(0), response.Code)
	assert.IsType(t, "", response.Message)
	// Data field is interface{} type, can hold any value
	assert.NotNil(t, response.Data) // In this case it holds "test data" string
}

func TestResponseWithNilData(t *testing.T) {
	tests := []struct {
		name            string
		createFunc      func() Response
		expectedCode    int
		expectedMessage string
	}{
		{
			name: "Success with nil data",
			createFunc: func() Response {
				return Success(200, "Success without data", nil)
			},
			expectedCode:    200,
			expectedMessage: "Success without data",
		},
		{
			name: "Error with nil data",
			createFunc: func() Response {
				return Error(400, "Error without data", nil)
			},
			expectedCode:    400,
			expectedMessage: "Error without data",
		},
		{
			name: "OK with nil data",
			createFunc: func() Response {
				return OK("OK without data", nil)
			},
			expectedCode:    200,
			expectedMessage: "OK without data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := tt.createFunc()

			assert.Equal(t, tt.expectedCode, response.Code)
			assert.Equal(t, tt.expectedMessage, response.Message)
			assert.Nil(t, response.Data)
		})
	}
}

func TestResponseWithComplexData(t *testing.T) {
	complexData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":     123,
			"name":   "Test User",
			"active": true,
			"roles":  []string{"admin", "user"},
			"metadata": map[string]string{
				"department": "IT",
				"location":   "Milan",
			},
		},
		"permissions": []string{"read", "write", "delete"},
		"timestamp":   "2025-01-01T00:00:00Z",
	}

	response := OK("Complex data response", complexData)

	assert.Equal(t, 200, response.Code)
	assert.Equal(t, "Complex data response", response.Message)
	assert.Equal(t, complexData, response.Data)

	// Verify complex data structure is preserved
	data, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)

	user, ok := data["user"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 123, user["id"])
	assert.Equal(t, "Test User", user["name"])
	assert.Equal(t, true, user["active"])

	roles, ok := user["roles"].([]string)
	assert.True(t, ok)
	assert.Contains(t, roles, "admin")
	assert.Contains(t, roles, "user")
}

func TestResponseWithEmptyStrings(t *testing.T) {
	tests := []struct {
		name            string
		createFunc      func() Response
		expectedCode    int
		expectedMessage string
		expectedData    interface{}
	}{
		{
			name: "Success with empty message",
			createFunc: func() Response {
				return Success(200, "", "some data")
			},
			expectedCode:    200,
			expectedMessage: "",
			expectedData:    "some data",
		},
		{
			name: "Error with empty message",
			createFunc: func() Response {
				return Error(400, "", nil)
			},
			expectedCode:    400,
			expectedMessage: "",
			expectedData:    nil,
		},
		{
			name: "OK with empty string data",
			createFunc: func() Response {
				return OK("Success message", "")
			},
			expectedCode:    200,
			expectedMessage: "Success message",
			expectedData:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := tt.createFunc()

			assert.Equal(t, tt.expectedCode, response.Code)
			assert.Equal(t, tt.expectedMessage, response.Message)
			assert.Equal(t, tt.expectedData, response.Data)
		})
	}
}

func TestAllHTTPStatusCodes(t *testing.T) {
	// Test all convenience functions return correct status codes
	statusTests := []struct {
		name         string
		createFunc   func() Response
		expectedCode int
	}{
		{"OK", func() Response { return OK("test", nil) }, 200},
		{"Created", func() Response { return Created("test", nil) }, 201},
		{"BadRequest", func() Response { return BadRequest("test", nil) }, 400},
		{"Unauthorized", func() Response { return Unauthorized("test", nil) }, 401},
		{"Forbidden", func() Response { return Forbidden("test", nil) }, 403},
		{"NotFound", func() Response { return NotFound("test", nil) }, 404},
		{"Conflict", func() Response { return Conflict("test", nil) }, 409},
		{"UnprocessableEntity", func() Response { return UnprocessableEntity("test", nil) }, 422},
		{"InternalServerError", func() Response { return InternalServerError("test", nil) }, 500},
	}

	for _, tt := range statusTests {
		t.Run(tt.name, func(t *testing.T) {
			response := tt.createFunc()
			assert.Equal(t, tt.expectedCode, response.Code)
			assert.Equal(t, "test", response.Message)
			assert.Nil(t, response.Data)
		})
	}
}

func TestResponseConsistency(t *testing.T) {
	// Test that Success and Error functions are used consistently by convenience functions

	// Test Success-based functions
	okResponse := OK("test", "data")
	successResponse := Success(200, "test", "data")
	assert.Equal(t, successResponse, okResponse)

	createdResponse := Created("test", "data")
	successCreatedResponse := Success(201, "test", "data")
	assert.Equal(t, successCreatedResponse, createdResponse)

	// Test Error-based functions
	badRequestResponse := BadRequest("test", "data")
	errorBadRequestResponse := Error(400, "test", "data")
	assert.Equal(t, errorBadRequestResponse, badRequestResponse)

	unauthorizedResponse := Unauthorized("test", "data")
	errorUnauthorizedResponse := Error(401, "test", "data")
	assert.Equal(t, errorUnauthorizedResponse, unauthorizedResponse)

	internalErrorResponse := InternalServerError("test", "data")
	errorInternalResponse := Error(500, "test", "data")
	assert.Equal(t, errorInternalResponse, internalErrorResponse)
}

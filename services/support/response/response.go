/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package response

import "net/http"

// Response is the standard API response format
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Success creates a success response
func Success(code int, message string, data interface{}) Response {
	return Response{Code: code, Message: message, Data: data}
}

// Error creates an error response
func Error(code int, message string, data interface{}) Response {
	return Response{Code: code, Message: message, Data: data}
}

// OK creates a 200 response
func OK(message string, data interface{}) Response {
	return Success(http.StatusOK, message, data)
}

// Created creates a 201 response
func Created(message string, data interface{}) Response {
	return Success(http.StatusCreated, message, data)
}

// BadRequest creates a 400 response
func BadRequest(message string, data interface{}) Response {
	return Error(http.StatusBadRequest, message, data)
}

// Unauthorized creates a 401 response
func Unauthorized(message string, data interface{}) Response {
	return Error(http.StatusUnauthorized, message, data)
}

// Forbidden creates a 403 response
func Forbidden(message string, data interface{}) Response {
	return Error(http.StatusForbidden, message, data)
}

// NotFound creates a 404 response
func NotFound(message string, data interface{}) Response {
	return Error(http.StatusNotFound, message, data)
}

// InternalServerError creates a 500 response
func InternalServerError(message string, data interface{}) Response {
	return Error(http.StatusInternalServerError, message, data)
}

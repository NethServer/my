package middleware

import (
	"net/http"

	"github.com/fatih/structs"
	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, structs.Map(response.StatusUnauthorized{
				Code:    401,
				Message: "user not found in context",
				Data:    nil,
			}))
			c.Abort()
			return
		}

		user, ok := userInterface.(*models.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
				Code:    500,
				Message: "invalid user context",
				Data:    nil,
			}))
			c.Abort()
			return
		}

		hasRole := false
		for _, userRole := range user.Roles {
			if userRole == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient permissions",
				Data:    nil,
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, structs.Map(response.StatusUnauthorized{
				Code:    401,
				Message: "user not found in context",
				Data:    nil,
			}))
			c.Abort()
			return
		}

		user, ok := userInterface.(*models.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
				Code:    500,
				Message: "invalid user context",
				Data:    nil,
			}))
			c.Abort()
			return
		}

		hasScope := false
		for _, userScope := range user.Scopes {
			if userScope == scope {
				hasScope = true
				break
			}
		}

		if !hasScope {
			c.JSON(http.StatusForbidden, structs.Map(response.StatusForbidden{
				Code:    403,
				Message: "insufficient scope",
				Data:    nil,
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

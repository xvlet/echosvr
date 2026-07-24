package custom

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// LoginHandler handles POST /test/api/login
// Returns a JSON payload with token and userId for Stateful extraction testing
func LoginHandler(c echo.Context) error {
	// JMeter stateful test expects $.token and regex "userId":\s*"([^"]+)"
	// #nosec G101 - Mock token for testing purposes
	return c.JSON(http.StatusOK, map[string]interface{}{
		"token":  "mock-token-xyz789",
		"userId": "user-test-001",
		"status": "success",
	})
}

// ProfileHandler handles GET /test/api/profile
// Echoes back the extracted userId in the response
func ProfileHandler(c echo.Context) error {
	userId := c.QueryParam("userId")
	if userId == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing userId query parameter",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"userId":  userId,
		"status":  "active",
		"message": fmt.Sprintf("Profile for %s successfully loaded", userId),
	})
}

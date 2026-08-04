package custom

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// GetCustomHandler returns a custom echo.HandlerFunc for specific routes if defined.
// Returns nil if no custom handler is configured for the given method and path.
func GetCustomHandler(method, path string) echo.HandlerFunc {
	if path == "/test/api/login" && method == "POST" {
		return LoginHandler
	}
	if path == "/test/api/profile" && method == "GET" {
		return ProfileHandler
	}
	if path == "/test/api/sse" && method == "GET" {
		return SSEHandler
	}
	return nil
}

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

// SSEHandler handles GET /test/api/sse
// Simulates an SSE stream sending a message every 1 second
func SSEHandler(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")
	// Flush headers immediately
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Flush()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	counter := 1

	for {
		select {
		case <-c.Request().Context().Done():
			// Client disconnected
			return nil
		case <-ticker.C:
			msg := fmt.Sprintf("id: %d\nevent: message\ndata: {\"counter\": %d, \"timestamp\": \"%s\"}\n\n", counter, counter, time.Now().Format(time.RFC3339))
			if _, err := c.Response().Write([]byte(msg)); err != nil {
				return err
			}
			c.Response().Flush()
			counter++
		}
	}
}

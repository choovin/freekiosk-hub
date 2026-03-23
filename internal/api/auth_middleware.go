package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// AuthMiddleware protects routes that require authentication
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skip auth for login page and static assets
		path := c.Path()
		if path == "/login" || path == "/swagger" || path == "/swagger/" || path == "/swaggerui" {
			return next(c)
		}

		// Skip auth for API docs and health checks
		if path == "/swagger/doc.json" || path == "/health" || path == "/health/mqtt" {
			return next(c)
		}

		// Skip auth for login POST action
		if path == "/login" && c.Request().Method == http.MethodPost {
			return next(c)
		}

		// Check for session cookie
		cookie, err := c.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			// Check if it's an HTMX request
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized",
				})
			}
			return c.Redirect(http.StatusFound, "/login")
		}

		// Store username in context for later use
		c.Set("username", cookie.Value)

		return next(c)
	}
}

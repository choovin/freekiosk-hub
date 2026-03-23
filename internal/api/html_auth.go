package api

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/wared2003/freekiosk-hub/internal/i18n"
	"github.com/wared2003/freekiosk-hub/ui"

	"github.com/labstack/echo/v4"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "freekiosk_session"
	// SessionDuration is how long the session lasts
	SessionDuration = 24 * time.Hour
)

// HtmlAuthHandler handles authentication-related HTML pages
type HtmlAuthHandler struct {
	username string
	password string
}

func NewHtmlAuthHandler(username, password string) *HtmlAuthHandler {
	return &HtmlAuthHandler{
		username: username,
		password: password,
	}
}

// HandleLogin renders the login page
func (h *HtmlAuthHandler) HandleLogin(c echo.Context) error {
	// If already authenticated, redirect to home
	if _, err := c.Cookie(SessionCookieName); err == nil {
		return c.Redirect(http.StatusFound, "/")
	}

	lang, _ := c.Get("lang").(string)
	if lang == "" {
		lang = "en"
	}

	return c.Render(http.StatusOK, "", ui.LoginPage(lang, ""))
}

// HandleLoginSubmit processes the login form submission
func (h *HtmlAuthHandler) HandleLoginSubmit(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	lang, _ := c.Get("lang").(string)
	if lang == "" {
		lang = "en"
	}
	t := func(key string) string { return i18n.TL(lang, key) }

	// Constant-time comparison to prevent timing attacks
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(h.username)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(h.password)) == 1

	if !usernameMatch || !passwordMatch {
		return c.Render(http.StatusOK, "", ui.LoginPage(lang, t("auth.invalid_credentials")))
	}

	// Create session cookie
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    username + ":" + time.Now().Format(time.RFC3339),
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(SessionDuration.Seconds()),
	}
	http.SetCookie(c.Response(), cookie)

	return c.Redirect(http.StatusFound, "/")
}

// HandleLogout logs out the user
func (h *HtmlAuthHandler) HandleLogout(c echo.Context) error {
	// Clear the session cookie
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(c.Response(), cookie)

	return c.Redirect(http.StatusFound, "/login")
}

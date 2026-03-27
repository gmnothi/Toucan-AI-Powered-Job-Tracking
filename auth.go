package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOAuthConfig *oauth2.Config
	sessionStore      *sessions.CookieStore
)

const sessionName = "toucan_session"

func InitAuth() {
	sessionStore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}

	googleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "https://jobtracker-api.fly.dev/auth/callback",
		Scopes:       []string{"openid", "email"},
		Endpoint:     google.Endpoint,
	}
}

func HandleGoogleLogin(c *gin.Context) {
	b := make([]byte, 16)
	rand.Read(b)
	state := hex.EncodeToString(b)

	session, _ := sessionStore.Get(c.Request, sessionName)
	session.Values["oauth_state"] = state
	session.Save(c.Request, c.Writer)

	url := googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func HandleGoogleCallback(c *gin.Context) {
	session, _ := sessionStore.Get(c.Request, sessionName)

	savedState, _ := session.Values["oauth_state"].(string)
	if savedState == "" || savedState != c.Query("state") {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid state"})
		return
	}
	delete(session.Values, "oauth_state")

	token, err := googleOAuthConfig.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "token exchange failed"})
		return
	}

	client := googleOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	json.Unmarshal(body, &userInfo)

	// Discard token — only need the user identity
	userID, err := UpsertUser(userInfo.ID, userInfo.Email)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to save user"})
		return
	}

	session.Values["user_id"] = userID
	session.Save(c.Request, c.Writer)

	frontendURL := os.Getenv("FRONTEND_URL")
	c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}

func HandleLogout(c *gin.Context) {
	session, _ := sessionStore.Get(c.Request, sessionName)
	session.Options.MaxAge = -1
	session.Save(c.Request, c.Writer)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func HandleMe(c *gin.Context) {
	session, _ := sessionStore.Get(c.Request, sessionName)
	userID, ok := session.Values["user_id"].(int)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"authenticated": false})
		return
	}
	email, err := GetUserEmail(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"authenticated": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"authenticated": true, "email": email})
}

func RequireAuth(c *gin.Context) {
	session, _ := sessionStore.Get(c.Request, sessionName)
	userID, ok := session.Values["user_id"].(int)
	if !ok || userID == 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	c.Set("user_id", userID)
	c.Next()
}

func GetSessionUserID(c *gin.Context) int {
	id, _ := c.Get("user_id")
	userID, _ := id.(int)
	return userID
}

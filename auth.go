package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleOAuthConfig *oauth2.Config

func InitAuth() {
	googleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "https://jobtracker-api.fly.dev/auth/callback",
		Scopes:       []string{"openid", "email", "https://www.googleapis.com/auth/gmail.readonly"},
		Endpoint:     google.Endpoint,
	}
}

// --- Token helpers ---

type tokenPayload struct {
	UserID    int   `json:"uid"`
	ExpiresAt int64 `json:"exp"`
}

func secret() []byte {
	return []byte(os.Getenv("SESSION_SECRET"))
}

func signToken(userID int) (string, error) {
	p := tokenPayload{UserID: userID, ExpiresAt: time.Now().Add(7 * 24 * time.Hour).Unix()}
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return payload + "." + sig, nil
}

func verifyToken(token string) (int, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid token")
	}
	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(parts[0]))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return 0, fmt.Errorf("invalid signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, err
	}
	var p tokenPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return 0, err
	}
	if time.Now().Unix() > p.ExpiresAt {
		return 0, fmt.Errorf("token expired")
	}
	return p.UserID, nil
}

// --- State helpers (HMAC-signed, no server storage needed) ---

func generateOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(b)
	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(nonce))
	sig := hex.EncodeToString(mac.Sum(nil))
	return nonce + "." + sig, nil
}

func validateOAuthState(state string) bool {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) != 2 {
		return false
	}
	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(parts[0]))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(parts[1]), []byte(expected))
}

// --- Handlers ---

func HandleGoogleLogin(c *gin.Context) {
	state, err := generateOAuthState()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "state generation failed"})
		return
	}
	url := googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func HandleGoogleCallback(c *gin.Context) {
	if !validateOAuthState(c.Query("state")) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid state"})
		return
	}

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

	userID, err := UpsertUser(userInfo.ID, userInfo.Email, token.RefreshToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to save user"})
		return
	}

	sessionToken, err := signToken(userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"?token="+sessionToken)
}

func HandleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func HandleMe(c *gin.Context) {
	userID := getUserIDFromRequest(c)
	if userID == 0 {
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
	userID := getUserIDFromRequest(c)
	if userID == 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	c.Set("user_id", userID)
	c.Next()
}

func getUserIDFromRequest(c *gin.Context) int {
	auth := c.GetHeader("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return 0
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	userID, err := verifyToken(token)
	if err != nil {
		return 0
	}
	return userID
}

func GetSessionUserID(c *gin.Context) int {
	id, _ := c.Get("user_id")
	userID, _ := id.(int)
	return userID
}

func GetOAuthConfig() *oauth2.Config {
	return googleOAuthConfig
}

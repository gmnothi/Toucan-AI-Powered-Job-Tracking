package main

import (
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func StartWebServer() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{os.Getenv("FRONTEND_URL")},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	r.LoadHTMLGlob("templates/*")

	// Auth routes (public)
	r.GET("/auth/google", HandleGoogleLogin)
	r.GET("/auth/callback", HandleGoogleCallback)
	r.POST("/auth/logout", HandleLogout)
	r.GET("/auth/me", HandleMe)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app")
	})

	// Logo proxy (public — no sensitive data)
	r.GET("/api/logo", func(c *gin.Context) {
		domain := c.Query("domain")
		if domain == "" {
			c.Status(http.StatusBadRequest)
			return
		}
		key := os.Getenv("LOGO_DEV")
		url := "https://img.logo.dev/" + domain + "?token=" + key
		resp, err := http.Get(url)
		if err != nil || resp.StatusCode != http.StatusOK {
			c.Status(http.StatusNotFound)
			return
		}
		defer resp.Body.Close()
		c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
	})

	// Protected API routes
	api := r.Group("/api")
	api.Use(RequireAuth)
	{
		api.GET("/logs", func(c *gin.Context) {
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("X-Accel-Buffering", "no")

			ch := logBus.Subscribe()
			defer logBus.Unsubscribe(ch)

			c.Stream(func(w io.Writer) bool {
				select {
				case msg, ok := <-ch:
					if !ok {
						return false
					}
					c.SSEvent("log", msg)
					return true
				case <-c.Request.Context().Done():
					return false
				}
			})
		})

		api.POST("/pause", func(c *gin.Context) {
			Progress.mu.Lock()
			Progress.Paused = !Progress.Paused
			paused := Progress.Paused
			Progress.mu.Unlock()
			c.JSON(http.StatusOK, gin.H{"paused": paused})
		})

		api.GET("/status", func(c *gin.Context) {
			Progress.mu.RLock()
			defer Progress.mu.RUnlock()
			c.JSON(http.StatusOK, gin.H{
				"running":   Progress.Running,
				"total":     Progress.Total,
				"processed": Progress.Processed,
				"saved":     Progress.Saved,
				"account":   Progress.Account,
			})
		})

		api.POST("/refresh", func(c *gin.Context) {
			userID := GetSessionUserID(c)
			var body struct {
				Since string `json:"since"`
			}
			c.ShouldBindJSON(&body)

			var since time.Time
			if body.Since != "" {
				var err error
				since, err = time.Parse("2006-01-02", body.Since)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
					return
				}
			}

			go CheckInboxSinceForUser(since, userID)
			c.JSON(http.StatusOK, gin.H{"message": "Refresh started"})
		})

		api.GET("/jobs", func(c *gin.Context) {
			userID := GetSessionUserID(c)
			jobs := GetJobsForUser(userID)
			c.JSON(http.StatusOK, jobs)
		})

		api.PUT("/jobs/:id/status", func(c *gin.Context) {
			userID := GetSessionUserID(c)
			id := c.Param("id")
			var body struct {
				Status string `json:"status"`
			}
			if err := c.ShouldBindJSON(&body); err != nil || body.Status == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
				return
			}
			if err := UpdateJobStatus(id, body.Status, userID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
		})

		api.DELETE("/jobs/:id", func(c *gin.Context) {
			userID := GetSessionUserID(c)
			id := c.Param("id")
			if err := DeleteJob(id, userID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
		})
	}

	r.Static("/app", "./frontend/dist")
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/app") {
			c.File("./frontend/dist/index.html")
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		}
	})

	r.Run(":8080")
}

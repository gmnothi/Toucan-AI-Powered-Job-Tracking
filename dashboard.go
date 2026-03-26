package main

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func StartWebServer() {
	r := gin.Default()

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
	}))

	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app")
	})

	r.GET("/api/logs", func(c *gin.Context) {
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

	r.POST("/api/pause", func(c *gin.Context) {
		Progress.mu.Lock()
		Progress.Paused = !Progress.Paused
		paused := Progress.Paused
		Progress.mu.Unlock()
		c.JSON(http.StatusOK, gin.H{"paused": paused})
	})

	r.GET("/api/status", func(c *gin.Context) {
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

	r.POST("/api/refresh", func(c *gin.Context) {
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

		go CheckInboxSince(since)
		c.JSON(http.StatusOK, gin.H{"message": "Refresh started"})
	})

	r.GET("/api/jobs", func(c *gin.Context) {
		jobs := GetAllJobs()
		c.JSON(http.StatusOK, jobs)
	})

	r.PUT("/api/jobs/:id/status", func(c *gin.Context) {
		id := c.Param("id")
		var body struct {
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Status == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		err := UpdateJobStatus(id, body.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
	})

	r.DELETE("/api/jobs/:id", func(c *gin.Context) {
		id := c.Param("id")
		err := DeleteJob(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
	})

	// Serve React frontend (built with Vite) under /app/*
	r.Static("/app", "./frontend/dist") // Make sure this path is correct

	// Serve index.html for any unknown frontend route (e.g. /app/dashboard)
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/app") {
			c.File("./frontend/dist/index.html")
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		}
	})

	r.Run(":8080")
}

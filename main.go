package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	log.Println("GMAIL_USER =", os.Getenv("GMAIL_USER"))
}

func main() {
	TestOpenAI()
	InitDB()

	go CheckInbox()

	c := cron.New()
	c.AddFunc("@every 20m", CheckInbox)
	c.Start()

	StartWebServer()
}

package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load() // no-op in production where env vars are set directly
	log.Println("GMAIL_USER =", os.Getenv("GMAIL_USER"))
}

func main() {
	TestOpenAI()
	InitDB()
	StartWebServer()
}

package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load()
	log.Println("GMAIL_USER =", os.Getenv("GMAIL_USER"))
}

func main() {
	TestOpenAI()
	InitDB()
	InitAuth()
	StartWebServer()
}

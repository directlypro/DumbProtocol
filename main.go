package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
)

const ServiceName = "DumbProtocol"

func main() {
	// Service initialization
	fmt.Printf("Starting the %v Service\n", ServiceName)

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbURL := os.Getenv("DATABASE")
	if dbURL == "" {
		fmt.Println("DATABASE environment variable not set")
	}

	fmt.Printf("Connecting to the %v Database\n", dbURL)

}

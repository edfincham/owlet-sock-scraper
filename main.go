package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	apiPkg "github.com/edfincham/owlet-sock-scraper/api"
	db "github.com/edfincham/owlet-sock-scraper/db"
	"github.com/edfincham/owlet-sock-scraper/db/models"
	sockPkg "github.com/edfincham/owlet-sock-scraper/sock"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file. Ensure the environment variables are set.")
	}

	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	pgdb := os.Getenv("POSTGRES_DB")

	postgresURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, pgdb)

	dbConn, err := db.NewDB(context.Background(), postgresURL)
	if err != nil {
		log.Fatalf("Error starting the database %v", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	err = dbConn.Migrate("db/migrations")
	if err != nil {
		log.Fatalf("Error running migrations: %v", err)
		os.Exit(1)
	}

	api, err := apiPkg.NewOwletAPI(os.Getenv("OWLET_REGION"), os.Getenv("OWLET_USERNAME"), os.Getenv("OWLET_PASSWORD"))
	if err != nil {
		log.Fatalf("Failed to create OwletAPI client: %v", err)
	}

	err = api.Authenticate()
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	devicesResponse, err := api.GetDevices()
	if err != nil {
		log.Fatalf("Failed to get devices: %v", err)
	}

	devices := devicesResponse["response"].([]apiPkg.Device)
	var socks []*sockPkg.Sock
	for _, device := range devices {
		sock := sockPkg.NewSock(api, device)
		socks = append(socks, sock)
		log.Printf("Initialized sock: %s", sock.Serial)
	}

	const (
		baseDelay     = 5 * time.Second
		maxDelay      = 15 * time.Minute
		backoffFactor = 2
	)

	currentDelay := baseDelay
	attemptCount := 0

	for {
		for _, sock := range socks {
			vitals, err := sock.UpdateVitals()
			if err != nil {
				log.Fatalf("Failed to update vitals for sock %s: %v", sock.Serial, err)
			}

			if vitals.Chg > 1 {
				log.Printf("Sock %s is charging. Backing off for %v (attempt %d)", sock.Serial, currentDelay, attemptCount)
				delay := time.Duration(float64(baseDelay) * math.Pow(backoffFactor, float64(attemptCount)))
				if delay > maxDelay {
					delay = maxDelay
				}

				currentDelay = delay
				attemptCount++
			} else {
				currentDelay = baseDelay
				attemptCount = 0

				err = models.InsertVitals(dbConn, *vitals, sock.Serial)
				if err != nil {
					log.Fatalf("Failed to insert vitals: %v", err)
				}
			}
		}

		time.Sleep(currentDelay)
	}
}

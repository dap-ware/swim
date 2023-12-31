package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"database/sql"

	swimStream "github.com/dap-ware/swim/certstream"
	swimConfig "github.com/dap-ware/swim/config"
	swimDb "github.com/dap-ware/swim/database"
	swimModels "github.com/dap-ware/swim/models"
	swimServer "github.com/dap-ware/swim/server"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Determine base directory
	baseDir := filepath.Join(os.Getenv("HOME"), "swim-framework")

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Failed to create base directory: %v", err)
	}

	// Define and create subdirectories
	logDir := filepath.Join(baseDir, "logs")
	configDir := filepath.Join(baseDir, "config")
	dataDir := filepath.Join(baseDir, "data")

	for _, dir := range []string{logDir, configDir, dataDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Log file setup
	logFilePath := filepath.Join(logDir, "log.txt")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	multi := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(multi)

	// Configuration file setup
	configPath := filepath.Join(configDir, "config.json")
	var swimCfg *swimConfig.Config

	// Check if the config file exists
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		log.Println("Config file not found, using default configuration")
		swimCfg = swimConfig.GetDefaultConfig()
		// Adjust the database file path to be within the dataDir
		swimCfg.Database.FilePath = filepath.Join(dataDir, "swim.db")
	} else if err != nil {
		log.Fatalf("Error checking config file: %v", err)
	} else {
		log.Println("Loading configuration from file")
		swimCfg, err = swimConfig.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		// Adjust the database file path if necessary
		if !filepath.IsAbs(swimCfg.Database.FilePath) {
			swimCfg.Database.FilePath = filepath.Join(dataDir, swimCfg.Database.FilePath)
		}
	}

	// Define the directory and paths for SSL/TLS certificates
	certDir := filepath.Join(baseDir, "cert")
	certFile := filepath.Join(certDir, "cert.pem")
	keyFile := filepath.Join(certDir, "key.pem")

	// Check if cert directory exists, and create it if not
	if err := os.MkdirAll(certDir, 0755); err != nil {
		log.Fatalf("Failed to create cert directory: %v", err)
	}

	// Check if cert.pem and key.pem exist
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		printInstructions(baseDir)
		return // or generate the certificates if you can automate this
	}

	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		printInstructions(baseDir)
		return // or generate the certificates if you can automate this
	}

	// Database setup
	dbPath := swimCfg.Database.FilePath

	var db *sql.DB

	// check if the database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// create a new file
		file, err := os.Create(swimCfg.Database.FilePath)
		if err != nil {
			log.Fatalf("Failed to create database file: %v", err)
		}
		file.Close()

		// open the newly created database
		db, err = sql.Open("sqlite3", swimCfg.Database.FilePath)
		if err != nil {
			log.Fatalf("Error opening new database: %v", err)
		}
		defer db.Close()

		// initialize the database
		if err := swimDb.SetupDatabase(db); err != nil {
			log.Fatalf("Failed to setup database: %v", err)
		}
	} else {
		// open the existing database
		db, err = sql.Open("sqlite3", swimCfg.Database.FilePath)
		if err != nil {
			log.Fatalf("Error opening database: %v", err)
		}
		defer db.Close()
	}

	domains := make(chan []swimModels.CertUpdateInfo, 100) // buffered channel for domain info
	rawMessages := make(chan []byte, 100)                  // buffered channel for raw messages
	stopProcessing := make(chan struct{})                  // channel to signal stopping of processing

	var wg sync.WaitGroup

	// start the database insert worker
	wg.Add(1)
	go swimDb.DbInsertWorker(db, domains, &wg)

	// start the message processing worker
	wg.Add(1)
	go swimStream.MessageProcessor(rawMessages, domains, stopProcessing, &wg, swimCfg.Database.BatchSize)

	// goroutine for CertStream connection
	wg.Add(1)
	go swimStream.ListenForEvents(rawMessages, stopProcessing, &wg)

	// server gets started in go routine in swimServer.StartServer
	srv, started := swimServer.StartServer(db, &wg, swimCfg, baseDir) // start the Gin server (with a rate limiter of 100 requests per hour. See config/config.yaml for the
	// wait for the server to start
	go func() {
		<-started // send a message to the channel when the server is started
	}()

	// signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// wait for interrupt signal
	<-sigs
	fmt.Println("Shutting down gracefully...")

	// graceful shutdown of the Gin server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	// signal to stop processing and close channels
	close(stopProcessing)
	close(rawMessages)
	close(domains)

	wg.Wait()
	fmt.Println("CertStream data processing completed.")
}

// printInstructions provides instructions for generating SSL/TLS certificates
func printInstructions(baseDir string) {
	certDir := filepath.Join(baseDir, "cert")
	certFile := filepath.Join(certDir, "cert.pem")
	keyFile := filepath.Join(certDir, "key.pem")

	fmt.Println("The SSL/TLS certificates were not found.")
	fmt.Println("Please generate the certificates using OpenSSL with the following commands:")
	fmt.Printf("mkdir -p %s\n", certDir)
	fmt.Printf("openssl req -newkey rsa:2048 -nodes -keyout %s -x509 -days 365 -out %s\n", keyFile, certFile)
	fmt.Println("This will create the necessary certificates in the 'cert' directory of the 'swim-framework'.")
	fmt.Println("After generating the certificates, try running the program again.")
}

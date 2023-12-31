package main

import (
	"context"
	"flag"
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

var (
	help = flag.Bool("h", false, "Display help")
)

// function to print SSL/TLS cert instructions if not found
func printInstructions() {
	fmt.Println("The 'cert' directory does not exist or is missing required files.")
	fmt.Println("Please run the following commands:")
	fmt.Println("mkdir -p cert")
	fmt.Println("cd cert")
	fmt.Println("Generate the certificates here. For example, using OpenSSL:")
	fmt.Println("openssl req -newkey rsa:2048 -nodes -keyout key.pem -x509 -days 365 -out cert.pem")
	fmt.Println("Then, try running the program again.")
}

func main() {
	// Determine the directory for the log file
	logDir := filepath.Dir("logs/log.txt")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		// Create the directory if it does not exist
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Fatalf("Failed to create directory for log file: %v", err)
		}
	}

	// Now, open the log file
	logFile, err := os.OpenFile("logs/log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// create a multi-writer that writes to both the log file and the terminal
	multi := io.MultiWriter(logFile, os.Stdout)

	// set the log output to the multi-writer
	log.SetOutput(multi)

	swimCfg, err := swimConfig.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}

	// Print the loaded configuration for debugging
	fmt.Printf("Loaded configuration: %+v\n", swimCfg)

	flag.Parse()

	if *help {
		fmt.Println("CertStream Data Processor")
		fmt.Println("\nThis program connects to the CertStream service, processes incoming domain data, and stores it in a SQLite database.")
		fmt.Println("\nUsage information and program description")
		flag.PrintDefaults()
		return
	}

	certDir := "cert"
	certFile := filepath.Join(certDir, "cert.pem")
	keyFile := filepath.Join(certDir, "key.pem")

	// check if cert directory exists
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		printInstructions()
		return
	}

	// check if cert.pem and key.pem exist
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		printInstructions()
		return
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		printInstructions()
		return
	}

	// ensure the directory for the database file exists
	dbDir := filepath.Dir(swimCfg.Database.FilePath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			log.Fatalf("Failed to create directory for database: %v", err)
		}
	}

	var db *sql.DB

	// check if the database file exists
	if _, err := os.Stat(swimCfg.Database.FilePath); os.IsNotExist(err) {
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
	srv, started := swimServer.StartServer(db, &wg, swimCfg) // start the Gin server (with a rate limiter of 100 requests per hour. See config/config.yaml for the
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

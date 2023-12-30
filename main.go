package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
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

func main() {
	// open a file for logging
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

	db, err := sql.Open("sqlite3", swimCfg.Database.FilePath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if swimDb.SetupDatabase(db); err != nil {
		log.Fatal(err)
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

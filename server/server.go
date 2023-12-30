package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	swimDb "github.com/dap-ware/swim/database"
	swimModels "github.com/dap-ware/swim/models"
	"github.com/gin-gonic/gin"
)

// StartServer starts the Gin server in a separate goroutine.
func StartServer(db *sql.DB, wg *sync.WaitGroup) (*http.Server, chan struct{}) {
	// setup Gin server in a separate goroutine
	r := gin.Default()
	server := &swimModels.Server{Db: db}
	r.GET("/v1/domains", func(c *gin.Context) { GetDomainNamesHandler(server, c) })

	// define the handler functions
	r.GET("/v1/get/domains", func(c *gin.Context) {
		// get the page number and size from the query parameters
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("size", "1000"))

		// fetch the domains from the database
		domains, err := swimDb.FetchDomainData(server.Db, page, size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occurred",
				"error":   err.Error(),
			})
			return
		}

		// send the domains as the response
		c.JSON(http.StatusOK, domains)
	})

	r.GET("/v1/get/:domain/subdomains", func(c *gin.Context) {
		// get the domain name from the path parameters
		domain := c.Param("domain")

		// fetch the domain and its subdomains from the database
		domainWithSubdomains, err := swimDb.FetchDomainWithSubdomains(server.Db, domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occurred",
				"error":   err.Error(),
			})
			return
		}

		// send the domain with its subdomains as the response
		c.JSON(http.StatusOK, domainWithSubdomains)
	})

	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: r,
	}
	// create a channel to signal when the server has started
	started := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
		close(started) // close the channel to signal that the server has started
	}()

	return srv, started
}

func GetDomainNamesHandler(s *swimModels.Server, c *gin.Context) {
	// get the page number and size from the query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "1000"))

	// create a channel to receive the domain names
	domainNames := make(chan []string)

	// start a goroutine to fetch the domain names from the database
	go func() {
		defer close(domainNames)
		if err := swimDb.FetchDomainNamesFromDatabase(s.Db, domainNames, page, size); err != nil {
			log.Printf("Error fetching domain names from database: %v", err)
		}
	}()

	// set the header to indicate that the response is a stream
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)

	// create a new JSON encoder that writes to the response body
	encoder := json.NewEncoder(c.Writer)

	// Iterate over the domain names and write them to the response
	for chunk := range domainNames {
		if err := encoder.Encode(chunk); err != nil {
			log.Printf("Error encoding domain names: %v", err)
			return
		}
	}
}

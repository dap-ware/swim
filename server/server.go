package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	swimDb "github.com/dap-ware/swim/database"
	swimModels "github.com/dap-ware/swim/models"
	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	visits map[string]*visitData
	mu     sync.Mutex
}

type visitData struct {
	count      int
	lastUpdate time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		visits: make(map[string]*visitData),
	}
}

func (rl *RateLimiter) RateLimit(limit int, resetTime time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		rl.mu.Lock()
		data, visited := rl.visits[clientIP]
		if !visited {
			rl.visits[clientIP] = &visitData{count: 1, lastUpdate: time.Now()}
			rl.mu.Unlock()
			c.Next()
			return
		}

		if time.Since(data.lastUpdate) > resetTime {
			data.count = 1
			data.lastUpdate = time.Now()
		} else {
			data.count++
			if data.count > limit {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
				rl.mu.Unlock()
				return
			}
		}
		rl.mu.Unlock()
		c.Next()
	}
}

// StartServer starts the Gin server in a separate goroutine.
func StartServer(db *sql.DB, wg *sync.WaitGroup) (*http.Server, chan struct{}) {
	// get new rate limiter
	rateLimiter := NewRateLimiter()

	// create a new Gin server
	r := gin.Default()

	// sse the rate limiter middleware with a limit of 100 requests per hour
	r.Use(rateLimiter.RateLimit(100, 1*time.Hour))

	server := &swimModels.Server{Db: db}

	// handler for fetching all domain names
	r.GET("/v1/domains", func(c *gin.Context) {
		page, size, err := parseQueryParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		GetDomainNamesHandler(server, c, page, size)
	})

	// handler for fetching certificate updates
	r.GET("/v1/cert-updates", func(c *gin.Context) {
		page, size, err := parseQueryParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		domains, err := swimDb.FetchDomainData(server.Db, page, size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cert updates"})
			return
		}

		c.JSON(http.StatusOK, domains)
	})

	// handler for fetching subdomains
	r.GET("/v1/subdomains/:domain", func(c *gin.Context) {
		domain := c.Param("domain")
		domainWithSubdomains, err := swimDb.FetchDomainWithSubdomains(server.Db, domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subdomains"})
			return
		}

		c.JSON(http.StatusOK, domainWithSubdomains)
	})

	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: r,
	}

	started := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
		close(started)
	}()

	return srv, started
}

// GetDomainNamesHandler handles requests to fetch domain names.
func GetDomainNamesHandler(s *swimModels.Server, c *gin.Context, page int, size int) {
	domainNames := make(chan []string)
	go func() {
		defer close(domainNames)
		if err := swimDb.FetchDomainNamesFromDatabase(s.Db, domainNames, page, size); err != nil {
			log.Printf("Error fetching domain names from database: %v", err)
		}
	}()

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(c.Writer)
	for chunk := range domainNames {
		if err := encoder.Encode(chunk); err != nil {
			log.Printf("Error encoding domain names: %v", err)
			return
		}
	}
}

// parseQueryParams parses and validates query parameters.
func parseQueryParams(c *gin.Context) (int, int, error) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		return 0, 0, err
	}
	size, err := strconv.Atoi(c.DefaultQuery("size", "1000"))
	if err != nil {
		return 0, 0, err
	}
	return page, size, nil
}

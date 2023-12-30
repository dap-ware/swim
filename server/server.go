package server

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	swimConfig "github.com/dap-ware/swim/config"
	swimDb "github.com/dap-ware/swim/database"
	swimModels "github.com/dap-ware/swim/models"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	visits    map[string]*visitData
	mu        sync.Mutex
	resetTime time.Duration
	limit     int
}

type visitData struct {
	count      int
	lastUpdate time.Time
}

func NewRateLimiter(limit int, resetTime time.Duration) *RateLimiter {
	return &RateLimiter{
		visits:    make(map[string]*visitData),
		limit:     limit,
		resetTime: resetTime,
	}
}

func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
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

		// calculate the allowed count using exponential backoff
		allowedCount := int(math.Pow(2, float64(data.count-1)))

		if time.Since(data.lastUpdate) > rl.resetTime {
			data.count = 1
			data.lastUpdate = time.Now()
		} else if data.count > allowedCount {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			rl.mu.Unlock()
			return
		} else {
			data.count++
		}

		rl.mu.Unlock()
		c.Next()
	}
}

// StartServer starts the Gin server in a separate goroutine.
func StartServer(db *sql.DB, wg *sync.WaitGroup, swimCfg *swimConfig.Config) (*http.Server, chan struct{}) {
	// get new rate limiter
	rateLimiter := NewRateLimiter(swimCfg.Rate.Limit, swimCfg.Rate.ResetTime)

	// create a new Gin server
	r := gin.Default()

	// configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"} // allow requests from localhost:3000 (react frontend)
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	r.Use(cors.New(config))

	// sse the rate limiter middleware with a limit of 100 requests per hour
	r.Use(rateLimiter.RateLimit())

	// handle OPTIONS requests
	r.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

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

		GetCertUpdatesHandler(server, c, page, size)
	})

	// handler for fetching subdomains
	r.GET("/v1/subdomains/:domain", func(c *gin.Context) {
		domain := c.Param("domain")
		GetSubdomainsHandler(server, c, domain)
	})

	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: r,
		// TLS configuration
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	started := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Change ListenAndServe to ListenAndServeTLS and specify cert and key files
		if err := srv.ListenAndServeTLS("cert/cert.pem", "cert/key.pem"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
		close(started)
	}()

	return srv, started
}

func StreamResponse[T interface{}](c *gin.Context, dataChan chan []T, encodeFunc func(*json.Encoder, []T) error) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(c.Writer)

	for chunk := range dataChan {
		if err := encodeFunc(encoder, chunk); err != nil {
			log.Printf("Error encoding response: %v", err)
			return
		}
	}
}

func GetDomainNamesHandler(s *swimModels.Server, c *gin.Context, page int, size int) {
	domainNames := make(chan []string)
	go func() {
		defer close(domainNames)
		if err := swimDb.FetchDomainNamesFromDatabase(s.Db, domainNames, page, size); err != nil {
			log.Printf("Error fetching domain names from database: %v", err)
		}
	}()

	StreamResponse(c, domainNames, func(enc *json.Encoder, chunk []string) error {
		return enc.Encode(chunk)
	})
}

func GetCertUpdatesHandler(s *swimModels.Server, c *gin.Context, page int, size int) {
	certUpdatesChan := make(chan []swimModels.CertUpdateInfo)
	go func() {
		defer close(certUpdatesChan)
		updates, err := swimDb.FetchCertUpdatesFromDatabase(s.Db, page, size)
		if err != nil {
			log.Printf("Error fetching certificate updates from database: %v", err)
			return
		}
		certUpdatesChan <- updates
	}()

	StreamResponse(c, certUpdatesChan, func(enc *json.Encoder, chunk []swimModels.CertUpdateInfo) error {
		return enc.Encode(chunk)
	})
}

func GetSubdomainsHandler(s *swimModels.Server, c *gin.Context, domain string) {
	subdomains := make(chan []swimModels.DomainWithSubdomains)
	go func() {
		defer close(subdomains)
		subs, err := swimDb.FetchSubdomainsFromDatabase(s.Db, domain)
		if err != nil {
			log.Printf("Error fetching subdomains from database: %v", err)
			return
		}
		subdomains <- []swimModels.DomainWithSubdomains{*subs}
	}()

	StreamResponse(c, subdomains, func(enc *json.Encoder, chunk []swimModels.DomainWithSubdomains) error {
		return enc.Encode(chunk)
	})
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

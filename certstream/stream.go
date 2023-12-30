package certstream

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	swimModels "github.com/dap-ware/swim/models"
	"github.com/gorilla/websocket"
)

const (
	certStreamURL = "wss://certstream.calidog.io/"
)

// function to establish a new WebSocket connection
func ConnectToCertStream() (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(certStreamURL, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	return c, nil
}

// ListenForEvents connects to CertStream and listens for events, sending raw messages to the provided channel.
// it stops processing when it receives a signal on the stopProcessing channel.
func ListenForEvents(rawMessages chan []byte, stopProcessing chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-stopProcessing:
			return // stop this goroutine
		default:
			c, err := ConnectToCertStream()
			if err != nil {
				log.Printf("Error connecting to CertStream: %v. Retrying in 5 seconds...", err)
				time.Sleep(5 * time.Second)
				continue
			}
			//log.Println("Connected to CertStream. Listening for events...")

			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					//log.Printf("Error reading message: %v. Reconnecting...", err)
					c.Close()
					break
				}
				rawMessages <- message
			}
		}
	}
}

// messageProcessor processes raw messages and sends extracted domain info to the domains channel
func MessageProcessor(rawMessages chan []byte, domains chan []swimModels.DomainInfo, stopProcessing chan struct{}, wg *sync.WaitGroup, batchSize int) {
	defer wg.Done()

	var batch []swimModels.DomainInfo
	for message := range rawMessages {
		var m map[string]interface{}
		if err := json.Unmarshal(message, &m); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		if data, ok := m["data"].(map[string]interface{}); ok {
			if leafCert, ok := data["leaf_cert"].(map[string]interface{}); ok {
				domainFlags := make(map[string]bool) // map to flag wildcard domains

				if domainsList, ok := leafCert["all_domains"].([]interface{}); ok {
					for _, domainInterface := range domainsList {
						if domain, ok := domainInterface.(string); ok {
							if strings.HasPrefix(domain, "*.") {
								// mark the corresponding non-wildcard domain as having a wildcard
								trimmedDomain := strings.TrimPrefix(domain, "*.")
								domainFlags[trimmedDomain] = true
								continue // skip adding wildcard domains as separate entries
							}

							var domainInfo swimModels.DomainInfo
							domainInfo.Domain = domain
							domainInfo.NotBefore = int64(leafCert["not_before"].(float64))
							domainInfo.NotAfter = int64(leafCert["not_after"].(float64))
							domainInfo.SerialNumber = leafCert["serial_number"].(string)
							domainInfo.Fingerprint = leafCert["fingerprint"].(string)

							// extracting additional fields from the extensions object
							if extensions, ok := leafCert["extensions"].(map[string]interface{}); ok {
								if keyUsage, ok := extensions["keyUsage"].(string); ok {
									domainInfo.KeyUsage = keyUsage
								}
								if extendedKeyUsage, ok := extensions["extendedKeyUsage"].(string); ok {
									domainInfo.ExtendedKeyUsage = extendedKeyUsage
								}
								if subjectKeyID, ok := extensions["subjectKeyIdentifier"].(string); ok {
									domainInfo.SubjectKeyID = subjectKeyID
								}
								if authorityKeyID, ok := extensions["authorityKeyIdentifier"].(string); ok {
									domainInfo.AuthorityKeyID = authorityKeyID
								}
								if authorityInfo, ok := extensions["authorityInfoAccess"].(string); ok {
									domainInfo.AuthorityInfo = authorityInfo
								}
								if subjectAltName, ok := extensions["subjectAltName"].(string); ok {
									domainInfo.SubjectAltName = subjectAltName
								}
								if certificatePolicies, ok := extensions["certificatePolicies"].(string); ok {
									domainInfo.CertificatePolicies = certificatePolicies
								}
							}

							// set wildcard flag based on earlier processing
							domainInfo.Wildcard = domainFlags[domain]

							batch = append(batch, domainInfo)
						}
					}
				}
			}
		}

		// send the batch if it reaches the specified size
		if len(batch) >= batchSize {
			domains <- batch
			batch = make([]swimModels.DomainInfo, 0) // reset batch
		}
	}

	// send any remaining domains in the batch
	if len(batch) > 0 {
		domains <- batch
	}
}

package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	swimModels "github.com/dap-ware/swim/models"
)

func SetupDatabase(db *sql.DB) error {
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS domains (
        id INTEGER PRIMARY KEY,
        domain TEXT NOT NULL UNIQUE,
		is_apex BOOLEAN NOT NULL,
		parent_domain TEXT,
        not_before INTEGER,
        not_after INTEGER,
        serial_number TEXT,
        fingerprint TEXT,
        key_usage TEXT,
        extended_key_usage TEXT,
        subject_key_id TEXT,
        authority_key_id TEXT,
        authority_info TEXT,
        subject_alt_name TEXT,
        certificate_policies TEXT,
        wildcard BOOLEAN
    );`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("error creating domains table: %w", err)
	}

	// check if the parent_domain column exists
	rows, err := db.Query("PRAGMA table_info(domains);")
	if err != nil {
		return fmt.Errorf("error getting domains table info: %w", err)
	}
	defer rows.Close()

	hasParentDomain := false
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notnull bool
		var dfltValue *string
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notnull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("error scanning row: %w", err)
		}
		if name == "parent_domain" {
			hasParentDomain = true
			break
		}
	}

	// if the parent_domain column doesn't exist, add it
	if !hasParentDomain {
		_, err := db.Exec("ALTER TABLE domains ADD COLUMN parent_domain TEXT;")
		if err != nil {
			return fmt.Errorf("error adding parent_domain column: %w", err)
		}
	}

	return nil
}

// dbInsertWorker is responsible for batch inserting domains into the database
func DbInsertWorker(db *sql.DB, domains chan []swimModels.DomainInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	for batch := range domains {
		var err error
		for attempt := 0; attempt < 3; attempt++ { // retry up to 3 times
			// start a transaction
			tx, err := db.Begin()
			if err != nil {
				log.Printf("Error starting transaction: %v", err)
				continue
			}

			err = insertBatch(tx, batch)
			if err == nil {
				// commit the transaction if there was no error
				if err := tx.Commit(); err != nil {
					log.Printf("Error committing transaction: %v", err)
				}
				break
			} else {
				// rollback the transaction if there was an error
				if err := tx.Rollback(); err != nil {
					log.Printf("Error rolling back transaction: %v", err)
				}
			}

			log.Printf("Retry %d: Error inserting batch: %v", attempt+1, err)
			time.Sleep(time.Second * 2) // wait for 2 seconds before retrying
		}
		if err != nil {
			log.Printf("Final error after retries: %v", err)
		}
	}
}

func insertBatch(tx *sql.Tx, batch []swimModels.DomainInfo) error {
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO domains (domain, not_before, not_after, serial_number, fingerprint, key_usage, extended_key_usage, subject_key_id, authority_key_id, authority_info, subject_alt_name, certificate_policies, wildcard, is_apex, parent_domain) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, domainInfo := range batch {

		// check if the domain is an apex domain
		domainInfo.IsApex = isApexDomain(domainInfo.Domain)

		// determine the parent domain
		parentDomain := getParentDomain(domainInfo.Domain)

		_, err = stmt.Exec(domainInfo.Domain, domainInfo.NotBefore, domainInfo.NotAfter, domainInfo.SerialNumber, domainInfo.Fingerprint, domainInfo.KeyUsage, domainInfo.ExtendedKeyUsage, domainInfo.SubjectKeyID, domainInfo.AuthorityKeyID, domainInfo.AuthorityInfo, domainInfo.SubjectAltName, domainInfo.CertificatePolicies, domainInfo.Wildcard, domainInfo.IsApex, parentDomain)
		if err != nil {
			return err
		}
	}

	return nil
}

func FetchDomainData(db *sql.DB, page, size int) ([]swimModels.DomainInfo, error) {
	// calculate the offset
	offset := (page - 1) * size

	// prepare the SQL query
	query := `SELECT * FROM domains ORDER BY domain LIMIT ? OFFSET ?`

	// execute the query
	rows, err := db.Query(query, size, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// scan the result into a slice of Domain structs
	var domains []swimModels.DomainInfo
	for rows.Next() {
		var domain swimModels.DomainInfo
		err := rows.Scan(
			&domain.ID,
			&domain.Domain,
			&domain.IsApex,
			&domain.ParentDomain,
			&domain.NotBefore,
			&domain.NotAfter,
			&domain.SerialNumber,
			&domain.Fingerprint,
			&domain.KeyUsage,
			&domain.ExtendedKeyUsage,
			&domain.SubjectKeyID,
			&domain.AuthorityKeyID,
			&domain.AuthorityInfo,
			&domain.SubjectAltName,
			&domain.CertificatePolicies,
			&domain.Wildcard,
		)
		if err != nil {
			return nil, err
		}

		// convert not_before to a human-readable time
		domain.NotBeforeTime = time.Unix(domain.NotBefore, 0).Format(time.RFC3339)

		domains = append(domains, domain)
	}

	// check for errors from iterating over rows.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return domains, nil
}

func FetchDomainWithSubdomains(db *sql.DB, domain string) (*swimModels.DomainWithSubdomains, error) {
	// query for the subdomains
	rows, err := db.Query("SELECT domain FROM domains WHERE parent_domain = ?", domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// scan the rows into a slice
	var subdomains []string
	for rows.Next() {
		var subdomain string
		if err := rows.Scan(&subdomain); err != nil {
			return nil, err
		}
		subdomains = append(subdomains, subdomain)
	}

	// check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &swimModels.DomainWithSubdomains{
		Domain:     domain,
		Subdomains: subdomains,
	}, nil
}

func FetchDomainNamesFromDatabase(db *sql.DB, domainNamesCh chan<- []string, page int, size int) error {
	// calculate the offset based on the page number and size
	offset := (page - 1) * size

	// define the SQL query with LIMIT and OFFSET clauses
	// select only domains that are marked as apex and do not start with 'www.'
	query := fmt.Sprintf("SELECT domain FROM domains WHERE is_apex = true AND domain NOT LIKE 'www.%%' LIMIT %d OFFSET %d;", size, offset)

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Database query error: %v", err)
		return err
	}

	var domains []string

	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			log.Printf("Error scanning row: %v", err)
			rows.Close() // close the rows before returning
			return err
		}
		domains = append(domains, domain)
	}

	// close the rows
	rows.Close()

	// send the chunk of domain names to the channel
	domainNamesCh <- domains

	return nil
}

func CheckDatabaseSize(dbPath, maxSizeStr string) bool {
	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		log.Printf("Error stating database file: %v", err)
		return false
	}

	currentSize := fileInfo.Size()
	maxSize, err := parseSize(maxSizeStr)
	if err != nil {
		log.Printf("Error parsing max size: %v", err)
		return false
	}
	return currentSize <= maxSize
}

func parseSize(sizeStr string) (int64, error) {
	var size int64
	var unit string

	_, err := fmt.Sscanf(sizeStr, "%d%s", &size, &unit)
	if err != nil {
		return 0, err
	}

	switch strings.ToUpper(unit) {
	case "M":
		size *= 1024 * 1024
	case "G":
		size *= 1024 * 1024 * 1024
	// Add more units as necessary
	default:
		return 0, fmt.Errorf("unknown unit %s", unit)
	}

	return size, nil
}

// isApexDomain checks if the given domain is an apex domain
func isApexDomain(domain string) bool {
	// Count the number of dots in the domain
	dotCount := strings.Count(domain, ".")
	return dotCount == 1
}

// getParentDomain extracts the parent domain if possible
func getParentDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return ""
}

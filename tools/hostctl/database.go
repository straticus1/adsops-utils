package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// initDB initializes the database connection
func initDB() error {
	if db != nil {
		return nil
	}

	host := getEnvOrDefault("INVENTORY_DB_HOST", "afterdarksys.com")
	port := getEnvOrDefault("INVENTORY_DB_PORT", "5432")
	dbname := getEnvOrDefault("INVENTORY_DB_NAME", "inventory")
	user := getEnvOrDefault("INVENTORY_DB_USER", "")
	password := getEnvOrDefault("INVENTORY_DB_PASSWORD", "")

	if user == "" {
		return fmt.Errorf("INVENTORY_DB_USER environment variable is required")
	}
	if password == "" {
		return fmt.Errorf("INVENTORY_DB_PASSWORD environment variable is required")
	}

	connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=require",
		host, port, dbname, user, password)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	return nil
}

// getDB returns the database connection, initializing it if necessary
func getDB() (*sql.DB, error) {
	if err := initDB(); err != nil {
		return nil, err
	}
	return db, nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// insertResource inserts a new resource into the database
func insertResource(opts *AddOptions) (*Resource, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	// Parse owners and mail groups
	owners := parseOwners(opts.Owners)
	mailGroups := parseOwners(opts.MailGroups)

	// Build metadata
	metadata := map[string]interface{}{}
	if opts.IP != "" {
		metadata["ip"] = opts.IP
	}
	if opts.Size != "" {
		metadata["size"] = opts.Size
	}
	if opts.Shape != "" {
		metadata["shape"] = opts.Shape
	}

	// Add custom tags if provided
	if opts.Tags != "" {
		tags, err := parseTags(opts.Tags)
		if err != nil {
			return nil, err
		}
		for k, v := range tags {
			metadata[k] = v
		}
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %v", err)
	}

	ownersJSON, err := json.Marshal(owners)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal owners: %v", err)
	}

	mailGroupsJSON, err := json.Marshal(mailGroups)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mailgroups: %v", err)
	}

	query := `
		INSERT INTO inventory_resources (
			resource_name, hostname, type, provider, region, status, environment,
			owners, mailgroups, metadata, average_daily_cost, average_monthly_cost,
			external_id, external_url, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, resource_name, hostname, type, provider, region, status, environment,
			owners, mailgroups, metadata, average_daily_cost, average_monthly_cost,
			external_id, external_url, created_at, updated_at
	`

	resource := &Resource{}
	var ownersData, mailGroupsData, metadataData []byte
	var region, externalID, externalURL sql.NullString
	var costDaily, costMonthly sql.NullFloat64

	if opts.Region != "" {
		region.String = opts.Region
		region.Valid = true
	}
	if opts.ExternalID != "" {
		externalID.String = opts.ExternalID
		externalID.Valid = true
	}
	if opts.ExternalURL != "" {
		externalURL.String = opts.ExternalURL
		externalURL.Valid = true
	}
	if opts.CostDaily > 0 {
		costDaily.Float64 = opts.CostDaily
		costDaily.Valid = true
	}
	if opts.CostMonthly > 0 {
		costMonthly.Float64 = opts.CostMonthly
		costMonthly.Valid = true
	}

	now := time.Now()
	err = db.QueryRow(query,
		opts.Hostname, opts.Hostname, opts.Type, opts.Provider, region, opts.Status,
		opts.Environment, ownersJSON, mailGroupsJSON, metadataJSON, costDaily, costMonthly,
		externalID, externalURL, now, now,
	).Scan(
		&resource.ID, &resource.ResourceName, &resource.Hostname, &resource.Type,
		&resource.Provider, &resource.Region, &resource.Status, &resource.Environment,
		&ownersData, &mailGroupsData, &metadataData, &resource.AverageDailyCost,
		&resource.AverageMonthlyCost, &resource.ExternalID, &resource.ExternalURL,
		&resource.CreatedAt, &resource.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert resource: %v", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal(ownersData, &resource.Owners)
	json.Unmarshal(mailGroupsData, &resource.MailGroups)
	json.Unmarshal(metadataData, &resource.Metadata)

	return resource, nil
}

// updateResource updates an existing resource
func updateResource(opts *UpdateOptions) (*Resource, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	// First, get the existing resource
	existing, err := getResourceByHostname(opts.Hostname)
	if err != nil {
		return nil, err
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if opts.Type != "" {
		updates = append(updates, fmt.Sprintf("type = $%d", argNum))
		args = append(args, opts.Type)
		argNum++
	}
	if opts.Provider != "" {
		updates = append(updates, fmt.Sprintf("provider = $%d", argNum))
		args = append(args, opts.Provider)
		argNum++
	}
	if opts.Region != "" {
		updates = append(updates, fmt.Sprintf("region = $%d", argNum))
		args = append(args, opts.Region)
		argNum++
	}
	if opts.Status != "" {
		updates = append(updates, fmt.Sprintf("status = $%d", argNum))
		args = append(args, opts.Status)
		argNum++
	}
	if opts.Environment != "" {
		updates = append(updates, fmt.Sprintf("environment = $%d", argNum))
		args = append(args, opts.Environment)
		argNum++
	}

	if opts.Owners != "" {
		owners := parseOwners(opts.Owners)
		ownersJSON, _ := json.Marshal(owners)
		updates = append(updates, fmt.Sprintf("owners = $%d", argNum))
		args = append(args, ownersJSON)
		argNum++
	}

	if opts.MailGroups != "" {
		mailGroups := parseOwners(opts.MailGroups)
		mailGroupsJSON, _ := json.Marshal(mailGroups)
		updates = append(updates, fmt.Sprintf("mailgroups = $%d", argNum))
		args = append(args, mailGroupsJSON)
		argNum++
	}

	if opts.CostDaily >= 0 {
		updates = append(updates, fmt.Sprintf("average_daily_cost = $%d", argNum))
		args = append(args, opts.CostDaily)
		argNum++
	}

	if opts.CostMonthly >= 0 {
		updates = append(updates, fmt.Sprintf("average_monthly_cost = $%d", argNum))
		args = append(args, opts.CostMonthly)
		argNum++
	}

	if opts.ExternalID != "" {
		updates = append(updates, fmt.Sprintf("external_id = $%d", argNum))
		args = append(args, opts.ExternalID)
		argNum++
	}

	if opts.ExternalURL != "" {
		updates = append(updates, fmt.Sprintf("external_url = $%d", argNum))
		args = append(args, opts.ExternalURL)
		argNum++
	}

	// Handle metadata updates
	if opts.IP != "" || opts.Size != "" || opts.Shape != "" || opts.Tags != "" {
		metadata := existing.Metadata
		if metadata == nil {
			metadata = map[string]interface{}{}
		}

		if opts.IP != "" {
			metadata["ip"] = opts.IP
		}
		if opts.Size != "" {
			metadata["size"] = opts.Size
		}
		if opts.Shape != "" {
			metadata["shape"] = opts.Shape
		}
		if opts.Tags != "" {
			tags, err := parseTags(opts.Tags)
			if err != nil {
				return nil, err
			}
			for k, v := range tags {
				metadata[k] = v
			}
		}

		metadataJSON, _ := json.Marshal(metadata)
		updates = append(updates, fmt.Sprintf("metadata = $%d", argNum))
		args = append(args, metadataJSON)
		argNum++
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Add updated_at
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	// Add hostname to WHERE clause
	args = append(args, opts.Hostname)

	query := fmt.Sprintf(`
		UPDATE inventory_resources
		SET %s
		WHERE hostname = $%d
		RETURNING id, resource_name, hostname, type, provider, region, status, environment,
			owners, mailgroups, metadata, average_daily_cost, average_monthly_cost,
			external_id, external_url, created_at, updated_at
	`, strings.Join(updates, ", "), argNum)

	resource := &Resource{}
	var ownersData, mailGroupsData, metadataData []byte

	err = db.QueryRow(query, args...).Scan(
		&resource.ID, &resource.ResourceName, &resource.Hostname, &resource.Type,
		&resource.Provider, &resource.Region, &resource.Status, &resource.Environment,
		&ownersData, &mailGroupsData, &metadataData, &resource.AverageDailyCost,
		&resource.AverageMonthlyCost, &resource.ExternalID, &resource.ExternalURL,
		&resource.CreatedAt, &resource.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update resource: %v", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal(ownersData, &resource.Owners)
	json.Unmarshal(mailGroupsData, &resource.MailGroups)
	json.Unmarshal(metadataData, &resource.Metadata)

	return resource, nil
}

// deleteResource deletes a resource by hostname
func deleteResource(hostname string) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	query := `DELETE FROM inventory_resources WHERE hostname = $1`
	result, err := db.Exec(query, hostname)
	if err != nil {
		return fmt.Errorf("failed to delete resource: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %v", err)
	}

	if rows == 0 {
		return fmt.Errorf("host not found: %s", hostname)
	}

	return nil
}

// updateResourceStatus updates the status of a resource
func updateResourceStatus(hostname, newStatus string) (*Resource, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	// Get current status for logging
	existing, err := getResourceByHostname(hostname)
	if err != nil {
		return nil, err
	}

	// Log the status change
	if err := logStatusChange(hostname, existing.Status, newStatus); err != nil {
		// Log error but don't fail the operation
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to log status change: %v\n", err)
		}
	}

	query := `
		UPDATE inventory_resources
		SET status = $1, updated_at = $2
		WHERE hostname = $3
		RETURNING id, resource_name, hostname, type, provider, region, status, environment,
			owners, mailgroups, metadata, average_daily_cost, average_monthly_cost,
			external_id, external_url, created_at, updated_at
	`

	resource := &Resource{}
	var ownersData, mailGroupsData, metadataData []byte

	err = db.QueryRow(query, newStatus, time.Now(), hostname).Scan(
		&resource.ID, &resource.ResourceName, &resource.Hostname, &resource.Type,
		&resource.Provider, &resource.Region, &resource.Status, &resource.Environment,
		&ownersData, &mailGroupsData, &metadataData, &resource.AverageDailyCost,
		&resource.AverageMonthlyCost, &resource.ExternalID, &resource.ExternalURL,
		&resource.CreatedAt, &resource.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update status: %v", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal(ownersData, &resource.Owners)
	json.Unmarshal(mailGroupsData, &resource.MailGroups)
	json.Unmarshal(metadataData, &resource.Metadata)

	return resource, nil
}

// getResourceByHostname retrieves a resource by hostname
func getResourceByHostname(hostname string) (*Resource, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, resource_name, hostname, type, provider, region, status, environment,
			owners, mailgroups, metadata, average_daily_cost, average_monthly_cost,
			external_id, external_url, created_at, updated_at
		FROM inventory_resources
		WHERE hostname = $1
	`

	resource := &Resource{}
	var ownersData, mailGroupsData, metadataData []byte

	err = db.QueryRow(query, hostname).Scan(
		&resource.ID, &resource.ResourceName, &resource.Hostname, &resource.Type,
		&resource.Provider, &resource.Region, &resource.Status, &resource.Environment,
		&ownersData, &mailGroupsData, &metadataData, &resource.AverageDailyCost,
		&resource.AverageMonthlyCost, &resource.ExternalID, &resource.ExternalURL,
		&resource.CreatedAt, &resource.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("host not found: %s", hostname)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query resource: %v", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal(ownersData, &resource.Owners)
	json.Unmarshal(mailGroupsData, &resource.MailGroups)
	json.Unmarshal(metadataData, &resource.Metadata)

	return resource, nil
}

// listResources lists resources with optional filters
func listResources(opts *ListOptions) ([]*Resource, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, resource_name, hostname, type, provider, region, status, environment,
			owners, mailgroups, metadata, average_daily_cost, average_monthly_cost,
			external_id, external_url, created_at, updated_at
		FROM inventory_resources
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if opts.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, opts.Status)
		argNum++
	}
	if opts.Environment != "" {
		query += fmt.Sprintf(" AND environment = $%d", argNum)
		args = append(args, opts.Environment)
		argNum++
	}
	if opts.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argNum)
		args = append(args, opts.Type)
		argNum++
	}
	if opts.Provider != "" {
		query += fmt.Sprintf(" AND provider = $%d", argNum)
		args = append(args, opts.Provider)
		argNum++
	}
	if opts.Region != "" {
		query += fmt.Sprintf(" AND region = $%d", argNum)
		args = append(args, opts.Region)
		argNum++
	}

	query += " ORDER BY hostname"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, opts.Limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query resources: %v", err)
	}
	defer rows.Close()

	resources := []*Resource{}
	for rows.Next() {
		resource := &Resource{}
		var ownersData, mailGroupsData, metadataData []byte

		err := rows.Scan(
			&resource.ID, &resource.ResourceName, &resource.Hostname, &resource.Type,
			&resource.Provider, &resource.Region, &resource.Status, &resource.Environment,
			&ownersData, &mailGroupsData, &metadataData, &resource.AverageDailyCost,
			&resource.AverageMonthlyCost, &resource.ExternalID, &resource.ExternalURL,
			&resource.CreatedAt, &resource.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resource: %v", err)
		}

		// Unmarshal JSON fields
		json.Unmarshal(ownersData, &resource.Owners)
		json.Unmarshal(mailGroupsData, &resource.MailGroups)
		json.Unmarshal(metadataData, &resource.Metadata)

		resources = append(resources, resource)
	}

	return resources, nil
}

// searchResources searches for resources by query string
func searchResources(query string) ([]*Resource, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	sqlQuery := `
		SELECT id, resource_name, hostname, type, provider, region, status, environment,
			owners, mailgroups, metadata, average_daily_cost, average_monthly_cost,
			external_id, external_url, created_at, updated_at
		FROM inventory_resources
		WHERE hostname ILIKE $1
			OR resource_name ILIKE $1
			OR metadata::text ILIKE $1
			OR external_id ILIKE $1
		ORDER BY hostname
		LIMIT 100
	`

	searchTerm := "%" + query + "%"
	rows, err := db.Query(sqlQuery, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search resources: %v", err)
	}
	defer rows.Close()

	resources := []*Resource{}
	for rows.Next() {
		resource := &Resource{}
		var ownersData, mailGroupsData, metadataData []byte

		err := rows.Scan(
			&resource.ID, &resource.ResourceName, &resource.Hostname, &resource.Type,
			&resource.Provider, &resource.Region, &resource.Status, &resource.Environment,
			&ownersData, &mailGroupsData, &metadataData, &resource.AverageDailyCost,
			&resource.AverageMonthlyCost, &resource.ExternalID, &resource.ExternalURL,
			&resource.CreatedAt, &resource.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resource: %v", err)
		}

		// Unmarshal JSON fields
		json.Unmarshal(ownersData, &resource.Owners)
		json.Unmarshal(mailGroupsData, &resource.MailGroups)
		json.Unmarshal(metadataData, &resource.Metadata)

		resources = append(resources, resource)
	}

	return resources, nil
}

// logStatusChange logs a status change to the database
func logStatusChange(hostname, oldStatus, newStatus string) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	// Try to create the status_changes table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS status_changes (
			id SERIAL PRIMARY KEY,
			hostname VARCHAR(255) NOT NULL,
			old_status VARCHAR(50) NOT NULL,
			new_status VARCHAR(50) NOT NULL,
			changed_at TIMESTAMP NOT NULL,
			changed_by VARCHAR(255)
		)
	`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create status_changes table: %v", err)
	}

	// Log the status change
	query := `
		INSERT INTO status_changes (hostname, old_status, new_status, changed_at, changed_by)
		VALUES ($1, $2, $3, $4, $5)
	`

	changedBy := os.Getenv("USER")
	if changedBy == "" {
		changedBy = "unknown"
	}

	_, err = db.Exec(query, hostname, oldStatus, newStatus, time.Now(), changedBy)
	if err != nil {
		return fmt.Errorf("failed to log status change: %v", err)
	}

	return nil
}

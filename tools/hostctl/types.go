package main

import (
	"database/sql"
	"time"
)

// Resource represents a host in the inventory
type Resource struct {
	ID                 int                    `json:"id"`
	ResourceName       string                 `json:"resource_name"`
	Hostname           string                 `json:"hostname"`
	Type               string                 `json:"type"`
	Provider           string                 `json:"provider"`
	Region             sql.NullString         `json:"region"`
	Status             string                 `json:"status"`
	Environment        string                 `json:"environment"`
	Owners             []string               `json:"owners"`
	MailGroups         []string               `json:"mailgroups"`
	Metadata           map[string]interface{} `json:"metadata"`
	AverageDailyCost   sql.NullFloat64        `json:"average_daily_cost"`
	AverageMonthlyCost sql.NullFloat64        `json:"average_monthly_cost"`
	ExternalID         sql.NullString         `json:"external_id"`
	ExternalURL        sql.NullString         `json:"external_url"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// AddOptions contains options for adding a host
type AddOptions struct {
	Hostname     string
	IP           string
	Type         string
	Provider     string
	Region       string
	Size         string
	Shape        string
	Environment  string
	Status       string
	Owners       string
	MailGroups   string
	CostDaily    float64
	CostMonthly  float64
	Tags         string
	ExternalID   string
	ExternalURL  string
}

// UpdateOptions contains options for updating a host
type UpdateOptions struct {
	Hostname     string
	IP           string
	Type         string
	Provider     string
	Region       string
	Size         string
	Shape         string
	Environment  string
	Status       string
	Owners       string
	MailGroups   string
	CostDaily    float64
	CostMonthly  float64
	Tags         string
	ExternalID   string
	ExternalURL  string
}

// ListOptions contains options for listing hosts
type ListOptions struct {
	Status      string
	Environment string
	Type        string
	Provider    string
	Region      string
	Limit       int
}

// StatusChange represents a status change log entry
type StatusChange struct {
	Hostname  string    `json:"hostname"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	ChangedAt time.Time `json:"changed_at"`
	ChangedBy string    `json:"changed_by"`
}

// TableColumn represents a column configuration for table output
type TableColumn struct {
	Header string
	Width  int
}

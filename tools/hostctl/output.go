package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

// printResource prints a single resource in a readable format
func printResource(r *Resource) {
	fmt.Printf("%sHostname:%s %s\n", colorBold, colorReset, r.Hostname)
	fmt.Printf("Type:        %s\n", r.Type)
	fmt.Printf("Provider:    %s\n", r.Provider)
	if r.Region.Valid {
		fmt.Printf("Region:      %s\n", r.Region.String)
	}
	fmt.Printf("Status:      %s%s%s\n", getStatusColor(r.Status), r.Status, colorReset)
	fmt.Printf("Environment: %s%s%s\n", getEnvColor(r.Environment), r.Environment, colorReset)

	if len(r.Owners) > 0 {
		fmt.Printf("Owners:      %s\n", strings.Join(r.Owners, ", "))
	}

	if r.Metadata != nil {
		if ip, ok := r.Metadata["ip"].(string); ok && ip != "" {
			fmt.Printf("IP Address:  %s\n", ip)
		}
		if size, ok := r.Metadata["size"].(string); ok && size != "" {
			fmt.Printf("Size:        %s\n", size)
		}
		if shape, ok := r.Metadata["shape"].(string); ok && shape != "" {
			fmt.Printf("Shape:       %s\n", shape)
		}
	}

	if r.AverageDailyCost.Valid && r.AverageDailyCost.Float64 > 0 {
		fmt.Printf("Daily Cost:  $%.2f\n", r.AverageDailyCost.Float64)
	}
	if r.AverageMonthlyCost.Valid && r.AverageMonthlyCost.Float64 > 0 {
		fmt.Printf("Monthly Cost: $%.2f\n", r.AverageMonthlyCost.Float64)
	}

	if r.ExternalID.Valid && r.ExternalID.String != "" {
		fmt.Printf("External ID: %s\n", r.ExternalID.String)
	}
	if r.ExternalURL.Valid && r.ExternalURL.String != "" {
		fmt.Printf("External URL: %s\n", r.ExternalURL.String)
	}

	fmt.Printf("Created:     %s\n", r.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", r.UpdatedAt.Format("2006-01-02 15:04:05"))
}

// printResourceDetailed prints detailed information about a resource
func printResourceDetailed(r *Resource) {
	fmt.Printf("\n%s=== Host Details ===%s\n\n", colorBold, colorReset)

	fmt.Printf("%sBasic Information:%s\n", colorBold, colorReset)
	fmt.Printf("  ID:          %d\n", r.ID)
	fmt.Printf("  Resource:    %s\n", r.ResourceName)
	fmt.Printf("  Hostname:    %s\n", r.Hostname)
	fmt.Printf("  Type:        %s\n", r.Type)
	fmt.Printf("  Provider:    %s\n", r.Provider)
	if r.Region.Valid {
		fmt.Printf("  Region:      %s\n", r.Region.String)
	}
	fmt.Printf("  Status:      %s%s%s\n", getStatusColor(r.Status), r.Status, colorReset)
	fmt.Printf("  Environment: %s%s%s\n", getEnvColor(r.Environment), r.Environment, colorReset)

	if len(r.Owners) > 0 || len(r.MailGroups) > 0 {
		fmt.Printf("\n%sContacts:%s\n", colorBold, colorReset)
		if len(r.Owners) > 0 {
			fmt.Printf("  Owners:      %s\n", strings.Join(r.Owners, ", "))
		}
		if len(r.MailGroups) > 0 {
			fmt.Printf("  Mail Groups: %s\n", strings.Join(r.MailGroups, ", "))
		}
	}

	if r.Metadata != nil && len(r.Metadata) > 0 {
		fmt.Printf("\n%sMetadata:%s\n", colorBold, colorReset)
		for k, v := range r.Metadata {
			fmt.Printf("  %-12s %v\n", k+":", v)
		}
	}

	if r.AverageDailyCost.Valid || r.AverageMonthlyCost.Valid {
		fmt.Printf("\n%sCost Information:%s\n", colorBold, colorReset)
		if r.AverageDailyCost.Valid {
			fmt.Printf("  Daily Cost:   $%.2f\n", r.AverageDailyCost.Float64)
		}
		if r.AverageMonthlyCost.Valid {
			fmt.Printf("  Monthly Cost: $%.2f\n", r.AverageMonthlyCost.Float64)
		}
	}

	if r.ExternalID.Valid || r.ExternalURL.Valid {
		fmt.Printf("\n%sExternal References:%s\n", colorBold, colorReset)
		if r.ExternalID.Valid {
			fmt.Printf("  External ID:  %s\n", r.ExternalID.String)
		}
		if r.ExternalURL.Valid {
			fmt.Printf("  External URL: %s\n", r.ExternalURL.String)
		}
	}

	fmt.Printf("\n%sTimestamps:%s\n", colorBold, colorReset)
	fmt.Printf("  Created:     %s\n", r.CreatedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  Updated:     %s\n", r.UpdatedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Println()
}

// printResourceTable prints resources in a table format
func printResourceTable(resources []*Resource) {
	// Define column widths
	colWidths := map[string]int{
		"hostname": 25,
		"type":     12,
		"provider": 10,
		"region":   12,
		"status":   12,
		"env":      12,
		"ip":       15,
	}

	// Print header
	fmt.Println()
	printTableSeparator(colWidths)
	fmt.Printf("| %-*s | %-*s | %-*s | %-*s | %-*s | %-*s | %-*s |\n",
		colWidths["hostname"], "HOSTNAME",
		colWidths["type"], "TYPE",
		colWidths["provider"], "PROVIDER",
		colWidths["region"], "REGION",
		colWidths["status"], "STATUS",
		colWidths["env"], "ENVIRONMENT",
		colWidths["ip"], "IP ADDRESS",
	)
	printTableSeparator(colWidths)

	// Print rows
	for _, r := range resources {
		hostname := truncate(r.Hostname, colWidths["hostname"])
		rtype := truncate(r.Type, colWidths["type"])
		provider := truncate(r.Provider, colWidths["provider"])
		region := ""
		if r.Region.Valid {
			region = truncate(r.Region.String, colWidths["region"])
		}
		status := truncate(r.Status, colWidths["status"])
		env := truncate(r.Environment, colWidths["env"])
		ip := ""
		if r.Metadata != nil {
			if ipVal, ok := r.Metadata["ip"].(string); ok {
				ip = truncate(ipVal, colWidths["ip"])
			}
		}

		fmt.Printf("| %-*s | %-*s | %-*s | %-*s | %s%-*s%s | %s%-*s%s | %-*s |\n",
			colWidths["hostname"], hostname,
			colWidths["type"], rtype,
			colWidths["provider"], provider,
			colWidths["region"], region,
			getStatusColor(r.Status), colWidths["status"], status, colorReset,
			getEnvColor(r.Environment), colWidths["env"], env, colorReset,
			colWidths["ip"], ip,
		)
	}

	printTableSeparator(colWidths)
}

// printTableSeparator prints a table separator line
func printTableSeparator(colWidths map[string]int) {
	totalWidth := colWidths["hostname"] + colWidths["type"] + colWidths["provider"] +
		colWidths["region"] + colWidths["status"] + colWidths["env"] + colWidths["ip"] + 21
	fmt.Println(strings.Repeat("-", totalWidth))
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// getStatusColor returns the color for a status
func getStatusColor(status string) string {
	switch status {
	case "active":
		return colorGreen
	case "inactive":
		return colorYellow
	case "build":
		return colorCyan
	case "blackout":
		return colorPurple
	case "maintenance":
		return colorBlue
	case "decommissioned":
		return colorRed
	default:
		return colorWhite
	}
}

// getEnvColor returns the color for an environment
func getEnvColor(env string) string {
	switch env {
	case "production":
		return colorRed
	case "staging":
		return colorYellow
	case "development":
		return colorGreen
	default:
		return colorWhite
	}
}

// printResourceJSON prints a resource as JSON
func printResourceJSON(r *Resource) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

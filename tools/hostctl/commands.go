package main

import (
	"fmt"
	"strings"
)

// runAdd executes the add command
func runAdd(opts *AddOptions) error {
	// Validate required fields
	if opts.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if opts.IP == "" {
		return fmt.Errorf("IP address is required")
	}

	// Validate type
	validTypes := []string{"server", "container", "vm", "k8s-node", "load-balancer", "database"}
	if !contains(validTypes, opts.Type) {
		return fmt.Errorf("invalid type: %s (must be one of: %s)", opts.Type, strings.Join(validTypes, ", "))
	}

	// Validate provider
	validProviders := []string{"oci", "gcp", "onprem", "other"}
	if !contains(validProviders, opts.Provider) {
		return fmt.Errorf("invalid provider: %s (must be one of: %s)", opts.Provider, strings.Join(validProviders, ", "))
	}

	// Validate environment
	validEnvironments := []string{"production", "staging", "development"}
	if !contains(validEnvironments, opts.Environment) {
		return fmt.Errorf("invalid environment: %s (must be one of: %s)", opts.Environment, strings.Join(validEnvironments, ", "))
	}

	// Validate status
	validStatuses := []string{"active", "inactive", "build", "blackout", "maintenance", "decommissioned"}
	if !contains(validStatuses, opts.Status) {
		return fmt.Errorf("invalid status: %s (must be one of: %s)", opts.Status, strings.Join(validStatuses, ", "))
	}

	// Insert the resource
	resource, err := insertResource(opts)
	if err != nil {
		printError(err.Error())
		return err
	}

	if jsonOutput {
		return printJSON(resource)
	}

	printSuccess(fmt.Sprintf("Successfully added host: %s", opts.Hostname))
	fmt.Println()
	printResource(resource)

	return nil
}

// runRemove executes the remove command
func runRemove(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}

	// Check if host exists first
	_, err := getResourceByHostname(hostname)
	if err != nil {
		printError(err.Error())
		return err
	}

	// Delete the resource
	if err := deleteResource(hostname); err != nil {
		printError(err.Error())
		return err
	}

	printSuccess(fmt.Sprintf("Successfully removed host: %s", hostname))
	return nil
}

// runUpdate executes the update command
func runUpdate(opts *UpdateOptions) error {
	if opts.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}

	// Validate type if provided
	if opts.Type != "" {
		validTypes := []string{"server", "container", "vm", "k8s-node", "load-balancer", "database"}
		if !contains(validTypes, opts.Type) {
			return fmt.Errorf("invalid type: %s", opts.Type)
		}
	}

	// Validate provider if provided
	if opts.Provider != "" {
		validProviders := []string{"oci", "gcp", "onprem", "other"}
		if !contains(validProviders, opts.Provider) {
			return fmt.Errorf("invalid provider: %s", opts.Provider)
		}
	}

	// Validate environment if provided
	if opts.Environment != "" {
		validEnvironments := []string{"production", "staging", "development"}
		if !contains(validEnvironments, opts.Environment) {
			return fmt.Errorf("invalid environment: %s", opts.Environment)
		}
	}

	// Validate status if provided
	if opts.Status != "" {
		validStatuses := []string{"active", "inactive", "build", "blackout", "maintenance", "decommissioned"}
		if !contains(validStatuses, opts.Status) {
			return fmt.Errorf("invalid status: %s", opts.Status)
		}
	}

	// Update the resource
	resource, err := updateResource(opts)
	if err != nil {
		printError(err.Error())
		return err
	}

	if jsonOutput {
		return printJSON(resource)
	}

	printSuccess(fmt.Sprintf("Successfully updated host: %s", opts.Hostname))
	fmt.Println()
	printResource(resource)

	return nil
}

// runStatus executes the status command
func runStatus(hostname, newStatus string) error {
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if newStatus == "" {
		return fmt.Errorf("status is required")
	}

	// Validate status
	validStatuses := []string{"active", "inactive", "build", "blackout", "maintenance", "decommissioned"}
	if !contains(validStatuses, newStatus) {
		return fmt.Errorf("invalid status: %s (must be one of: %s)", newStatus, strings.Join(validStatuses, ", "))
	}

	// Update the status
	resource, err := updateResourceStatus(hostname, newStatus)
	if err != nil {
		printError(err.Error())
		return err
	}

	if jsonOutput {
		return printJSON(resource)
	}

	printSuccess(fmt.Sprintf("Successfully updated status for %s to: %s", hostname, newStatus))
	fmt.Println()
	printResource(resource)

	return nil
}

// runList executes the list command
func runList(opts *ListOptions) error {
	// Validate filters
	if opts.Status != "" {
		validStatuses := []string{"active", "inactive", "build", "blackout", "maintenance", "decommissioned"}
		if !contains(validStatuses, opts.Status) {
			return fmt.Errorf("invalid status: %s", opts.Status)
		}
	}

	if opts.Environment != "" {
		validEnvironments := []string{"production", "staging", "development"}
		if !contains(validEnvironments, opts.Environment) {
			return fmt.Errorf("invalid environment: %s", opts.Environment)
		}
	}

	if opts.Type != "" {
		validTypes := []string{"server", "container", "vm", "k8s-node", "load-balancer", "database"}
		if !contains(validTypes, opts.Type) {
			return fmt.Errorf("invalid type: %s", opts.Type)
		}
	}

	// List resources
	resources, err := listResources(opts)
	if err != nil {
		printError(err.Error())
		return err
	}

	if jsonOutput {
		return printJSON(resources)
	}

	if len(resources) == 0 {
		fmt.Println("No hosts found matching the criteria.")
		return nil
	}

	printResourceTable(resources)
	fmt.Printf("\nTotal: %d host(s)\n", len(resources))

	return nil
}

// runShow executes the show command
func runShow(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}

	resource, err := getResourceByHostname(hostname)
	if err != nil {
		printError(err.Error())
		return err
	}

	if jsonOutput {
		return printJSON(resource)
	}

	printResourceDetailed(resource)
	return nil
}

// runSearch executes the search command
func runSearch(query string) error {
	if query == "" {
		return fmt.Errorf("search query is required")
	}

	resources, err := searchResources(query)
	if err != nil {
		printError(err.Error())
		return err
	}

	if jsonOutput {
		return printJSON(resources)
	}

	if len(resources) == 0 {
		fmt.Printf("No hosts found matching: %s\n", query)
		return nil
	}

	printResourceTable(resources)
	fmt.Printf("\nTotal: %d host(s) found\n", len(resources))

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

#!/bin/bash
# hostctl Examples - Comprehensive demonstration of all features

set -e

echo "=========================================="
echo "hostctl CLI Tool - Example Commands"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print section headers
print_section() {
    echo ""
    echo -e "${BLUE}=========================================="
    echo -e "$1"
    echo -e "==========================================${NC}"
    echo ""
}

# Function to print commands before executing
print_command() {
    echo -e "${GREEN}$ $1${NC}"
}

print_section "1. Adding Hosts"

echo "Adding a web server to production:"
print_command "hostctl add web-prod-01 --ip 10.0.1.10 --type server --env production --provider oci --region us-phoenix-1 --status active --owners ops@example.com --cost-monthly 450.00 --tags '{\"role\":\"webserver\",\"app\":\"nginx\"}'"

echo ""
echo "Adding a Kubernetes node:"
print_command "hostctl add k8s-worker-01 --ip 10.0.2.20 --type k8s-node --env production --provider gcp --region us-west1 --status active --shape n1-standard-4 --owners platform@example.com --cost-daily 8.50"

echo ""
echo "Adding a database server:"
print_command "hostctl add postgres-prod-01 --ip 10.0.3.30 --type database --env production --provider oci --region us-ashburn-1 --status active --size VM.Standard2.4 --owners dba@example.com,ops@example.com --cost-monthly 600.00 --tags '{\"db\":\"postgresql\",\"version\":\"15.2\"}'"

echo ""
echo "Adding a staging server:"
print_command "hostctl add api-staging-01 --ip 10.0.4.40 --type server --env staging --provider onprem --status active --owners dev@example.com"

print_section "2. Listing Hosts"

echo "List all hosts:"
print_command "hostctl list"

echo ""
echo "List production hosts only:"
print_command "hostctl list --env production"

echo ""
echo "List active servers:"
print_command "hostctl list --status active --type server"

echo ""
echo "List OCI hosts:"
print_command "hostctl list --provider oci"

echo ""
echo "List hosts with limit:"
print_command "hostctl list --limit 10"

echo ""
echo "List hosts in JSON format:"
print_command "hostctl list --json"

print_section "3. Showing Host Details"

echo "Show detailed information about a host:"
print_command "hostctl show web-prod-01"

echo ""
echo "Show host details in JSON:"
print_command "hostctl show web-prod-01 --json"

print_section "4. Updating Hosts"

echo "Update host metadata:"
print_command "hostctl update web-prod-01 --tags '{\"role\":\"webserver\",\"app\":\"nginx\",\"version\":\"1.24.0\"}'"

echo ""
echo "Update host cost:"
print_command "hostctl update web-prod-01 --cost-monthly 475.00"

echo ""
echo "Update multiple fields:"
print_command "hostctl update api-staging-01 --status active --owners dev@example.com,ops@example.com --cost-daily 5.00"

echo ""
echo "Update host IP in metadata:"
print_command "hostctl update web-prod-01 --ip 10.0.1.11"

print_section "5. Status Management"

echo "Put host into maintenance:"
print_command "hostctl status web-prod-01 maintenance"

echo ""
echo "Bring host back to active:"
print_command "hostctl status web-prod-01 active"

echo ""
echo "Mark host as inactive:"
print_command "hostctl status api-staging-01 inactive"

echo ""
echo "Decommission a host:"
print_command "hostctl status old-server-01 decommissioned"

print_section "6. Searching Hosts"

echo "Search by hostname:"
print_command "hostctl search web"

echo ""
echo "Search by IP address:"
print_command "hostctl search 10.0.1"

echo ""
echo "Search by tags:"
print_command "hostctl search nginx"

echo ""
echo "Search by external ID:"
print_command "hostctl search ocid1.instance"

print_section "7. Removing Hosts"

echo "Remove a host:"
print_command "hostctl remove old-server-01"

echo ""
echo "WARNING: This permanently deletes the host from the database!"

print_section "8. Advanced Usage Examples"

echo "Filter production servers in Phoenix:"
print_command "hostctl list --env production --type server --region us-phoenix-1"

echo ""
echo "Find all databases in production:"
print_command "hostctl list --type database --env production --json"

echo ""
echo "Find hosts in maintenance:"
print_command "hostctl list --status maintenance"

echo ""
echo "Export all hosts to JSON file:"
print_command "hostctl list --json > all_hosts.json"

echo ""
echo "Find specific host and show details:"
print_command "hostctl search postgres | head -1 | awk '{print \$1}' | xargs hostctl show"

print_section "9. Batch Operations"

echo "Update all staging hosts to build status:"
print_command "for host in \$(hostctl list --env staging --json | jq -r '.[].hostname'); do hostctl status \$host build; done"

echo ""
echo "Export production costs to CSV:"
print_command "hostctl list --env production --json | jq -r '.[] | [.hostname, .average_monthly_cost] | @csv'"

echo ""
echo "Find expensive hosts (>$500/month):"
print_command "hostctl list --json | jq '.[] | select(.average_monthly_cost > 500) | {hostname, cost: .average_monthly_cost}'"

print_section "10. Integration with Other Tools"

echo "Use with jq to filter JSON:"
print_command "hostctl list --json | jq '.[] | select(.status == \"active\" and .environment == \"production\")'"

echo ""
echo "Count hosts by status:"
print_command "hostctl list --json | jq 'group_by(.status) | map({status: .[0].status, count: length})'"

echo ""
echo "Get total monthly cost:"
print_command "hostctl list --json | jq '[.[].average_monthly_cost // 0] | add'"

echo ""
echo "List owners across all hosts:"
print_command "hostctl list --json | jq -r '.[].owners[]' | sort -u"

print_section "11. Monitoring and Reporting"

echo "Check for hosts without owners:"
print_command "hostctl list --json | jq '.[] | select(.owners | length == 0) | .hostname'"

echo ""
echo "Find hosts without cost data:"
print_command "hostctl list --json | jq '.[] | select(.average_monthly_cost == null) | .hostname'"

echo ""
echo "List hosts by provider:"
print_command "hostctl list --json | jq -r '.[] | .provider' | sort | uniq -c"

echo ""
echo "Generate status report:"
print_command "hostctl list --json | jq 'group_by(.status) | map({status: .[0].status, count: length, hosts: [.[].hostname]})'"

print_section "Complete!"

echo -e "${YELLOW}Note: These are example commands. Adjust hostnames, IPs, and values as needed.${NC}"
echo -e "${YELLOW}For more information, run: hostctl --help${NC}"
echo ""

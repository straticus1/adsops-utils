#!/bin/bash
# SSH Key Tracker - Track which SSH keys work with which hosts
# Part of AdsOps Utils

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_FILE="${DATA_FILE:-$HOME/.adsops/ssh-key-mappings.json}"
KEYS_DIR="${KEYS_DIR:-$HOME/.ssh}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Initialize data file
init_data_file() {
    mkdir -p "$(dirname "$DATA_FILE")"
    if [ ! -f "$DATA_FILE" ]; then
        echo '{"mappings": [], "last_updated": ""}' > "$DATA_FILE"
    fi
}

# Add a mapping
add_mapping() {
    local host="$1"
    local ip="$2"
    local user="$3"
    local key_file="$4"
    local notes="${5:-}"

    if [ -z "$host" ] || [ -z "$user" ] || [ -z "$key_file" ]; then
        echo -e "${RED}Error: host, user, and key_file are required${NC}"
        echo "Usage: $0 add <host> <ip> <user> <key_file> [notes]"
        exit 1
    fi

    # Verify key exists
    if [ ! -f "$key_file" ]; then
        echo -e "${RED}Error: Key file not found: $key_file${NC}"
        exit 1
    fi

    init_data_file

    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local key_name=$(basename "$key_file")

    # Create mapping JSON
    local mapping=$(cat <<EOF
{
  "host": "$host",
  "ip": "$ip",
  "user": "$user",
  "key_file": "$key_file",
  "key_name": "$key_name",
  "notes": "$notes",
  "added_at": "$timestamp",
  "last_verified": "$timestamp",
  "status": "active"
}
EOF
)

    # Add to file using jq
    local temp_file=$(mktemp)
    jq ".mappings += [$mapping] | .last_updated = \"$timestamp\"" "$DATA_FILE" > "$temp_file"
    mv "$temp_file" "$DATA_FILE"

    echo -e "${GREEN}✓ Added mapping: $user@$host → $key_name${NC}"
}

# Remove a mapping
remove_mapping() {
    local host="$1"
    local user="${2:-}"

    if [ -z "$host" ]; then
        echo -e "${RED}Error: host is required${NC}"
        echo "Usage: $0 remove <host> [user]"
        exit 1
    fi

    init_data_file

    local filter=".host == \"$host\""
    if [ -n "$user" ]; then
        filter="$filter and .user == \"$user\""
    fi

    local temp_file=$(mktemp)
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    jq ".mappings = [.mappings[] | select($filter | not)] | .last_updated = \"$timestamp\"" "$DATA_FILE" > "$temp_file"
    mv "$temp_file" "$DATA_FILE"

    echo -e "${GREEN}✓ Removed mapping(s) for: $user@$host${NC}"
}

# List mappings
list_mappings() {
    local filter="${1:-}"

    init_data_file

    echo -e "${BLUE}SSH Key Mappings:${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if [ -n "$filter" ]; then
        jq -r ".mappings[] | select(.host | contains(\"$filter\") or .ip | contains(\"$filter\")) |
            \"\(.host) (\(.ip)) → \(.user)@\(.key_name) - \(.status)\"" "$DATA_FILE"
    else
        jq -r '.mappings[] |
            "\(.host) (\(.ip)) → \(.user)@\(.key_name) - \(.status)"' "$DATA_FILE"
    fi
}

# Get key for host
get_key() {
    local host="$1"
    local user="${2:-}"

    if [ -z "$host" ]; then
        echo -e "${RED}Error: host is required${NC}"
        exit 1
    fi

    init_data_file

    local filter=".host == \"$host\" or .ip == \"$host\""
    if [ -n "$user" ]; then
        filter="($filter) and .user == \"$user\""
    fi

    local result=$(jq -r ".mappings[] | select($filter) | .key_file" "$DATA_FILE" | head -1)

    if [ -n "$result" ]; then
        echo "$result"
    else
        echo -e "${RED}No key found for: $user@$host${NC}" >&2
        exit 1
    fi
}

# Test a mapping
test_mapping() {
    local host="$1"
    local user="${2:-}"

    if [ -z "$host" ]; then
        echo -e "${RED}Error: host is required${NC}"
        exit 1
    fi

    init_data_file

    local filter=".host == \"$host\" or .ip == \"$host\""
    if [ -n "$user" ]; then
        filter="($filter) and .user == \"$user\""
    fi

    local mappings=$(jq -c ".mappings[] | select($filter)" "$DATA_FILE")

    if [ -z "$mappings" ]; then
        echo -e "${RED}No mappings found for: $user@$host${NC}"
        exit 1
    fi

    echo -e "${BLUE}Testing SSH connections...${NC}"

    while IFS= read -r mapping; do
        local target=$(echo "$mapping" | jq -r '.ip // .host')
        local map_user=$(echo "$mapping" | jq -r '.user')
        local key_file=$(echo "$mapping" | jq -r '.key_file')
        local key_name=$(echo "$mapping" | jq -r '.key_name')

        echo -n "Testing $map_user@$target with $key_name... "

        if timeout 3 ssh -i "$key_file" -o BatchMode=yes -o ConnectTimeout=2 "$map_user@$target" "echo OK" &>/dev/null; then
            echo -e "${GREEN}✓ SUCCESS${NC}"

            # Update last_verified
            local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
            local temp_file=$(mktemp)
            jq "(.mappings[] | select(.host == \"${host}\" and .user == \"${map_user}\") | .last_verified) = \"$timestamp\"" "$DATA_FILE" > "$temp_file"
            mv "$temp_file" "$DATA_FILE"
        else
            echo -e "${RED}✗ FAILED${NC}"
        fi
    done <<< "$mappings"
}

# Scan for working keys
scan_host() {
    local host="$1"
    local ip="${2:-$host}"

    if [ -z "$host" ]; then
        echo -e "${RED}Error: host is required${NC}"
        exit 1
    fi

    echo -e "${BLUE}Scanning SSH keys for $host ($ip)...${NC}"

    local users=("ubuntu" "opc" "ec2-user" "admin" "root")
    local keys=$(find "$KEYS_DIR" -type f \( -name "*.pem" -o -name "*_rsa" -o -name "*_key" -o -name "*.key" -o -name "id_*" \) ! -name "*.pub" 2>/dev/null)

    for user in "${users[@]}"; do
        for key in $keys; do
            local key_name=$(basename "$key")
            echo -n "  Testing $user with $key_name... "

            if timeout 3 ssh -i "$key" -o BatchMode=yes -o ConnectTimeout=2 "$user@$ip" "echo OK" &>/dev/null; then
                echo -e "${GREEN}✓ WORKS!${NC}"

                # Automatically add mapping
                add_mapping "$host" "$ip" "$user" "$key" "Auto-discovered"
                return 0
            else
                echo -e "${RED}✗${NC}"
            fi
        done
    done

    echo -e "${YELLOW}No working keys found${NC}"
    return 1
}

# Export mappings
export_mappings() {
    local format="${1:-json}"

    init_data_file

    case "$format" in
        json)
            cat "$DATA_FILE"
            ;;
        csv)
            echo "host,ip,user,key_file,key_name,status,added_at,last_verified"
            jq -r '.mappings[] |
                [.host, .ip, .user, .key_file, .key_name, .status, .added_at, .last_verified] |
                @csv' "$DATA_FILE"
            ;;
        ssh-config)
            jq -r '.mappings[] |
                "Host \(.host)\n  HostName \(.ip)\n  User \(.user)\n  IdentityFile \(.key_file)\n"' "$DATA_FILE"
            ;;
        *)
            echo -e "${RED}Unknown format: $format${NC}"
            echo "Supported formats: json, csv, ssh-config"
            exit 1
            ;;
    esac
}

# Show help
show_help() {
    cat <<EOF
SSH Key Tracker - Track which SSH keys work with which hosts

Usage: $0 <command> [options]

Commands:
  add <host> <ip> <user> <key_file> [notes]
      Add a new SSH key mapping
      Example: $0 add api-server-1 132.145.179.230 opc ~/.ssh/darkapi_key "API Proxy"

  remove <host> [user]
      Remove SSH key mapping(s) for a host
      Example: $0 remove api-server-1 opc

  list [filter]
      List all SSH key mappings (optionally filtered by host/IP)
      Example: $0 list api

  get <host> [user]
      Get the SSH key file for a specific host
      Example: $0 get api-server-1 opc

  test <host> [user]
      Test SSH connection using stored mappings
      Example: $0 test api-server-1

  scan <host> [ip]
      Scan for working SSH keys (tries all keys and users)
      Example: $0 scan api-server-1 132.145.179.230

  export [format]
      Export mappings (formats: json, csv, ssh-config)
      Example: $0 export csv

  help
      Show this help message

Data file: $DATA_FILE
Keys directory: $KEYS_DIR

Environment Variables:
  DATA_FILE    Path to mappings file (default: ~/.adsops/ssh-key-mappings.json)
  KEYS_DIR     Directory containing SSH keys (default: ~/.ssh)
EOF
}

# Main command dispatcher
main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        add)
            add_mapping "$@"
            ;;
        remove|rm)
            remove_mapping "$@"
            ;;
        list|ls)
            list_mappings "$@"
            ;;
        get)
            get_key "$@"
            ;;
        test)
            test_mapping "$@"
            ;;
        scan)
            scan_host "$@"
            ;;
        export)
            export_mappings "$@"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}Unknown command: $command${NC}"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

main "$@"

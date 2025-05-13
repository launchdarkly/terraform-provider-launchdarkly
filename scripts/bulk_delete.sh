#!/bin/bash

set -e

# Function to show usage
usage() {
    echo "Usage: $0 --api-key <api_key> --resource <resource_type>"
    echo "Resource types: projects, relay-proxies, teams"
    exit 1
}

# Function to delete projects
delete_projects() {
    local api_key=$1
    local count=0
    
    echo "Fetching projects..."
    projects=$(curl -s -H "Authorization: $api_key" \
                   -H "Content-Type: application/json" \
                   "https://app.launchdarkly.com/api/v2/projects")
    
    if ! echo "$projects" | jq -e '.items' >/dev/null 2>&1; then
        echo "Failed to fetch projects"
        return 1
    fi
    
    # Extract project keys and loop through them
    echo "$projects" | jq -r '.items[].key' | while read -r project_key; do
        if [ "$project_key" = "default" ]; then
            echo "Skipping default project as it cannot be deleted"
            continue
        fi
        
        # Check if project key is exactly 10 characters and contains no dashes
        if [ ${#project_key} -eq 10 ] && [[ ! "$project_key" =~ "-" ]]; then
            echo "Deleting project: $project_key"
            response=$(curl -s -w "%{http_code}" -X DELETE \
                           -H "Authorization: $api_key" \
                           -H "Content-Type: application/json" \
                           "https://app.launchdarkly.com/api/v2/projects/$project_key")
            
            if [ "$response" = "204" ]; then
                echo "Successfully deleted project: $project_key"
                count=$((count + 1))
            else
                echo "Failed to delete project $project_key: $response"
                return 1
            fi
            
            sleep 0.5
        else
            echo "Skipping project $project_key (does not match criteria: 10 chars, no dashes)"
        fi
    done
    echo "$count" > /tmp/ld_delete_count
}

# Function to delete relay proxies
delete_relay_proxies() {
    local api_key=$1
    local count=0
    
    echo "Fetching relay proxies..."
    proxies=$(curl -s -H "Authorization: $api_key" \
                  -H "Content-Type: application/json" \
                  "https://app.launchdarkly.com/api/v2/relay-proxies")
    
    if ! echo "$proxies" | jq -e '.items' >/dev/null 2>&1; then
        echo "Failed to fetch relay proxies"
        return 1
    fi
    
    # Extract proxy IDs and loop through them
    echo "$proxies" | jq -r '.items[]._id' | while read -r proxy_id; do
        echo "Deleting relay proxy: $proxy_id"
        response=$(curl -s -w "%{http_code}" -X DELETE \
                       -H "Authorization: $api_key" \
                       -H "Content-Type: application/json" \
                       "https://app.launchdarkly.com/api/v2/relay-proxies/$proxy_id")
        
        if [ "$response" = "204" ]; then
            echo "Successfully deleted relay proxy: $proxy_id"
            count=$((count + 1))
        else
            echo "Failed to delete relay proxy $proxy_id: $response"
            return 1
        fi
        
        sleep 0.5
    done
    echo "$count" > /tmp/ld_delete_count
}

# Function to delete teams
delete_teams() {
    local api_key=$1
    local count=0
    
    echo "Fetching teams..."
    teams=$(curl -s -H "Authorization: $api_key" \
                -H "Content-Type: application/json" \
                "https://app.launchdarkly.com/api/v2/teams")
    
    if ! echo "$teams" | jq -e '.items' >/dev/null 2>&1; then
        echo "Failed to fetch teams"
        return 1
    fi
    
    # Extract team keys and loop through them
    echo "$teams" | jq -r '.items[].key' | while read -r team_key; do
        echo "Deleting team: $team_key"
        response=$(curl -s -w "%{http_code}" -X DELETE \
                       -H "Authorization: $api_key" \
                       -H "Content-Type: application/json" \
                       "https://app.launchdarkly.com/api/v2/teams/$team_key")
        
        if [ "$response" = "204" ]; then
            echo "Successfully deleted team: $team_key"
            count=$((count + 1))
        else
            echo "Failed to delete team $team_key: $response"
            return 1
        fi
        
        sleep 0.5
    done
    echo "$count" > /tmp/ld_delete_count
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --api-key)
            API_KEY="$2"
            shift 2
            ;;
        --resource)
            RESOURCE="$2"
            shift 2
            ;;
        *)
            usage
            ;;
    esac
done

# Validate required arguments
if [ -z "$API_KEY" ] || [ -z "$RESOURCE" ]; then
    usage
fi

# Ensure API key is properly formatted
if [[ ! "$API_KEY" =~ ^api- ]]; then
    API_KEY="api-$API_KEY"
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed. Please install jq to use this script."
    exit 1
fi

# Execute the appropriate deletion function
case "$RESOURCE" in
    projects)
        delete_projects "$API_KEY"
        count=$(cat /tmp/ld_delete_count)
        echo -e "\nSummary: Deleted $count project(s)"
        ;;
    relay-proxies)
        delete_relay_proxies "$API_KEY"
        count=$(cat /tmp/ld_delete_count)
        echo -e "\nSummary: Deleted $count relay proxy/proxies"
        ;;
    teams)
        delete_teams "$API_KEY"
        count=$(cat /tmp/ld_delete_count)
        echo -e "\nSummary: Deleted $count team(s)"
        ;;
    *)
        echo "Invalid resource type: $RESOURCE"
        usage
        ;;
esac

# Cleanup
rm -f /tmp/ld_delete_count 
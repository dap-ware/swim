#!/bin/bash

# Default values
REQUESTS=200
INTERVAL=300 # Default interval in milliseconds
URL="http://localhost:8080/v1/cert-updates?page=1&size=100"

# Function to display usage
usage() {
    echo "Usage: $0 -reqs number_of_requests -int interval_in_milliseconds -url endpoint_url"
    exit 1
}

# Cleanup function to be called when the script exits or is interrupted
cleanup() {
    echo "Exiting script. Cleanup if required."
    exit 0
}

# Trap signals and call cleanup function
trap cleanup SIGINT SIGTERM

# Parse command line arguments
while getopts ":reqs:int:url:" flag; do
    case "${flag}" in
        reqs) REQUESTS=${OPTARG};;
        int) INTERVAL=${OPTARG};;
        url) URL=${OPTARG};;
        \?) usage;; # Handle unknown options
        :) echo "Missing option argument for -$OPTARG" >&2; exit 1;;
    esac
done

# Convert interval to seconds for sleep command
INTERVAL_SEC=$(echo "scale=3; $INTERVAL/1000" | bc)

echo "Starting rate limit test:"
echo "URL: $URL"
echo "Requests: $REQUESTS"
echo "Interval: $INTERVAL milliseconds ($INTERVAL_SEC seconds)"

for (( i=1; i<=REQUESTS; i++ ))
do
    echo "Request $i of $REQUESTS"
    curl -s "$URL" | jq
    sleep $INTERVAL_SEC
done

echo "Rate limit test completed."

cleanup

#!/bin/bash

# Test script to demonstrate the git-repo-downloader functionality
# This example uses the Kubernetes organization which has public repositories

echo "Testing git-repo-downloader with Kubernetes organization (public repos only)..."
echo "This will download repositories to a test directory to avoid cluttering ~/dev"

# Create a test directory
TEST_DIR="./test-downloads"
mkdir -p "$TEST_DIR"

echo "Running: ./git-repo-downloader -platform=github -org=kubernetes -dir=$TEST_DIR"
echo "Note: This will only download the first few repositories as a test..."

# You can uncomment the line below to actually run the test
# ./git-repo-downloader -platform=github -org=kubernetes -dir="$TEST_DIR"

echo ""
echo "To test with your own organization, run:"
echo "./git-repo-downloader -platform=github -org=your-org -token=your-token"
echo ""
echo "To test with GitLab, run:"
echo "./git-repo-downloader -platform=gitlab -org=your-group -token=your-token" 
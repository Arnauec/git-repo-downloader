# Pentest Scheduler

A Go application that analyzes git repositories to prioritize penetration testing efforts based on code change activity over configurable time periods.

## Overview

The Pentest Scheduler helps security teams prioritize which applications need penetration testing by analyzing git repository activity. It calculates a risk score based on:

- **Code change percentage** - How much of the codebase has changed
- **Commit frequency** - Number of commits in the time period
- **Recency of changes** - How recently changes were made
- **Files modified** - Number of files that have been updated

## Features

- 🔍 **Automatic Repository Discovery** - Recursively finds all git repositories in a directory
- 📊 **Risk-Based Prioritization** - Calculates risk scores and assigns priority levels (HIGH/MEDIUM/LOW)
- ⏰ **Configurable Time Periods** - Analyze changes over 1 month, 6 months, 1 year, or custom periods
- 📈 **Multiple Output Formats** - Table, JSON, and CSV output formats
- 🎯 **Filtering Options** - Filter by minimum change threshold and include/exclude inactive repos
- 📁 **Flexible Directory Structure** - Works with any directory containing git repositories
- 🔄 **Scheduler Ready** - Designed to run periodically via cron or CI/CD

## Installation

1. Clone or download this repository:
```bash
cd ~/dev
git clone <pentest-scheduler-repo-url>
cd pentest-scheduler
```

2. Build the application:
```bash
go build -o pentest-scheduler
```

Or install globally:
```bash
go install
```

## Usage

### Basic Usage

```bash
# Analyze repositories in current directory for the last 6 months
./pentest-scheduler

# Analyze specific directory for the last year
./pentest-scheduler -dir=~/dev -period=1y

# Get JSON output for automation
./pentest-scheduler -dir=~/dev -period=6m -format=json -output=results.json

# Filter out repositories with less than 5% change
./pentest-scheduler -dir=~/dev -period=6m -min-change=5.0

# Include detailed logging
./pentest-scheduler -dir=~/dev -period=6m -verbose
```

### Command Line Options

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `-dir` | Directory containing git repositories | `.` (current) | `-dir=~/dev` |
| `-period` | Time period to analyze | `6m` | `-period=1y` |
| `-format` | Output format: table, json, csv | `table` | `-format=json` |
| `-output` | Output file path | stdout | `-output=results.json` |
| `-min-change` | Minimum change percentage to include | `0.0` | `-min-change=5.0` |
| `-include-inactive` | Include repositories with no changes | `false` | `-include-inactive` |
| `-verbose` | Enable verbose logging | `false` | `-verbose` |

### Time Period Formats

| Format | Description |
|--------|-------------|
| `1m`, `1month` | 1 month (30 days) |
| `3m`, `3months` | 3 months (90 days) |
| `6m`, `6months` | 6 months (180 days) |
| `1y`, `1year` | 1 year (365 days) |
| `2y`, `2years` | 2 years (730 days) |
| `720h`, `168h` | Custom duration (Go duration format) |

## Output Examples

### Table Format (Default)

```
Pentest Priority Analysis Results
=================================

REPOSITORY               PRIORITY   RISK     COMMITS      CHANGES%     LAST COMMIT      FILES   
----------------------------------------------------------------------------------------------------
web-app-api             🔴 HIGH    87.5     45           12.34%       2024-05-30       23      
mobile-app              🟡 MED     56.2     23           8.45%        2024-05-28       15      
legacy-system           🟢 LOW     23.1     5            2.10%        2024-04-15       8       

Priority Summary:
-----------------
🔴 HIGH priority:   1 repositories (immediate pentesting recommended)
🟡 MEDIUM priority: 1 repositories (pentest within 3 months)
🟢 LOW priority:    1 repositories (pentest within 6 months)

Recommendations:
----------------
• Start with HIGH priority repositories - these have significant recent changes
• Schedule MEDIUM priority repositories for upcoming pentest cycles
• LOW priority repositories can be tested during maintenance cycles
```

### JSON Format

```json
{
  "generated_at": "2024-05-31T10:30:00Z",
  "time_period": "6m0s",
  "repositories_dir": "/Users/user/dev",
  "total_repositories": 3,
  "active_repositories": 3,
  "results": [
    {
      "name": "web-app-api",
      "path": "/Users/user/dev/web-app-api",
      "last_commit_date": "2024-05-30T15:20:00Z",
      "commit_count": 45,
      "files_changed": 23,
      "lines_added": 1250,
      "lines_deleted": 890,
      "lines_modified": 2140,
      "total_changes": 2140,
      "change_percentage": 12.34,
      "risk_score": 87.5,
      "recommended_priority": "HIGH"
    }
  ]
}
```

## Risk Scoring Algorithm

The risk score is calculated using three factors:

1. **Change Percentage (0-40 points)**
   - Based on lines modified vs. total repository size
   - Higher percentage = higher risk

2. **Commit Frequency (0-30 points)**
   - Number of commits in the time period
   - More commits = more changes = higher risk

3. **Recency (0-30 points)**
   - How recently the last commit was made
   - Recent changes = higher risk
   - < 7 days: 30 points
   - < 30 days: 20 points  
   - < 90 days: 10 points

**Priority Levels:**
- **HIGH (70-100 points)**: Immediate pentesting recommended
- **MEDIUM (40-69 points)**: Pentest within 3 months
- **LOW (0-39 points)**: Pentest within 6 months

## Automation & Scheduling

### Running Periodically with Cron

Add to crontab for weekly analysis:

```bash
# Run every Monday at 9 AM
0 9 * * 1 /path/to/pentest-scheduler -dir=/path/to/repos -format=json -output=/var/reports/pentest-priority.json
```

### CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Pentest Priority Analysis
on:
  schedule:
    - cron: '0 9 * * 1'  # Weekly on Monday
  workflow_dispatch:

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Build pentest-scheduler
        run: go build -o pentest-scheduler
      
      - name: Analyze repositories
        run: |
          ./pentest-scheduler -dir=. -period=6m -format=json -output=pentest-analysis.json
          
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: pentest-analysis
          path: pentest-analysis.json
```

### Docker Usage

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o pentest-scheduler

FROM alpine:latest
RUN apk add --no-cache git
WORKDIR /app
COPY --from=builder /app/pentest-scheduler .
ENTRYPOINT ["./pentest-scheduler"]
```

```bash
# Build and run
docker build -t pentest-scheduler .
docker run -v /path/to/repos:/repos pentest-scheduler -dir=/repos -period=6m
```

## Integration with Security Tools

### Integrate with JIRA

```bash
# Generate JSON report and create JIRA tickets for HIGH priority repos
./pentest-scheduler -dir=~/dev -period=6m -format=json | \
  jq '.results[] | select(.recommended_priority == "HIGH")' | \
  while read repo; do
    # Create JIRA ticket logic here
  done
```

### Integrate with Slack

```bash
# Send summary to Slack
SUMMARY=$(./pentest-scheduler -dir=~/dev -period=6m 2>/dev/null | grep -A 3 "Priority Summary")
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"Weekly Pentest Priority Report:\n'"$SUMMARY"'"}' \
  $SLACK_WEBHOOK_URL
```

## Best Practices

1. **Regular Analysis**: Run weekly or bi-weekly for up-to-date priorities
2. **Baseline Establishment**: Run initial analysis to establish baseline risk scores
3. **Custom Thresholds**: Adjust `-min-change` based on your organization's risk tolerance
4. **Historical Tracking**: Keep historical JSON outputs to track trends over time
5. **Team Integration**: Share results with development and security teams

## Troubleshooting

### Common Issues

1. **Permission denied errors**: Ensure the application has read access to all repositories
2. **Git command not found**: Ensure git is installed and in PATH
3. **Empty results**: Check that repositories contain commits in the specified time period
4. **High memory usage**: For large repositories, consider filtering or running on smaller subsets

### Debug Mode

Run with `-verbose` to see detailed processing information:

```bash
./pentest-scheduler -dir=~/dev -period=6m -verbose
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details. 
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Config struct {
	RepositoriesDir    string        // Directory containing git repositories
	TimePeriod         time.Duration // Time period to analyze (6 months, 1 year, etc.)
	OutputFormat       string        // Output format: json, csv, table
	OutputFile         string        // Output file path (optional)
	MinChangeThreshold float64       // Minimum change percentage to include in results
	IncludeInactive    bool          // Include repositories with no changes
	Verbose            bool          // Verbose logging
}

type RepositoryAnalysis struct {
	Name                string    `json:"name"`
	Path                string    `json:"path"`
	LastCommitDate      time.Time `json:"last_commit_date"`
	CommitCount         int       `json:"commit_count"`
	FilesChanged        int       `json:"files_changed"`
	LinesAdded          int       `json:"lines_added"`
	LinesDeleted        int       `json:"lines_deleted"`
	LinesModified       int       `json:"lines_modified"`
	TotalChanges        int       `json:"total_changes"`
	ChangePercentage    float64   `json:"change_percentage"`
	RiskScore           float64   `json:"risk_score"`
	RecommendedPriority string    `json:"recommended_priority"`
	Error               string    `json:"error,omitempty"`
}

type AnalysisResults struct {
	GeneratedAt     time.Time             `json:"generated_at"`
	TimePeriod      string                `json:"time_period"`
	RepositoriesDir string                `json:"repositories_dir"`
	TotalRepos      int                   `json:"total_repositories"`
	ActiveRepos     int                   `json:"active_repositories"`
	Results         []*RepositoryAnalysis `json:"results"`
}

func main() {
	var config Config
	var timePeriodStr string

	// Parse command line flags
	flag.StringVar(&config.RepositoriesDir, "dir", ".", "Directory containing git repositories")
	flag.StringVar(&timePeriodStr, "period", "6m", "Time period to analyze (e.g., 6m, 1y, 3m)")
	flag.StringVar(&config.OutputFormat, "format", "table", "Output format: table, json, csv")
	flag.StringVar(&config.OutputFile, "output", "", "Output file path (optional, defaults to stdout)")
	flag.Float64Var(&config.MinChangeThreshold, "min-change", 0.0, "Minimum change percentage to include")
	flag.BoolVar(&config.IncludeInactive, "include-inactive", false, "Include repositories with no changes")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")

	flag.Parse()

	// Show help if no arguments
	if len(os.Args) == 1 {
		fmt.Println("Pentest Scheduler - Repository Change Analysis")
		fmt.Println("==============================================")
		fmt.Println()
		fmt.Println("Usage:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  pentest-scheduler -dir=~/dev -period=6m -format=table")
		fmt.Println("  pentest-scheduler -dir=/path/to/repos -period=1y -format=json -output=results.json")
		fmt.Println("  pentest-scheduler -dir=. -period=3m -min-change=5.0 -verbose")
		fmt.Println()
		fmt.Println("Time period formats:")
		fmt.Println("  1m, 1month = 1 month")
		fmt.Println("  3m, 3months = 3 months")
		fmt.Println("  6m, 6months = 6 months")
		fmt.Println("  1y, 1year = 1 year")
		fmt.Println("  2y, 2years = 2 years")
		fmt.Println("  Custom: 720h, 168h, etc.")
		os.Exit(0)
	}

	// Parse time period
	var err error
	config.TimePeriod, err = parseTimePeriod(timePeriodStr)
	if err != nil {
		log.Fatalf("Invalid time period '%s': %v", timePeriodStr, err)
	}

	// Expand ~ in directory path
	if config.RepositoriesDir[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Error getting home directory: %v", err)
		}
		config.RepositoriesDir = filepath.Join(homeDir, config.RepositoriesDir[2:])
	}

	// Validate input directory
	if _, err := os.Stat(config.RepositoriesDir); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", config.RepositoriesDir)
	}

	fmt.Printf("Pentest Scheduler - Repository Change Analysis\n")
	fmt.Printf("==============================================\n")
	fmt.Printf("Analyzing repositories in: %s\n", config.RepositoriesDir)
	fmt.Printf("Time period: %s\n", timePeriodStr)
	fmt.Printf("Output format: %s\n", config.OutputFormat)
	if config.OutputFile != "" {
		fmt.Printf("Output file: %s\n", config.OutputFile)
	}
	fmt.Println()

	// Run analysis
	results, err := analyzeRepositories(config)
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// Output results
	if err := outputResults(results, config); err != nil {
		log.Fatalf("Failed to output results: %v", err)
	}

	// Print summary
	fmt.Printf("\nAnalysis Summary:\n")
	fmt.Printf("- Total repositories: %d\n", results.TotalRepos)
	fmt.Printf("- Active repositories: %d\n", results.ActiveRepos)
	fmt.Printf("- High priority: %d\n", countByPriority(results.Results, "HIGH"))
	fmt.Printf("- Medium priority: %d\n", countByPriority(results.Results, "MEDIUM"))
	fmt.Printf("- Low priority: %d\n", countByPriority(results.Results, "LOW"))
}

func parseTimePeriod(period string) (time.Duration, error) {
	switch period {
	case "1m", "1month":
		return 30 * 24 * time.Hour, nil
	case "3m", "3months":
		return 90 * 24 * time.Hour, nil
	case "6m", "6months":
		return 180 * 24 * time.Hour, nil
	case "1y", "1year":
		return 365 * 24 * time.Hour, nil
	case "2y", "2years":
		return 730 * 24 * time.Hour, nil
	default:
		return time.ParseDuration(period)
	}
}

func countByPriority(results []*RepositoryAnalysis, priority string) int {
	count := 0
	for _, r := range results {
		if r.RecommendedPriority == priority {
			count++
		}
	}
	return count
} 
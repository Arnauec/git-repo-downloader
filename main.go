package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Platform     string // Platform: github or gitlab
	Organization string // Organization (GitHub) or Group (GitLab) name
	Token        string // Personal access token for authentication
	TargetDir    string // Target directory for downloaded repositories
	UseSSH       bool   // Use SSH URLs instead of HTTPS
	GitLabURL    string // GitLab instance URL (for self-hosted)
	ProdMode     bool   // Enable production mode to only download repos with lifecycle: production
	AllGroups    bool   // Download from all groups (GitLab only)
}

type CatalogInfo struct {
	RepoName     string
	RepoPath     string
	CatalogPath  string
	HasCatalog   bool
}

// CatalogYAML represents the structure of .catalog.yml files
type CatalogYAML struct {
	Version   string `yaml:"version"`
	Type      string `yaml:"type"`
	Component struct {
		Name        string   `yaml:"name"`
		Service     string   `yaml:"service"`
		Team        string   `yaml:"team"`
		Description string   `yaml:"description"`
		Tags        []string `yaml:"tags"`
		Lifecycle   string   `yaml:"lifecycle"`
		Kafka       struct {
			Consumer struct {
				Groups []string `yaml:"groups"`
				Topics []string `yaml:"topics"`
			} `yaml:"consumer"`
			Producer struct {
				Topics []string `yaml:"topics"`
			} `yaml:"producer"`
		} `yaml:"kafka"`
	} `yaml:"component"`
}

func main() {
	var config Config

	// Parse command line flags
	flag.StringVar(&config.Platform, "platform", "", "Platform to use: github or gitlab (required)")
	flag.StringVar(&config.Organization, "org", "", "Organization (GitHub) or Group (GitLab) name (required)")
	flag.StringVar(&config.Token, "token", "", "Personal access token for authentication")
	flag.StringVar(&config.TargetDir, "dir", "./repositories", "Target directory for downloaded repositories")
	flag.BoolVar(&config.UseSSH, "ssh", false, "Use SSH URLs instead of HTTPS")
	flag.StringVar(&config.GitLabURL, "gitlab-url", "https://gitlab.com", "GitLab instance URL (for self-hosted)")
	flag.BoolVar(&config.ProdMode, "prod", false, "Enable production mode to only download repositories with component.lifecycle: production")
	flag.BoolVar(&config.AllGroups, "all-groups", false, "Download from all groups (GitLab only)")

	flag.Parse()

	// Show help if no arguments or missing required flags
	if len(os.Args) == 1 || config.Platform == "" || (config.Organization == "" && !config.AllGroups) {
		fmt.Println("Git Repository Downloader")
		fmt.Println("=========================")
		fmt.Println()
		fmt.Println("Downloads all repositories from GitHub organizations or GitLab groups.")
		fmt.Println()
		fmt.Println("Usage:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Download public repositories from GitHub")
		fmt.Println("  git-repo-downloader -platform=github -org=kubernetes")
		fmt.Println()
		fmt.Println("  # Download all repositories with authentication")
		fmt.Println("  git-repo-downloader -platform=github -org=mycompany -token=ghp_xxxx")
		fmt.Println()
		fmt.Println("  # Download from GitLab group")
		fmt.Println("  git-repo-downloader -platform=gitlab -org=mygroup -token=glpat_xxxx")
		fmt.Println()
		fmt.Println("  # Download to specific directory using SSH")
		fmt.Println("  git-repo-downloader -platform=github -org=myorg -token=ghp_xxxx -dir=~/repos -ssh")
		fmt.Println()
		fmt.Println("  # Download from self-hosted GitLab")
		fmt.Println("  git-repo-downloader -platform=gitlab -org=mygroup -token=glpat_xxxx -gitlab-url=https://gitlab.company.com")
		fmt.Println()
		fmt.Println("  # Download only production repositories (with component.lifecycle: production)")
		fmt.Println("  git-repo-downloader -platform=github -org=myorg -token=ghp_xxxx --prod")
		fmt.Println()
		fmt.Println("  # Download from ALL GitLab groups (auto-discover)")
		fmt.Println("  git-repo-downloader -platform=gitlab -token=glpat_xxxx -gitlab-url=https://gitlab.company.com --all-groups")
		fmt.Println()
		
		if config.Platform == "" {
			fmt.Println("Error: -platform flag is required")
		}
		if config.Organization == "" && !config.AllGroups {
			fmt.Println("Error: -org flag is required (or use --all-groups for GitLab)")
		}
		if config.AllGroups && config.Platform != "gitlab" {
			fmt.Println("Error: --all-groups flag only works with -platform=gitlab")
		}
		os.Exit(1)
	}

	// Validate platform
	config.Platform = strings.ToLower(config.Platform)
	if config.Platform != "github" && config.Platform != "gitlab" {
		log.Fatalf("Invalid platform '%s'. Must be 'github' or 'gitlab'", config.Platform)
	}

	// Validate all-groups flag
	if config.AllGroups && config.Platform != "gitlab" {
		log.Fatalf("--all-groups flag only works with GitLab platform")
	}

	// Expand ~ in directory path
	if strings.HasPrefix(config.TargetDir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Error getting home directory: %v", err)
		}
		config.TargetDir = filepath.Join(homeDir, config.TargetDir[2:])
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(config.TargetDir, 0755); err != nil {
		log.Fatalf("Error creating target directory '%s': %v", config.TargetDir, err)
	}

	// Print configuration
	fmt.Printf("Git Repository Downloader\n")
	fmt.Printf("=========================\n")
	fmt.Printf("Platform: %s\n", config.Platform)
	if config.AllGroups {
		fmt.Printf("Mode: Auto-discover all groups\n")
	} else {
		fmt.Printf("Organization/Group: %s\n", config.Organization)
	}
	fmt.Printf("Target directory: %s\n", config.TargetDir)
	if config.Token != "" {
		fmt.Printf("Authentication: Using provided token\n")
	} else {
		fmt.Printf("Authentication: No token (public repositories only)\n")
	}
	fmt.Printf("Clone method: %s\n", getCloneMethod(config.UseSSH))
	if config.Platform == "gitlab" {
		fmt.Printf("GitLab URL: %s\n", config.GitLabURL)
	}
	if config.ProdMode {
		fmt.Printf("Production mode: Enabled (only downloading repos with lifecycle: production)\n")
	}
	fmt.Println()

	// Download repositories based on platform
	var err error
	switch config.Platform {
	case "github":
		err = downloadGitHubRepos(config)
	case "gitlab":
		err = downloadGitLabRepos(config)
	default:
		log.Fatalf("Unsupported platform: %s", config.Platform)
	}

	if err != nil {
		log.Fatalf("Failed to download repositories: %v", err)
	}

	fmt.Printf("\n✅ Repository download completed successfully!\n")
	fmt.Printf("All repositories have been downloaded to: %s\n", config.TargetDir)

	// If production mode is enabled, show final scan results
	if config.ProdMode {
		fmt.Printf("\n🔍 Final scan of downloaded repositories...\n")
		catalogInfo, err := scanForCatalogFiles(config.TargetDir)
		if err != nil {
			log.Printf("Warning: Failed to scan for catalog files: %v", err)
		} else {
			displayCatalogResults(catalogInfo)
		}
	}
}

func getCloneMethod(useSSH bool) string {
	if useSSH {
		return "SSH"
	}
	return "HTTPS"
}

// scanForCatalogFiles scans all repositories in the target directory for .catalog.yml files
func scanForCatalogFiles(targetDir string) ([]CatalogInfo, error) {
	var catalogInfo []CatalogInfo

	// Read all entries in the target directory
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read target directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		repoName := entry.Name()
		repoPath := filepath.Join(targetDir, repoName)
		catalogPath := filepath.Join(repoPath, ".catalog.yml")

		// Check if .catalog.yml exists
		info := CatalogInfo{
			RepoName:    repoName,
			RepoPath:    repoPath,
			CatalogPath: catalogPath,
			HasCatalog:  false,
		}

		if _, err := os.Stat(catalogPath); err == nil {
			info.HasCatalog = true
		}

		catalogInfo = append(catalogInfo, info)
	}

	return catalogInfo, nil
}

// displayCatalogResults displays the results of the catalog file scan
func displayCatalogResults(catalogInfo []CatalogInfo) {
	fmt.Printf("\nCatalog File Scan Results\n")
	fmt.Printf("=========================\n")

	reposWithCatalog := 0
	reposWithoutCatalog := 0

	fmt.Printf("Repository Analysis:\n")
	fmt.Printf("--------------------\n")

	for _, info := range catalogInfo {
		if info.HasCatalog {
			fmt.Printf("✅ %s - .catalog.yml found\n", info.RepoName)
			reposWithCatalog++
		} else {
			fmt.Printf("❌ %s - .catalog.yml missing\n", info.RepoName)
			reposWithoutCatalog++
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("--------\n")
	fmt.Printf("Total repositories scanned: %d\n", len(catalogInfo))
	fmt.Printf("Repositories with .catalog.yml: %d\n", reposWithCatalog)
	fmt.Printf("Repositories missing .catalog.yml: %d\n", reposWithoutCatalog)

	if reposWithoutCatalog > 0 {
		fmt.Printf("\n⚠️  Repositories missing .catalog.yml files:\n")
		for _, info := range catalogInfo {
			if !info.HasCatalog {
				fmt.Printf("   - %s\n", info.RepoName)
			}
		}
	}

	if reposWithCatalog > 0 {
		fmt.Printf("\n📋 Repositories with .catalog.yml files:\n")
		for _, info := range catalogInfo {
			if info.HasCatalog {
				fmt.Printf("   - %s (%s)\n", info.RepoName, info.CatalogPath)
			}
		}
	}
} 
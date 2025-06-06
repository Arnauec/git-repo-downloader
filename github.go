package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

func downloadGitHubRepos(config Config) error {
	ctx := context.Background()

	// Create GitHub client
	var client *github.Client
	if config.Token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: config.Token},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
		fmt.Println("Warning: No token provided. Only public repositories will be accessible.")
	}

	// List all repositories for the organization
	fmt.Printf("Fetching repositories for GitHub organization: %s\n", config.Organization)
	
	var allRepos []*github.Repository
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, config.Organization, opt)
		if err != nil {
			return fmt.Errorf("error listing repositories: %w", err)
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	fmt.Printf("Found %d repositories\n", len(allRepos))

	// If production mode is enabled, filter repositories
	var reposToDownload []*github.Repository
	if config.ProdMode {
		fmt.Printf("🔍 Production mode enabled: Checking .catalog.yml files for lifecycle: production\n")
		reposToDownload = filterProductionRepos(ctx, client, allRepos, config.Organization)
		fmt.Printf("📋 Found %d repositories with lifecycle: production\n", len(reposToDownload))
	} else {
		reposToDownload = allRepos
	}

	if len(reposToDownload) == 0 {
		if config.ProdMode {
			fmt.Printf("⚠️  No repositories found with component.lifecycle: production\n")
		} else {
			fmt.Printf("⚠️  No repositories to download\n")
		}
		return nil
	}

	fmt.Printf("\n")

	// Download each repository
	for i, repo := range reposToDownload {
		fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(reposToDownload), repo.GetName())
		
		if err := cloneRepository(repo.GetName(), getGitHubCloneURL(repo, config.UseSSH), config.TargetDir); err != nil {
			log.Printf("Warning: Failed to clone %s: %v", repo.GetName(), err)
			continue
		}
		
		fmt.Printf("✓ Successfully cloned: %s\n\n", repo.GetName())
	}

	return nil
}

// filterProductionRepos checks each repository for .catalog.yml with lifecycle: production
func filterProductionRepos(ctx context.Context, client *github.Client, repos []*github.Repository, org string) []*github.Repository {
	var productionRepos []*github.Repository

	for i, repo := range repos {
		fmt.Printf("[%d/%d] Checking %s for .catalog.yml...", i+1, len(repos), repo.GetName())
		
		isProduction, err := checkGitHubCatalogFile(ctx, client, org, repo.GetName())
		if err != nil {
			fmt.Printf(" ❌ Error: %v\n", err)
			continue
		}

		if isProduction {
			fmt.Printf(" ✅ Production lifecycle found\n")
			productionRepos = append(productionRepos, repo)
		} else {
			fmt.Printf(" ⏭️  Not production or no .catalog.yml\n")
		}
	}

	return productionRepos
}

// checkGitHubCatalogFile fetches and parses .catalog.yml to check for lifecycle: production
func checkGitHubCatalogFile(ctx context.Context, client *github.Client, owner, repo string) (bool, error) {
	// Try to get .catalog.yml file from the repository
	fileContent, _, resp, err := client.Repositories.GetContents(ctx, owner, repo, ".catalog.yml", nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false, nil // File not found, not an error
		}
		return false, fmt.Errorf("failed to fetch .catalog.yml: %w", err)
	}

	if fileContent == nil {
		return false, nil // File not found
	}

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(*fileContent.Content)
	if err != nil {
		return false, fmt.Errorf("failed to decode file content: %w", err)
	}

	// Parse YAML
	var catalog CatalogYAML
	if err := yaml.Unmarshal(content, &catalog); err != nil {
		return false, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Check if lifecycle is production
	return catalog.Component.Lifecycle == "production", nil
}

func getGitHubCloneURL(repo *github.Repository, useSSH bool) string {
	if useSSH {
		return repo.GetSSHURL()
	}
	return repo.GetCloneURL()
}

func cloneRepository(repoName, cloneURL, targetDir string) error {
	repoPath := filepath.Join(targetDir, repoName)
	
	// Check if repository already exists
	if _, err := os.Stat(repoPath); err == nil {
		fmt.Printf("  Repository already exists at %s, skipping...\n", repoPath)
		return nil
	}

	// Clone the repository
	fmt.Printf("  Cloning from: %s\n", cloneURL)
	fmt.Printf("  Target path: %s\n", repoPath)
	
	cmd := exec.Command("git", "clone", cloneURL, repoPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	
	return nil
} 
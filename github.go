package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
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

	fmt.Printf("Found %d repositories\n\n", len(allRepos))

	// Download each repository
	for i, repo := range allRepos {
		fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(allRepos), repo.GetName())
		
		if err := cloneRepository(repo.GetName(), getGitHubCloneURL(repo, config.UseSSH), config.TargetDir); err != nil {
			log.Printf("Warning: Failed to clone %s: %v", repo.GetName(), err)
			continue
		}
		
		fmt.Printf("✓ Successfully cloned: %s\n\n", repo.GetName())
	}

	return nil
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
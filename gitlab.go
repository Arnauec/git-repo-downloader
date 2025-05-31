package main

import (
	"fmt"
	"log"
	"net/url"

	"github.com/xanzy/go-gitlab"
)

func downloadGitLabRepos(config Config) error {
	// Create GitLab client
	var client *gitlab.Client
	var err error

	if config.Token != "" {
		client, err = gitlab.NewClient(config.Token, gitlab.WithBaseURL(config.GitLabURL))
		if err != nil {
			return fmt.Errorf("error creating GitLab client: %w", err)
		}
	} else {
		// For public repositories, we can still try without authentication
		client, err = gitlab.NewClient("", gitlab.WithBaseURL(config.GitLabURL))
		if err != nil {
			return fmt.Errorf("error creating GitLab client: %w", err)
		}
		fmt.Println("Warning: No token provided. Only public repositories will be accessible.")
	}

	// Find the group/organization
	fmt.Printf("Fetching repositories for GitLab group: %s\n", config.Organization)
	
	// Search for the group
	groups, _, err := client.Groups.SearchGroup(config.Organization)
	if err != nil {
		return fmt.Errorf("error searching for group: %w", err)
	}

	if len(groups) == 0 {
		return fmt.Errorf("group '%s' not found", config.Organization)
	}

	// Find exact match or first match if no exact match
	var selectedGroup *gitlab.Group
	for _, group := range groups {
		if group.Path == config.Organization || group.Name == config.Organization {
			selectedGroup = group
			break
		}
	}

	if selectedGroup == nil {
		selectedGroup = groups[0] // Use first match if no exact match
		fmt.Printf("Using group: %s (path: %s)\n", selectedGroup.Name, selectedGroup.Path)
	}

	// List all projects in the group
	var allProjects []*gitlab.Project
	opt := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		IncludeSubGroups: gitlab.Bool(true), // Include subgroups
	}

	for {
		projects, resp, err := client.Groups.ListGroupProjects(selectedGroup.ID, opt)
		if err != nil {
			return fmt.Errorf("error listing group projects: %w", err)
		}

		allProjects = append(allProjects, projects...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	fmt.Printf("Found %d repositories\n\n", len(allProjects))

	// Download each repository
	for i, project := range allProjects {
		fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(allProjects), project.Name)
		
		cloneURL := getGitLabCloneURL(project, config.UseSSH)
		if err := cloneRepository(project.Name, cloneURL, config.TargetDir); err != nil {
			log.Printf("Warning: Failed to clone %s: %v", project.Name, err)
			continue
		}
		
		fmt.Printf("✓ Successfully cloned: %s\n\n", project.Name)
	}

	return nil
}

func getGitLabCloneURL(project *gitlab.Project, useSSH bool) string {
	if useSSH {
		return project.SSHURLToRepo
	}
	
	// For HTTPS, we return the HTTP URL which can be used with tokens
	cloneURL := project.HTTPURLToRepo
	
	// If using a token, we might want to embed it in the URL for automatic authentication
	// However, this is handled by git credential helpers in most cases
	return cloneURL
}

// Helper function to extract hostname from GitLab URL
func getGitLabHostname(gitlabURL string) string {
	parsedURL, err := url.Parse(gitlabURL)
	if err != nil {
		return "gitlab.com" // fallback
	}
	return parsedURL.Host
} 
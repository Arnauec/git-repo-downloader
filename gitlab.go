package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/url"

	"github.com/xanzy/go-gitlab"
	"gopkg.in/yaml.v3"
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

	if config.AllGroups {
		// Download from all groups
		return downloadFromAllGroups(client, config)
	} else {
		// Download from specific group
		return downloadFromSpecificGroup(client, config)
	}
}

// downloadFromAllGroups discovers all groups and downloads repositories from each
func downloadFromAllGroups(client *gitlab.Client, config Config) error {
	fmt.Printf("🔍 Discovering all groups you have access to...\n")

	// List all groups the user has access to
	var allGroups []*gitlab.Group
	opt := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		AllAvailable: gitlab.Bool(true), // Get all groups user has access to
	}

	for {
		groups, resp, err := client.Groups.ListGroups(opt)
		if err != nil {
			return fmt.Errorf("error listing groups: %w", err)
		}

		allGroups = append(allGroups, groups...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	fmt.Printf("📋 Found %d groups you have access to:\n", len(allGroups))
	for i, group := range allGroups {
		fmt.Printf("  [%d] %s (path: %s)\n", i+1, group.Name, group.Path)
	}
	fmt.Println()

	if len(allGroups) == 0 {
		fmt.Printf("⚠️  No groups found. You may need proper permissions or a valid token.\n")
		return nil
	}

	totalReposDownloaded := 0
	totalReposScanned := 0

	// Download repositories from each group
	for i, group := range allGroups {
		fmt.Printf("🗂️  [%d/%d] Processing group: %s\n", i+1, len(allGroups), group.Name)
		
		// Create a temporary config for this specific group
		groupConfig := config
		groupConfig.Organization = group.Path
		
		downloaded, scanned, err := downloadFromSpecificGroupInternal(client, groupConfig, group)
		if err != nil {
			log.Printf("Warning: Failed to process group %s: %v", group.Name, err)
			continue
		}

		totalReposDownloaded += downloaded
		totalReposScanned += scanned
		fmt.Printf("   ✓ Group %s: %d repositories downloaded\n\n", group.Name, downloaded)
	}

	fmt.Printf("🎉 All groups processed!\n")
	fmt.Printf("📊 Summary:\n")
	fmt.Printf("   - Groups processed: %d\n", len(allGroups))
	fmt.Printf("   - Total repositories scanned: %d\n", totalReposScanned)
	fmt.Printf("   - Total repositories downloaded: %d\n", totalReposDownloaded)

	return nil
}

// downloadFromSpecificGroup downloads repositories from a single specified group
func downloadFromSpecificGroup(client *gitlab.Client, config Config) error {
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

	_, _, err = downloadFromSpecificGroupInternal(client, config, selectedGroup)
	return err
}

// downloadFromSpecificGroupInternal handles the actual downloading logic for a group
func downloadFromSpecificGroupInternal(client *gitlab.Client, config Config, group *gitlab.Group) (int, int, error) {
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
		projects, resp, err := client.Groups.ListGroupProjects(group.ID, opt)
		if err != nil {
			return 0, 0, fmt.Errorf("error listing group projects: %w", err)
		}

		allProjects = append(allProjects, projects...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	if !config.AllGroups {
		fmt.Printf("Found %d repositories\n", len(allProjects))
	}

	// If production mode is enabled, filter repositories
	var projectsToDownload []*gitlab.Project
	if config.ProdMode {
		if !config.AllGroups {
			fmt.Printf("🔍 Production mode enabled: Checking .catalog.yml files for lifecycle: production\n")
		}
		projectsToDownload = filterProductionProjects(client, allProjects)
		if !config.AllGroups {
			fmt.Printf("📋 Found %d repositories with lifecycle: production\n", len(projectsToDownload))
		}
	} else {
		projectsToDownload = allProjects
	}

	if len(projectsToDownload) == 0 {
		if !config.AllGroups {
			if config.ProdMode {
				fmt.Printf("⚠️  No repositories found with component.lifecycle: production\n")
			} else {
				fmt.Printf("⚠️  No repositories to download\n")
			}
		}
		return 0, len(allProjects), nil
	}

	if !config.AllGroups {
		fmt.Printf("\n")
	}

	downloadedCount := 0

	// Download each repository
	for i, project := range projectsToDownload {
		if config.AllGroups {
			fmt.Printf("     [%d/%d] Processing: %s\n", i+1, len(projectsToDownload), project.Name)
		} else {
			fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(projectsToDownload), project.Name)
		}
		
		cloneURL := getGitLabCloneURL(project, config.UseSSH)
		if err := cloneRepository(project.Name, cloneURL, config.TargetDir); err != nil {
			log.Printf("Warning: Failed to clone %s: %v", project.Name, err)
			continue
		}
		
		if config.AllGroups {
			fmt.Printf("     ✓ Successfully cloned: %s\n", project.Name)
		} else {
			fmt.Printf("✓ Successfully cloned: %s\n\n", project.Name)
		}
		downloadedCount++
	}

	return downloadedCount, len(allProjects), nil
}

// filterProductionProjects checks each project for .catalog.yml with lifecycle: production
func filterProductionProjects(client *gitlab.Client, projects []*gitlab.Project) []*gitlab.Project {
	var productionProjects []*gitlab.Project

	for i, project := range projects {
		fmt.Printf("[%d/%d] Checking %s for .catalog.yml...", i+1, len(projects), project.Name)
		
		isProduction, err := checkGitLabCatalogFile(client, project.ID)
		if err != nil {
			fmt.Printf(" ❌ Error: %v\n", err)
			continue
		}

		if isProduction {
			fmt.Printf(" ✅ Production lifecycle found\n")
			productionProjects = append(productionProjects, project)
		} else {
			fmt.Printf(" ⏭️  Not production or no .catalog.yml\n")
		}
	}

	return productionProjects
}

// checkGitLabCatalogFile fetches and parses .catalog.yml to check for lifecycle: production
func checkGitLabCatalogFile(client *gitlab.Client, projectID int) (bool, error) {
	// Try to get .catalog.yml file from the repository
	file, resp, err := client.RepositoryFiles.GetFile(projectID, ".catalog.yml", &gitlab.GetFileOptions{
		Ref: gitlab.String("main"), // Try main branch first
	})
	if err != nil {
		// If main branch fails, try master branch
		if resp != nil && resp.StatusCode == 404 {
			file, resp, err = client.RepositoryFiles.GetFile(projectID, ".catalog.yml", &gitlab.GetFileOptions{
				Ref: gitlab.String("master"),
			})
			if err != nil {
				if resp != nil && resp.StatusCode == 404 {
					return false, nil // File not found, not an error
				}
				return false, fmt.Errorf("failed to fetch .catalog.yml: %w", err)
			}
		} else {
			return false, fmt.Errorf("failed to fetch .catalog.yml: %w", err)
		}
	}

	if file == nil {
		return false, nil // File not found
	}

	// Decode content (GitLab API returns base64 encoded content)
	content, err := base64.StdEncoding.DecodeString(file.Content)
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
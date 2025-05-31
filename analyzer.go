package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func analyzeRepositories(config Config) (*AnalysisResults, error) {
	results := &AnalysisResults{
		GeneratedAt:     time.Now(),
		TimePeriod:      config.TimePeriod.String(),
		RepositoriesDir: config.RepositoriesDir,
		Results:         make([]*RepositoryAnalysis, 0),
	}

	// Find all potential git repositories
	repositories, err := findGitRepositories(config.RepositoriesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find repositories: %w", err)
	}

	results.TotalRepos = len(repositories)

	if config.Verbose {
		fmt.Printf("Found %d potential repositories\n", len(repositories))
	}

	// Analyze each repository
	for i, repoPath := range repositories {
		if config.Verbose {
			fmt.Printf("[%d/%d] Analyzing: %s\n", i+1, len(repositories), filepath.Base(repoPath))
		}

		analysis := analyzeRepository(repoPath, config)
		
		// Apply filters
		if !config.IncludeInactive && analysis.CommitCount == 0 {
			continue
		}
		
		if analysis.ChangePercentage < config.MinChangeThreshold {
			continue
		}

		if analysis.Error == "" {
			results.ActiveRepos++
		}

		results.Results = append(results.Results, analysis)
	}

	// Sort results by risk score (descending)
	sortRepositoriesByRisk(results.Results)

	return results, nil
}

func findGitRepositories(baseDir string) ([]string, error) {
	var repositories []string

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for .git directories
		if info.IsDir() && info.Name() == ".git" {
			repoPath := filepath.Dir(path)
			repositories = append(repositories, repoPath)
			return filepath.SkipDir // Don't go deeper into .git
		}

		return nil
	})

	return repositories, err
}

func analyzeRepository(repoPath string, config Config) *RepositoryAnalysis {
	analysis := &RepositoryAnalysis{
		Name: filepath.Base(repoPath),
		Path: repoPath,
	}

	// Check if it's actually a git repository
	if !isGitRepository(repoPath) {
		analysis.Error = "Not a valid git repository"
		return analysis
	}

	// Get the cutoff date
	cutoffDate := time.Now().Add(-config.TimePeriod)

	// Analyze commits since cutoff date
	commitCount, lastCommit, err := getCommitInfo(repoPath, cutoffDate)
	if err != nil {
		analysis.Error = fmt.Sprintf("Failed to get commit info: %v", err)
		return analysis
	}

	analysis.CommitCount = commitCount
	analysis.LastCommitDate = lastCommit

	// Analyze file changes
	filesChanged, linesAdded, linesDeleted, err := getChangeStats(repoPath, cutoffDate)
	if err != nil {
		analysis.Error = fmt.Sprintf("Failed to get change stats: %v", err)
		return analysis
	}

	analysis.FilesChanged = filesChanged
	analysis.LinesAdded = linesAdded
	analysis.LinesDeleted = linesDeleted
	analysis.LinesModified = linesAdded + linesDeleted
	analysis.TotalChanges = analysis.LinesModified

	// Calculate change percentage based on repository size
	repoSize, err := getRepositorySize(repoPath)
	if err != nil {
		repoSize = 1 // Avoid division by zero
	}

	if repoSize > 0 {
		analysis.ChangePercentage = float64(analysis.TotalChanges) / float64(repoSize) * 100
	}

	// Calculate risk score and priority
	analysis.RiskScore = calculateRiskScore(analysis)
	analysis.RecommendedPriority = calculatePriority(analysis)

	return analysis
}

func isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false
	}
	return true
}

func getCommitInfo(repoPath string, since time.Time) (commitCount int, lastCommit time.Time, err error) {
	// Get commit count since date
	sinceStr := since.Format("2006-01-02")
	cmd := exec.Command("git", "rev-list", "--count", "--since="+sinceStr, "HEAD")
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return 0, time.Time{}, err
	}

	countStr := strings.TrimSpace(string(output))
	commitCount, err = strconv.Atoi(countStr)
	if err != nil {
		return 0, time.Time{}, err
	}

	// Get last commit date
	cmd = exec.Command("git", "log", "-1", "--format=%ci")
	cmd.Dir = repoPath
	
	output, err = cmd.Output()
	if err != nil {
		return commitCount, time.Time{}, err
	}

	lastCommitStr := strings.TrimSpace(string(output))
	if lastCommitStr != "" {
		lastCommit, err = time.Parse("2006-01-02 15:04:05 -0700", lastCommitStr)
		if err != nil {
			return commitCount, time.Time{}, err
		}
	}

	return commitCount, lastCommit, nil
}

func getChangeStats(repoPath string, since time.Time) (filesChanged, linesAdded, linesDeleted int, err error) {
	sinceStr := since.Format("2006-01-02")
	
	// Get diff stats
	cmd := exec.Command("git", "diff", "--numstat", "--since="+sinceStr, "HEAD~1", "HEAD")
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		// Try alternative approach for repositories with commits in the period
		cmd = exec.Command("git", "log", "--since="+sinceStr, "--numstat", "--pretty=format:")
		cmd.Dir = repoPath
		output, err = cmd.Output()
		if err != nil {
			return 0, 0, 0, err
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	fileSet := make(map[string]bool)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			// Format: added deleted filename
			if parts[0] != "-" {
				if added, err := strconv.Atoi(parts[0]); err == nil {
					linesAdded += added
				}
			}
			if parts[1] != "-" {
				if deleted, err := strconv.Atoi(parts[1]); err == nil {
					linesDeleted += deleted
				}
			}
			
			// Count unique files
			filename := parts[2]
			fileSet[filename] = true
		}
	}
	
	filesChanged = len(fileSet)
	return filesChanged, linesAdded, linesDeleted, nil
}

func getRepositorySize(repoPath string) (int, error) {
	// Get total lines of code in the repository
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return 1, err // Return 1 to avoid division by zero
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	totalLines := 0

	for _, file := range files {
		if file == "" {
			continue
		}
		
		filePath := filepath.Join(repoPath, file)
		if lines, err := countLinesInFile(filePath); err == nil {
			totalLines += lines
		}
	}

	if totalLines == 0 {
		return 1, nil // Avoid division by zero
	}
	
	return totalLines, nil
}

func countLinesInFile(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	
	return lines, scanner.Err()
}

func calculateRiskScore(analysis *RepositoryAnalysis) float64 {
	score := 0.0
	
	// Factor 1: Change percentage (0-40 points)
	score += analysis.ChangePercentage * 0.4
	
	// Factor 2: Commit frequency (0-30 points)
	if analysis.CommitCount > 0 {
		score += float64(analysis.CommitCount) * 3.0
		if score > 30 { // Cap at 30 points
			score = 30
		}
	}
	
	// Factor 3: Recency of changes (0-30 points)
	daysSinceLastCommit := time.Since(analysis.LastCommitDate).Hours() / 24
	if daysSinceLastCommit < 7 {
		score += 30
	} else if daysSinceLastCommit < 30 {
		score += 20
	} else if daysSinceLastCommit < 90 {
		score += 10
	}
	
	// Normalize to 0-100
	if score > 100 {
		score = 100
	}
	
	return score
}

func calculatePriority(analysis *RepositoryAnalysis) string {
	score := analysis.RiskScore
	
	if score >= 70 {
		return "HIGH"
	} else if score >= 40 {
		return "MEDIUM"
	} else {
		return "LOW"
	}
}

func sortRepositoriesByRisk(results []*RepositoryAnalysis) {
	// Sort by risk score (descending), then by change percentage (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].RiskScore < results[j].RiskScore ||
				(results[i].RiskScore == results[j].RiskScore && results[i].ChangePercentage < results[j].ChangePercentage) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
} 
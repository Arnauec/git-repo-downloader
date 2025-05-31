package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func outputResults(results *AnalysisResults, config Config) error {
	var output string
	var err error

	switch strings.ToLower(config.OutputFormat) {
	case "json":
		output, err = formatJSON(results)
	case "csv":
		output, err = formatCSV(results)
	case "table", "":
		output, err = formatTable(results)
	default:
		return fmt.Errorf("unsupported output format: %s", config.OutputFormat)
	}

	if err != nil {
		return err
	}

	// Write to file or stdout
	if config.OutputFile != "" {
		return writeToFile(output, config.OutputFile)
	}

	fmt.Print(output)
	return nil
}

func formatTable(results *AnalysisResults) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString("\nPentest Priority Analysis Results\n")
	sb.WriteString("=================================\n\n")

	if len(results.Results) == 0 {
		sb.WriteString("No repositories found matching the criteria.\n")
		return sb.String(), nil
	}

	// Table header
	format := "%-25s %-10s %-8s %-12s %-12s %-15s %-8s\n"
	sb.WriteString(fmt.Sprintf(format, "REPOSITORY", "PRIORITY", "RISK", "COMMITS", "CHANGES%", "LAST COMMIT", "FILES"))
	sb.WriteString(strings.Repeat("-", 100) + "\n")

	// Table rows
	for _, repo := range results.Results {
		priority := colorPriority(repo.RecommendedPriority)
		riskScore := fmt.Sprintf("%.1f", repo.RiskScore)
		changePercentage := fmt.Sprintf("%.2f%%", repo.ChangePercentage)
		
		lastCommit := "Never"
		if !repo.LastCommitDate.IsZero() {
			if time.Since(repo.LastCommitDate) < 24*time.Hour {
				lastCommit = "Today"
			} else if time.Since(repo.LastCommitDate) < 7*24*time.Hour {
				lastCommit = fmt.Sprintf("%d days ago", int(time.Since(repo.LastCommitDate).Hours()/24))
			} else {
				lastCommit = repo.LastCommitDate.Format("2006-01-02")
			}
		}

		repoName := repo.Name
		if len(repoName) > 24 {
			repoName = repoName[:21] + "..."
		}

		if repo.Error != "" {
			sb.WriteString(fmt.Sprintf(format, repoName, "ERROR", "-", "-", "-", "-", "-"))
			sb.WriteString(fmt.Sprintf("  Error: %s\n", repo.Error))
		} else {
			sb.WriteString(fmt.Sprintf(format, 
				repoName,
				priority,
				riskScore,
				strconv.Itoa(repo.CommitCount),
				changePercentage,
				lastCommit,
				strconv.Itoa(repo.FilesChanged),
			))
		}
	}

	// Summary section
	sb.WriteString("\nPriority Summary:\n")
	sb.WriteString("-----------------\n")
	
	highCount := countByPriority(results.Results, "HIGH")
	mediumCount := countByPriority(results.Results, "MEDIUM")
	lowCount := countByPriority(results.Results, "LOW")
	
	sb.WriteString(fmt.Sprintf("🔴 HIGH priority:   %d repositories (immediate pentesting recommended)\n", highCount))
	sb.WriteString(fmt.Sprintf("🟡 MEDIUM priority: %d repositories (pentest within 3 months)\n", mediumCount))
	sb.WriteString(fmt.Sprintf("🟢 LOW priority:    %d repositories (pentest within 6 months)\n", lowCount))

	// Recommendations
	sb.WriteString("\nRecommendations:\n")
	sb.WriteString("----------------\n")
	if highCount > 0 {
		sb.WriteString("• Start with HIGH priority repositories - these have significant recent changes\n")
	}
	if mediumCount > 0 {
		sb.WriteString("• Schedule MEDIUM priority repositories for upcoming pentest cycles\n")
	}
	if lowCount > 0 {
		sb.WriteString("• LOW priority repositories can be tested during maintenance cycles\n")
	}

	return sb.String(), nil
}

func formatJSON(results *AnalysisResults) (string, error) {
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func formatCSV(results *AnalysisResults) (string, error) {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	// Write header
	header := []string{
		"Repository",
		"Priority",
		"Risk Score",
		"Change Percentage",
		"Commit Count",
		"Files Changed",
		"Lines Added",
		"Lines Deleted",
		"Last Commit Date",
		"Path",
		"Error",
	}
	
	if err := writer.Write(header); err != nil {
		return "", err
	}

	// Write data rows
	for _, repo := range results.Results {
		lastCommitStr := ""
		if !repo.LastCommitDate.IsZero() {
			lastCommitStr = repo.LastCommitDate.Format("2006-01-02 15:04:05")
		}

		row := []string{
			repo.Name,
			repo.RecommendedPriority,
			fmt.Sprintf("%.2f", repo.RiskScore),
			fmt.Sprintf("%.2f", repo.ChangePercentage),
			strconv.Itoa(repo.CommitCount),
			strconv.Itoa(repo.FilesChanged),
			strconv.Itoa(repo.LinesAdded),
			strconv.Itoa(repo.LinesDeleted),
			lastCommitStr,
			repo.Path,
			repo.Error,
		}
		
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func colorPriority(priority string) string {
	switch priority {
	case "HIGH":
		return "🔴 HIGH"
	case "MEDIUM":
		return "🟡 MED"
	case "LOW":
		return "🟢 LOW"
	default:
		return priority
	}
}

func writeToFile(content, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
} 
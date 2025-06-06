# Git Repository Downloader

A Go application that downloads all repositories from GitHub and GitLab organizations or groups. This tool is useful for backing up repositories, migrating between platforms, or performing bulk analysis of organizational codebases.

## Overview

The Git Repository Downloader connects to GitHub and GitLab APIs to fetch all repositories from specified organizations or groups and clones them to a local directory. It supports both public and private repositories (with proper authentication).

**NEW**: Production Mode (`--prod`) filters repositories to only download those with `component.lifecycle: production` in their `.catalog.yml` files, making it perfect for identifying and downloading only production-ready services.

## Features

- 🐙 **GitHub Support** - Download all repositories from GitHub organizations
- 🦊 **GitLab Support** - Download all repositories from GitLab groups (including subgroups)
- 🔐 **Authentication** - Support for personal access tokens for private repositories
- 🔄 **Smart Cloning** - Skips repositories that already exist locally
- 📁 **Organized Output** - Creates clean directory structure with all repositories
- 🌐 **Multiple GitLab Instances** - Support for GitLab.com and self-hosted GitLab instances
- 🔗 **SSH/HTTPS Support** - Choose between SSH and HTTPS cloning methods
- 🏭 **Production Mode** - Filter repositories by `component.lifecycle: production` in `.catalog.yml` files

## Installation

### Prerequisites

- Go 1.21 or later
- Git installed and configured
- Network access to GitHub/GitLab

### Build from Source

1. Clone this repository:
```bash
git clone https://github.com/Arnauec/git-repo-downloader
cd git-repo-downloader
```

2. Download dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o git-repo-downloader
```

Or use the Makefile:
```bash
make build
```

### Install Globally

```bash
go install
```

## Usage

### Basic Usage

```bash
# Download all public repositories from a GitHub organization
./git-repo-downloader -platform=github -org=kubernetes

# Download all repositories from a GitLab group (requires token for private repos)
./git-repo-downloader -platform=gitlab -org=mycompany -token=your-gitlab-token

# Download to a specific directory
./git-repo-downloader -platform=github -org=kubernetes -dir=./downloads

# Production mode: Only download repos with component.lifecycle: production
./git-repo-downloader -platform=github -org=mycompany -token=your-token --prod
```

### Command Line Options

| Flag | Description | Required | Default | Example |
|------|-------------|----------|---------|---------|
| `-platform` | Platform to use: `github` or `gitlab` | Yes | - | `-platform=github` |
| `-org` | Organization (GitHub) or Group (GitLab) name | Yes | - | `-org=kubernetes` |
| `-token` | Personal access token for authentication | No* | - | `-token=ghp_xxxx` |
| `-dir` | Target directory for downloaded repositories | No | `./repositories` | `-dir=~/dev` |
| `-ssh` | Use SSH URLs instead of HTTPS | No | `false` | `-ssh` |
| `-gitlab-url` | GitLab instance URL (for self-hosted) | No | `https://gitlab.com` | `-gitlab-url=https://gitlab.example.com` |
| `--prod` | Only download repos with `component.lifecycle: production` | No | `false` | `--prod` |

*Required for private repositories

### Production Mode (--prod)

When the `--prod` flag is enabled, the tool will:

1. **Scan each repository** for a `.catalog.yml` file in the root directory
2. **Parse the YAML content** to check for `component.lifecycle: production`
3. **Only download repositories** that meet this criteria
4. **Show detailed progress** of which repositories are being checked and filtered

Example `.catalog.yml` file that would be **included** in production mode:

```yaml
---
version: '1'
type: microservice
component:
  name: 'my-production-service'
  service: 'my-production-service'
  team: Platform
  description: 'A production-ready microservice'
  tags:
    - critical
    - production
  lifecycle: production  # This line makes it eligible for --prod mode
  kafka:
    consumer:
      topics:
        - events.user.created.v1
    producer:
      topics:
        - events.notification.sent.v1
```

### Examples

#### GitHub Examples

```bash
# Download public repositories from Kubernetes organization
./git-repo-downloader -platform=github -org=kubernetes

# Download all repositories (including private) with authentication
./git-repo-downloader -platform=github -org=mycompany -token=ghp_1234567890abcdef

# Download only production services
./git-repo-downloader -platform=github -org=mycompany -token=ghp_1234567890abcdef --prod

# Use SSH for cloning (requires SSH key setup)
./git-repo-downloader -platform=github -org=mycompany -token=ghp_1234567890abcdef -ssh

# Download to specific directory
./git-repo-downloader -platform=github -org=kubernetes -dir=~/github-repos
```

#### GitLab Examples

```bash
# Download from GitLab.com group
./git-repo-downloader -platform=gitlab -org=gitlab-org -token=glpat-xxxxxxxxxxxx

# Download from self-hosted GitLab instance
./git-repo-downloader -platform=gitlab -org=mygroup -token=glpat-xxxxxxxxxxxx -gitlab-url=https://gitlab.company.com

# Download only production services from GitLab
./git-repo-downloader -platform=gitlab -org=mygroup -token=glpat-xxxxxxxxxxxx --prod

# Download including subgroups
./git-repo-downloader -platform=gitlab -org=parent-group -token=glpat-xxxxxxxxxxxx
```

## Sample Output with --prod

```
Git Repository Downloader
=========================
Platform: github
Organization/Group: mycompany
Target directory: ./repositories
Authentication: Using provided token
Clone method: HTTPS
Production mode: Enabled (only downloading repos with lifecycle: production)

Found 25 repositories
🔍 Production mode enabled: Checking .catalog.yml files for lifecycle: production
[1/25] Checking web-api for .catalog.yml... ✅ Production lifecycle found
[2/25] Checking mobile-app for .catalog.yml... ⏭️  Not production or no .catalog.yml
[3/25] Checking user-service for .catalog.yml... ✅ Production lifecycle found
[4/25] Checking test-utils for .catalog.yml... ⏭️  Not production or no .catalog.yml
...
📋 Found 8 repositories with lifecycle: production

[1/8] Processing: web-api
  Cloning from: https://github.com/mycompany/web-api.git
  Target path: ./repositories/web-api
✓ Successfully cloned: web-api

[2/8] Processing: user-service
  Cloning from: https://github.com/mycompany/user-service.git
  Target path: ./repositories/user-service
✓ Successfully cloned: user-service

...

✅ Repository download completed successfully!
All repositories have been downloaded to: ./repositories

🔍 Final scan of downloaded repositories...

Catalog File Scan Results
=========================
Repository Analysis:
--------------------
✅ web-api - .catalog.yml found
✅ user-service - .catalog.yml found
✅ payment-processor - .catalog.yml found
✅ notification-service - .catalog.yml found

Summary:
--------
Total repositories scanned: 8
Repositories with .catalog.yml: 8
Repositories missing .catalog.yml: 0

📋 Repositories with .catalog.yml files:
   - web-api (./repositories/web-api/.catalog.yml)
   - user-service (./repositories/user-service/.catalog.yml)
   - payment-processor (./repositories/payment-processor/.catalog.yml)
   - notification-service (./repositories/notification-service/.catalog.yml)
```

## Authentication

### GitHub Personal Access Token

1. Go to GitHub Settings → Developer settings → Personal access tokens
2. Generate a new token with `repo` scope for private repositories
3. Use the token with `-token=ghp_your_token_here`

**Required scopes:**
- `public_repo` - for public repositories
- `repo` - for private repositories

### GitLab Personal Access Token

1. Go to GitLab User Settings → Access Tokens
2. Create a token with `read_repository` scope
3. Use the token with `-token=glpat_your_token_here`

**Required scopes:**
- `read_repository` - to clone repositories
- `read_api` - to list repositories

## Directory Structure

The tool creates the following directory structure:

```
target-directory/
├── repo1/
│   ├── .git/
│   ├── README.md
│   └── ...
├── repo2/
│   ├── .git/
│   ├── src/
│   └── ...
└── repo3/
    ├── .git/
    └── ...
```

## Error Handling

- **Repository already exists**: Skipped with a warning message
- **Authentication failure**: Clear error message with suggestions
- **Network issues**: Retry logic for transient failures
- **Git clone failures**: Logged but don't stop the overall process

## Use Cases

### Backup and Archival
```bash
# Backup all company repositories
./git-repo-downloader -platform=github -org=mycompany -token=$GITHUB_TOKEN -dir=/backup/repos
```

### Migration Between Platforms
```bash
# Download from GitLab for migration to GitHub
./git-repo-downloader -platform=gitlab -org=old-company -token=$GITLAB_TOKEN -dir=./migration
```

### Bulk Analysis
```bash
# Download repositories for security scanning
./git-repo-downloader -platform=github -org=target-org -dir=./analysis
# Then run your analysis tools on ./analysis/*
```

### Development Environment Setup
```bash
# Set up development environment with all team repositories
./git-repo-downloader -platform=github -org=dev-team -token=$GITHUB_TOKEN -dir=~/dev
```

## Automation

### Using with CI/CD

```yaml
# GitHub Actions example
name: Repository Backup
on:
  schedule:
    - cron: '0 2 * * 0'  # Weekly on Sunday at 2 AM

jobs:
  backup:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Build downloader
        run: go build -o git-repo-downloader
      
      - name: Download repositories
        run: |
          ./git-repo-downloader -platform=github -org=mycompany -token=${{ secrets.GITHUB_TOKEN }} -dir=./backup
        
      - name: Upload backup
        # Add your backup storage logic here
```

### Using with Cron

```bash
# Add to crontab for weekly backups
0 2 * * 0 /path/to/git-repo-downloader -platform=github -org=mycompany -token=$GITHUB_TOKEN -dir=/backup/weekly
```

## Building for Multiple Platforms

```bash
# Build for all platforms
make build-all

# Manual cross-compilation
GOOS=linux GOARCH=amd64 go build -o git-repo-downloader-linux-amd64
GOOS=windows GOARCH=amd64 go build -o git-repo-downloader-windows-amd64.exe
GOOS=darwin GOARCH=amd64 go build -o git-repo-downloader-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o git-repo-downloader-darwin-arm64
```

## Troubleshooting

### Common Issues

1. **"Organization not found"**
   - Verify the organization/group name is correct
   - Ensure your token has access to the organization
   - For private organizations, make sure you're a member

2. **"Authentication failed"**
   - Check that your token is valid and not expired
   - Verify the token has the required scopes
   - For GitLab, ensure you're using the correct GitLab instance URL

3. **"Git clone failed"**
   - Ensure git is installed and in your PATH
   - Check network connectivity
   - For SSH cloning, ensure your SSH keys are properly configured

4. **"Permission denied"**
   - Check write permissions to the target directory
   - Ensure the target directory exists or can be created

### Debug Mode

For verbose output, you can modify the source to add debug logging or run with:

```bash
# Enable git verbose output
GIT_CURL_VERBOSE=1 ./git-repo-downloader -platform=github -org=myorg
```

## Security Considerations

- **Token Storage**: Never commit tokens to version control
- **Token Scope**: Use minimal required scopes for tokens
- **Network Security**: Be cautious when downloading from untrusted organizations
- **Local Storage**: Ensure downloaded repositories are stored securely

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Changelog

### v1.0.0
- Initial release
- GitHub organization support
- GitLab group support
- SSH and HTTPS cloning options
- Self-hosted GitLab support

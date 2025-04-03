# RDS IAM Connect

`rds-iam-connect` is a command-line tool for securely connecting to AWS RDS clusters using IAM authentication. It simplifies the process of generating IAM authentication tokens and establishing secure database connections without storing permanent credentials.

## Features

- **Secure IAM Authentication:** Uses temporary AWS IAM authentication tokens for RDS access.
- **Multi-Cluster Support:** Allows users to select from multiple RDS clusters filtered by tag.
- **Interactive CLI:** Provides an interactive command-line interface for user selections.
- **Configuration Management:** Uses a YAML configuration file for flexible settings.
- **Caching:** Caches the list of RDS clusters per environment to avoid unnecessary API calls.
- **Cross-Platform:** Built with Go and compatible with major operating systems.
- **Check IAM User:** Checks if the IAM user has the necessary permissions to connect to the RDS cluster.
- **Environment Management:** Supports multiple environments with different regions and release states.
- **Debug Mode:** Provides detailed logging for troubleshooting.
- **Configuration Validation:** Validates configuration and AWS credentials before use.

## Prerequisites

- Go 1.23.4 or later
- AWS CLI configured with appropriate credentials
- Access to AWS RDS instances with IAM authentication enabled. See an example of IAM policy below:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "rds:DescribeDBClusters",
                "rds:ListTagsForResource",
                "rds:GenerateDBAuthToken"
            ],
            "Resource": "*"
        }
    ]
}
```

## Installation

### Using Homebrew (Recommended)

1. **Add the tap repository:**
   ```bash
   brew tap ngelik/tap
   ```

2. **Install rds-iam-connect:**
   ```bash
   brew install rds-iam-connect
   ```

3. **Update to the latest version:**
   ```bash
   brew upgrade rds-iam-connect
   ```

### Manual Installation

If you prefer to install manually, follow these steps:

1. **Clone the Repository:**
   ```bash
   git clone https://github.com/yourorg/rds-iam-connect.git
   cd rds-iam-connect
   ```

2. **Build the Project:**
   
   The project includes a build script that can create binaries for multiple platforms. By default, it runs linting before building:

   ```bash
   # Build for your current platform (default: macOS ARM64)
   ./build.sh

   # Build for a specific platform
   ./build.sh linux    # Build for Linux AMD64
   ./build.sh windows  # Build for Windows AMD64
   
   # Build for all supported platforms
   ./build.sh all

   # Skip linting
   ./build.sh --no-lint     # Build without running linter
   ./build.sh all --no-lint # Build all platforms without linting
   ```

   The build script will:
   1. Check if `golangci-lint` is installed and install it if needed
   2. Run the linter to check code quality (unless `--no-lint` is specified)
   3. Build binaries for the specified platform(s)

   Supported platforms:
   - macOS (Apple Silicon/ARM64 and Intel/AMD64)
   - Linux (AMD64 and ARM64)
   - Windows (AMD64 and ARM64)

   The compiled binaries will be available in the `bin` directory with platform-specific names:
   ```
   bin/
   ├── rds-iam-connect-darwin-amd64    # macOS Intel
   ├── rds-iam-connect-darwin-arm64    # macOS Apple Silicon
   ├── rds-iam-connect-linux-amd64     # Linux Intel/AMD
   ├── rds-iam-connect-linux-arm64     # Linux ARM
   ├── rds-iam-connect-windows-amd64.exe
   └── rds-iam-connect-windows-arm64.exe
   ```

3. **Configure AWS Credentials:** Ensure your AWS credentials are set up correctly using `aws configure`.

4. **Create a Config File:**
   ```yaml
   # config.yaml
   rdsTags:
     tagName: "Environment"
     tagValue: "Production"
   allowedIAMUsers:
     - "db_user"

   envTag:
     prod:
       releaseState: "prod"
       region: "us-west-2"
     staging:
       releaseState: "staging"
       region: "us-east-1"

   caching:
     enabled: true
     duration: "1d"
   
   checkIAMPermissions: true
   debug: false
   ```

## Usage

1. **Run the Tool:**
   ```bash
   ./rds-iam-connect --config config.yaml
   ```

2. **Select RDS Cluster and IAM User:**
   The tool will prompt you to select an RDS cluster and IAM user interactively.

3. **Connect to RDS:**
   After selection, it will generate an IAM authentication token and connect to the RDS cluster using the `mysql` CLI.

### Check Mode

The tool includes a check mode that validates your configuration and AWS setup:

```bash
./rds-iam-connect --check
```

This will:
1. Verify AWS credentials
2. Validate configuration settings
3. Check RDS connectivity for each environment
4. Verify cache functionality

## Configuration

The configuration file is stored in `~/.rds-iam-connect/config.yaml` by default. On first run, if no configuration file exists, a default configuration will be created from `config.example.yaml`.

You can specify a different configuration file location using the `--config` flag:
```bash
./rds-iam-connect --config /path/to/your/config.yaml
```

The default configuration file location is determined as follows:
1. If no `--config` flag is provided, the tool will look for `config.yaml` in the current directory
2. If `config.yaml` is not found in the current directory, it will use `~/.rds-iam-connect/config.yaml`
3. If neither file exists, it will create a new configuration file at `~/.rds-iam-connect/config.yaml` using the example configuration

The configuration file (`config.yaml`) supports the following options:

```yaml
# RDS tags used to filter clusters
rdsTags:
  tagName: "Environment"  # Tag name to filter RDS clusters
  tagValue: "Production" # Tag value to match

# List of allowed IAM users
allowedIAMUsers:
  - "user1"
  - "user2"

# Environment configurations
envTag:
  prod:
    releaseState: "prod"  # Release state for production
    region: "us-west-2"   # AWS region
  staging:
    releaseState: "staging"
    region: "us-east-1"

# Cache settings
caching:
  enabled: true          # Enable/disable caching
  duration: "24h"        # Cache duration (e.g., "24h", "1h30m")

# Security settings
checkIAMPermissions: true  # Verify IAM permissions before connecting

# Debug mode
debug: false              # Enable detailed logging
```

## Caching

The tool implements an efficient caching system for RDS cluster information:

- **Location:** Cache files are stored in `~/.rds-iam-connect/` directory
- **Format:** JSON file containing cluster information and timestamp
- **Expiration:** Cache entries automatically expire based on configured duration
- **Environment Awareness:** Each environment has its own cache file (e.g., `rds-clusters-cache-prod.json`, `rds-clusters-cache-staging.json`)
- **Auto-refresh:** Expired cache is automatically refreshed with new API calls
- **Validation:** Cache files are validated for integrity and permissions
- **Error Handling:** Graceful fallback to API calls if cache is invalid or expired

### Cache Configuration

In your `config.yaml`, you can configure caching behavior:
```yaml
caching:
  enabled: true      # Enable/disable caching
  duration: "24h"    # Cache duration (e.g., "24h", "1h30m")
```

### Clearing Cache

To force a refresh of the cluster information, you can either:
- Delete the cache file for a specific environment: `rm ~/.rds-iam-connect/rds-clusters-cache-<env>.json`
- Delete all cache files: `rm ~/.rds-iam-connect/rds-clusters-cache-*.json`
- Disable caching in config: `enabled: false`

## Debug Mode

The tool includes a debug mode for troubleshooting:

```yaml
debug: true
```

When enabled, it provides detailed logging for:
- AWS API calls
- Cache operations
- Configuration loading
- IAM permission checks
- Connection attempts

## Best Practices

- Regularly rotate IAM credentials
- Use least-privilege IAM policies
- Monitor cache expiration settings based on your needs
- Keep the tool updated for security fixes
- Review AWS CloudTrail logs for RDS connection attempts
- Use debug mode for troubleshooting connection issues
- Regularly validate IAM permissions using the check mode

## Contributing

Contributions are welcome! Please follow these steps:
- Fork the repository.
- Create a feature branch.
- Commit your changes.
- Open a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Support

For support, open an issue in the GitHub repository or contact the maintainers.

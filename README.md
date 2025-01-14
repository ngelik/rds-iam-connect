# RDS IAM Connect

`rds-iam-connect` is a command-line tool for securely connecting to AWS RDS clusters using IAM authentication. It simplifies the process of generating IAM authentication tokens and establishing secure database connections without storing permanent credentials.

## Features

- **Secure IAM Authentication:** Uses temporary AWS IAM authentication tokens for RDS access.
- **Multi-Cluster Support:** Allows users to select from multiple RDS clusters filtered by tag.
- **Interactive CLI:** Provides an interactive command-line interface for user selections.
- **Configuration Management:** Uses a YAML configuration file for flexible settings.
- **Cross-Platform:** Built with Go and compatible with major operating systems.

## Prerequisites

- Go 1.23 or later
- AWS CLI configured with appropriate credentials
- Access to AWS RDS instances with IAM authentication enabled

## Project Structure

```plaintext
.github/workflows/     # CI/CD workflows
cmd/                   # Main CLI commands
config/                # Configuration management
internal/              # Core services and utilities
  aws/                 # AWS SDK interactions
  cli/                 # CLI interaction logic
  rds/                 # RDS interaction and token generation
  utils/               # Utility functions
main.go                # Entry point
README.md              # Project documentation
go.mod, go.sum         # Go module dependencies
```

## Setup Instructions

1. **Clone the Repository:**
   ```bash
   git clone https://github.com/yourorg/rds-iam-connect.git
   cd rds-iam-connect
   ```

2. **Build the Project:**
   ```bash
   go build -o rds-iam-connect ./...
   ```

3. **Configure AWS Credentials:** Ensure your AWS credentials are set up correctly using `aws configure`.

4. **Create a Config File:**
   ```yaml
   # config.yaml
   aws:
     tagName: "Environment"
     tagValue: "Production"
   allowedIAMUsers:
     - admin
     - dev_user
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

## Example Configuration

```yaml
aws:
  tagName: "Environment"
  tagValue: "Development"
allowedIAMUsers:
  - dev_user
  - test_user
```

## CI/CD

The project includes a GitHub Actions workflow to automate builds and tests:
- **Location:** `.github/workflows/go.yml`
- **Trigger:** Pushes and pull requests to `main`

```yaml
on:
  push:
    branches: ["main"]
  pull_request:
    branches: ["main"]
```

## Testing

Run tests using:
```bash
go test ./...
```

## Best Practices

- Regularly rotate IAM credentials.
- Use least-privilege IAM policies.
- Enable AWS CloudTrail logging.
- Monitor RDS connection attempts.

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


#!/bin/bash

# Exit on error
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
print_message() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if version argument is provided
if [ -z "$1" ]; then
    print_error "Usage: $0 <version>"
    print_error "Example: $0 v0.1.0"
    exit 1
fi

# Validate version format
if ! [[ $1 =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    print_error "Invalid version format. Must be in format vX.Y.Z (e.g., v0.1.0)"
    exit 1
fi

VERSION=$1
REPO="ngelik/rds-iam-connect"
RELEASE_DIR="release"
DRY_RUN=false
TAP_REPO="ngelik/homebrew-tap"
TAP_FORMULA="Formula/rds-iam-connect.rb"

# Parse arguments
for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN=true
            print_warning "Running in dry-run mode. No changes will be made."
            ;;
    esac
done

# Ensure Go bin directory is in PATH
export PATH="$(go env GOPATH)/bin:$PATH"

# Check for required tools
check_requirements() {
    print_message "Checking requirements..."
    
    # Check for gh CLI
    if ! command -v gh &> /dev/null; then
        print_error "GitHub CLI (gh) is not installed. Please install it first:"
        print_error "brew install gh"
        exit 1
    fi
    
    # Check for Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install it first:"
        print_error "brew install go"
        exit 1
    fi
    
    # Check GitHub authentication
    if ! gh auth status &> /dev/null; then
        print_error "Not authenticated with GitHub. Please run:"
        print_error "gh auth login"
        exit 1
    fi
}

# Create release directory
mkdir -p $RELEASE_DIR

# Build all binaries
build_binaries() {
    print_message "Building binaries..."
    if [ "$DRY_RUN" = true ]; then
        print_warning "Skipping build in dry-run mode"
        return
    fi
    
    # Build only for macOS ARM64
    print_message "Building for macOS ARM64..."
    GOOS=darwin GOARCH=arm64 go build -o bin/rds-iam-connect-darwin-arm64
    
    # Make binary executable
    chmod +x bin/rds-iam-connect-darwin-arm64
}

# Create git tag
create_tag() {
    print_message "Creating git tag $VERSION..."
    if [ "$DRY_RUN" = true ]; then
        print_warning "Would create and push tag: $VERSION"
        return
    fi
    
    if ! git tag -a $VERSION -m "Release $VERSION"; then
        print_error "Failed to create git tag"
        exit 1
    fi
    
    if ! git push origin $VERSION; then
        print_error "Failed to push git tag"
        exit 1
    fi
}

# Create GitHub release
create_release() {
    print_message "Creating GitHub release..."
    if [ "$DRY_RUN" = true ]; then
        print_warning "Would create release with the following assets:"
        ls -1 bin/rds-iam-connect-darwin-arm64 2>/dev/null || print_warning "No binaries found in bin directory"
        return
    fi
    
    if ! gh release create $VERSION \
        --title "$VERSION" \
        --notes "Release $VERSION" \
        bin/rds-iam-connect-darwin-arm64; then
        print_error "Failed to create GitHub release"
        exit 1
    fi
}

# Update Homebrew formula with SHA256 sums
update_formula() {
    print_message "Updating Homebrew formula with SHA256 sums..."
    if [ "$DRY_RUN" = true ]; then
        print_warning "Would update formula with SHA256 sums"
        return
    fi

    # Clone the tap repository
    TAP_DIR=$(mktemp -d)
    git clone "https://github.com/$TAP_REPO.git" "$TAP_DIR"
    cd "$TAP_DIR"

    # Calculate SHA256 sum for ARM64
    ARM64_SHA=$(shasum -a 256 "$OLDPWD/bin/rds-iam-connect-darwin-arm64" | cut -d' ' -f1)
    print_message "Calculated SHA256: $ARM64_SHA"

    # Create a temporary file for the new formula content
    TMP_FORMULA=$(mktemp)
    
    # Write the updated formula content
    cat > "$TMP_FORMULA" << EOF
class RdsIamConnect < Formula
    desc "CLI tool for securely connecting to AWS RDS clusters using IAM authentication"
    homepage "https://github.com/ngelik/rds-iam-connect"
    version "${VERSION#v}"
  
    if OS.mac?
      if Hardware::CPU.arm?
        url "https://github.com/ngelik/rds-iam-connect/releases/download/${VERSION}/rds-iam-connect-darwin-arm64"
        sha256 "$ARM64_SHA"
      end
    end
  
    depends_on "go" => :build
  
    def install
      bin.install "rds-iam-connect-darwin-arm64" => "rds-iam-connect"
    end
  
    def caveats
      <<~EOS
        Before using rds-iam-connect, make sure you have:
        1. AWS CLI configured with appropriate credentials
        2. Necessary IAM permissions for RDS access
        3. Created a configuration file at ~/.rds-iam-connect/config.yaml
  
        Example configuration can be found at:
        https://github.com/ngelik/rds-iam-connect#configuration
      EOS
    end
  
    test do
      assert_match "rds-iam-connect version #{version}", shell_output("#{bin}/rds-iam-connect --version", 2)
    end
end
EOF

    # Replace the formula file with the new content
    mv "$TMP_FORMULA" "$TAP_FORMULA"

    # Commit and push changes
    git add "$TAP_FORMULA"
    git commit -m "Update rds-iam-connect to $VERSION"
    git push origin main

    # Clean up
    cd - > /dev/null
    rm -rf "$TAP_DIR"
}

# Main execution
print_message "Starting release process for version $VERSION"
check_requirements
build_binaries
create_tag
create_release
update_formula

print_message "Release $VERSION created successfully!" 
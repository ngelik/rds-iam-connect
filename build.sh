#!/bin/bash

# Exit on error
set -e

# Function to run tests with detailed coverage
run_tests() {
    echo "Running tests with coverage..."
    
    # Run tests with coverage and generate detailed output
    go test -v -coverprofile=coverage.out -covermode=atomic -coverpkg=./... ./...
    
    # Generate HTML coverage report with function details
    go tool cover -html=coverage.out -o coverage.html
    
    # Generate coverage by function
    echo "Coverage by function:"
    go tool cover -func=coverage.out
    
    # Generate coverage by package
    echo -e "\nCoverage by package:"
    go test -cover ./... | grep -v "no test files"
    
    # Check coverage thresholds
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    MIN_COVERAGE=0.1
    
    if (( $(echo "$COVERAGE < $MIN_COVERAGE" | bc -l) )); then
        echo -e "\n❌ Coverage is below minimum threshold of $MIN_COVERAGE%"
        echo "Current coverage: $COVERAGE%"
        exit 1
    else
        echo -e "\n✅ Coverage is above minimum threshold of $MIN_COVERAGE%"
        echo "Current coverage: $COVERAGE%"
    fi
    
    # Generate test statistics
    echo -e "\nGenerating test statistics..."
    go test -json ./... > test-results.json
    
    # Generate test summary
    echo -e "\nTest Summary:"
    echo "Total packages: $(go test -list . ./... | wc -l)"
    echo "Test files: $(find . -name "*_test.go" | wc -l)"
    echo "Test functions: $(grep -r "func Test" --include="*_test.go" . | wc -l)"
}

# Function to run linter
run_linter() {
    echo "Running linter..."
    if ! command -v golangci-lint &> /dev/null; then
        echo "golangci-lint not found. Installing..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
    fi
    golangci-lint run
}

# Function to build the application
build_app() {
    echo "Building for $(uname -s)..."
    GOOS=$(uname -s | tr '[:upper:]' '[:lower:]') GOARCH=amd64 go build -o bin/rds-iam-connect-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64
    GOOS=$(uname -s | tr '[:upper:]' '[:lower:]') GOARCH=arm64 go build -o bin/rds-iam-connect-$(uname -s | tr '[:upper:]' '[:lower:]')-arm64
}

# Create bin directory if it doesn't exist
mkdir -p bin

# Get version from git tag or use default
VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}"

# Function to show usage
show_usage() {
    echo "Usage: $0 [platform] [--no-lint] [--no-test]"
    echo "Available platforms:"
    echo "  mac     - Build for macOS ARM64 (default)"
    echo "  linux   - Build for Linux AMD64"
    echo "  windows - Build for Windows AMD64"
    echo "  all     - Build for all platforms"
    echo ""
    echo "Options:"
    echo "  --no-lint  Skip linting step"
    echo "  --no-test  Skip test execution"
    echo ""
    echo "Examples:"
    echo "  $0          # Build for macOS ARM64 with linting and testing"
    echo "  $0 linux    # Build for Linux AMD64 with linting and testing"
    echo "  $0 all      # Build for all platforms with linting and testing"
    echo "  $0 --no-lint # Build for macOS ARM64 without linting and testing"
    echo "  $0 --no-test # Build for macOS ARM64 with linting but without testing"
}

# Parse command line arguments
NO_LINT=false
NO_TEST=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --no-lint)
            NO_LINT=true
            shift
            ;;
        --no-test)
            NO_TEST=true
            shift
            ;;
        linux|windows|darwin)
            PLATFORM="$1"
            shift
            ;;
        all)
            PLATFORM="all"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run linter if not skipped
if [ "$NO_LINT" = false ]; then
    run_linter
    if [ $? -ne 0 ]; then
        echo "Linting failed"
        exit 1
    fi
fi

# Run tests if not skipped
if [ "$NO_TEST" = false ]; then
    run_tests
    if [ $? -ne 0 ]; then
        echo "Tests failed"
        exit 1
    fi
fi

# Build for specified platform(s)
if [ -z "$PLATFORM" ]; then
    # Default to current platform
    PLATFORM=$(uname -s | tr '[:upper:]' '[:lower:]')
fi

case $PLATFORM in
    all)
        echo "Building for all platforms..."
        GOOS=darwin GOARCH=amd64 go build -o bin/rds-iam-connect-darwin-amd64
        GOOS=darwin GOARCH=arm64 go build -o bin/rds-iam-connect-darwin-arm64
        GOOS=linux GOARCH=amd64 go build -o bin/rds-iam-connect-linux-amd64
        GOOS=linux GOARCH=arm64 go build -o bin/rds-iam-connect-linux-arm64
        GOOS=windows GOARCH=amd64 go build -o bin/rds-iam-connect-windows-amd64.exe
        GOOS=windows GOARCH=arm64 go build -o bin/rds-iam-connect-windows-arm64.exe
        ;;
    darwin)
        echo "Building for macOS..."
        GOOS=darwin GOARCH=amd64 go build -o bin/rds-iam-connect-darwin-amd64
        GOOS=darwin GOARCH=arm64 go build -o bin/rds-iam-connect-darwin-arm64
        ;;
    linux)
        echo "Building for Linux..."
        GOOS=linux GOARCH=amd64 go build -o bin/rds-iam-connect-linux-amd64
        GOOS=linux GOARCH=arm64 go build -o bin/rds-iam-connect-linux-arm64
        ;;
    windows)
        echo "Building for Windows..."
        GOOS=windows GOARCH=amd64 go build -o bin/rds-iam-connect-windows-amd64.exe
        GOOS=windows GOARCH=arm64 go build -o bin/rds-iam-connect-windows-arm64.exe
        ;;
esac

# Make binaries executable
chmod +x bin/*

echo "Build completed successfully!"
echo "Binaries are available in the bin directory:"
ls -lh bin/ 
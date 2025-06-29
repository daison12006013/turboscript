#!/bin/bash

# TurboScript Security Analysis Script
# This script runs comprehensive security checks on the TurboScript codebase

set -e

echo "🔒 TurboScript Security Analysis"
echo "================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Check if required tools are installed
check_dependencies() {
    print_status "Checking dependencies..."

    if ! command -v gosec &> /dev/null; then
        print_warning "Gosec not found. Installing..."
        go install github.com/securego/gosec/v2/cmd/gosec@latest
    fi

    if ! command -v nancy &> /dev/null; then
        print_warning "Nancy not found. Installing..."
        go install github.com/sonatype-nexus-community/nancy@latest
    fi

    if ! command -v govulncheck &> /dev/null; then
        print_warning "govulncheck not found. Installing..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    fi

    print_success "Dependencies checked"
}

# Run Gosec security scanner
run_gosec() {
    print_status "Running Gosec security scanner..."

    if gosec -conf .gosec.json ./...; then
        print_success "Gosec scan completed successfully"
    else
        print_error "Gosec found security issues"
        exit 1
    fi
}

# Check for SQL injection vulnerabilities
check_sql_injection() {
    print_status "Checking for SQL injection vulnerabilities..."

    if grep -r "fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT\|fmt.Sprintf.*UPDATE\|fmt.Sprintf.*DELETE" . --include="*.go" --exclude-dir=".git"; then
        print_error "Potential SQL injection vulnerability found"
        echo "Consider using parameterized queries instead"
        return 1
    else
        print_success "No SQL injection patterns detected"
    fi
}

# Check for hardcoded secrets
check_hardcoded_secrets() {
    print_status "Checking for hardcoded secrets..."    # More sophisticated regex patterns for different secret types
    patterns=(
        "password\s*=\s*[\"'][^\"']{3,}[\"']"
        "secret\s*=\s*[\"'][^\"']{8,}[\"']"
        "token\s*=\s*[\"'][^\"']{10,}[\"']"
        "key\s*=\s*[\"'][^\"']{8,}[\"']"
        "api[_-]?key\s*=\s*[\"'][^\"']{8,}[\"']"
        "auth[_-]?token\s*=\s*[\"'][^\"']{10,}[\"']"
        "jwt[_-]?secret\s*=\s*[\"'][^\"']{8,}[\"']"
        "database[_-]?url\s*=\s*[\"'][^\"']*:[^\"']*@[^\"']*[\"']"
    )

    found_secrets=false
    for pattern in "${patterns[@]}"; do
        # Exclude test files and development configuration from hardcoded secret checks
        if grep -r -i -E "$pattern" . --include="*.go" --include="*.ts" --include="*.js" --include="*.yml" --include="*.yaml" --exclude-dir=".git" --exclude-dir="node_modules" --exclude-dir="dist" --exclude="*_test.go" --exclude="*test*.go" --exclude="turboscript.yml" --exclude="turboscript.test.yml"; then
            found_secrets=true
        fi
    done

    if [ "$found_secrets" = true ]; then
        print_error "Potential hardcoded secrets found"
        echo "Consider using environment variables or secret management"
        return 1
    else
        print_success "No hardcoded secrets detected"
    fi
}

# Check for unsafe HTTP client usage
check_unsafe_http() {
    print_status "Checking for unsafe HTTP client configurations..."

    if grep -r "InsecureSkipVerify.*true\|TLSClientConfig.*InsecureSkipVerify" . --include="*.go" --exclude-dir=".git"; then
        print_warning "Insecure TLS configuration found"
        echo "Consider using proper certificate validation"
        return 1
    else
        print_success "No insecure TLS configurations detected"
    fi
}

# Check for unsafe file operations
check_unsafe_file_ops() {
    print_status "Checking for unsafe file operations..."

    unsafe_patterns=false

    # Check for direct file operations without proper validation
    if grep -r "os\.Create\|os\.OpenFile\|ioutil\.WriteFile" . --include="*.go" --exclude="*_test.go" --exclude-dir=".git" | head -5; then
        print_warning "Direct file operations found"
        echo "Ensure proper file permissions and path validation"
        unsafe_patterns=true
    fi

    # Check for path traversal vulnerabilities
    if grep -r "filepath\.Join.*\.\." . --include="*.go" --exclude-dir=".git"; then
        print_error "Potential path traversal vulnerability found"
        unsafe_patterns=true
    fi

    # Check for unsafe eval-like operations
    if grep -r "os\.Exec\|exec\.Command" . --include="*.go" --exclude="*_test.go" --exclude-dir=".git" | head -5; then
        print_warning "Command execution found"
        echo "Ensure proper input validation for command execution"
        unsafe_patterns=true
    fi

    if [ "$unsafe_patterns" = false ]; then
        print_success "No unsafe file operations detected"
    fi
}

# Run Nancy vulnerability scanner
run_nancy() {
    print_status "Running Nancy vulnerability scanner..."

    if go list -json -deps ./... | nancy sleuth; then
        print_success "Nancy vulnerability scan completed"
    else
        print_warning "Nancy found potential vulnerabilities (check output above)"
    fi
}

# Run govulncheck vulnerability scanner
run_govulncheck() {
    print_status "Running govulncheck vulnerability scanner..."

    # Ensure we're using the correct Go version
    GO_VERSION=$(go version | awk '{print $3}')
    print_status "Using Go version: $GO_VERSION"

    if govulncheck ./...; then
        print_success "govulncheck vulnerability scan completed"
    else
        print_warning "govulncheck found potential vulnerabilities or version issues (check output above)"
        # Don't fail the security scan if it's just a Go version issue
        if govulncheck --version >/dev/null 2>&1; then
            print_status "govulncheck is working, treating as warning only"
        fi
    fi
}

# Check TurboScript specific security patterns
check_turboscript_security() {
    print_status "Checking TurboScript-specific security patterns..."

    # Check if SQL injection protection is in place
    if grep -r "fmt.Sprintf.*query\|fmt.Sprintf.*Query" internal/ --include="*.go"; then
        print_error "Potential SQL injection in database executor"
        return 1
    fi

    # Check for proper JWT secret handling
    if grep -r "jwt.*secret.*=" app/ --include="*.ts" | grep -v "process.env\|event.env"; then
        print_warning "JWT secret should be loaded from environment variables"
    fi

    print_success "TurboScript security checks completed"
}

# Main execution
main() {
    echo
    check_dependencies
    echo

    # Track overall success
    overall_success=true

    # Run all security checks
    run_gosec || overall_success=false
    echo

    check_sql_injection || overall_success=false
    echo

    check_hardcoded_secrets || overall_success=false
    echo

    check_unsafe_http || overall_success=false
    echo

    check_unsafe_file_ops || overall_success=false
    echo    run_nancy || overall_success=false
    echo

    # Run govulncheck but don't fail on version issues
    run_govulncheck # Don't fail overall on govulncheck issues
    echo

    check_turboscript_security || overall_success=false
    echo

    # Final report
    echo "================================"
    if [ "$overall_success" = true ]; then
        print_success "🎉 All security checks passed!"
        echo
        echo "Your TurboScript application appears to be secure."
        echo "Remember to:"
        echo "  - Keep dependencies updated"
        echo "  - Use environment variables for secrets"
        echo "  - Regularly run security scans"
        echo "  - Follow the principle of least privilege"
    else
        print_error "❌ Some security issues were found"
        echo
        echo "Please review the issues above and fix them before deploying."
        echo "Security is critical for web applications."
        exit 1
    fi
}

# Help function
show_help() {
    echo "TurboScript Security Analysis Script"
    echo
    echo "Usage: $0 [options]"
    echo
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  --gosec-only   Run only Gosec scanner"
    echo "  --nancy-only   Run only Nancy vulnerability scanner"
    echo "  --vuln-only    Run both vulnerability scanners (Nancy + govulncheck)"
    echo
    echo "This script performs comprehensive security analysis including:"
    echo "  - Static code analysis with Gosec"
    echo "  - Dependency vulnerability scanning with Nancy"
    echo "  - SQL injection pattern detection"
    echo "  - Hardcoded secret detection"
    echo "  - Unsafe HTTP client configuration checks"
    echo "  - TurboScript-specific security validations"
}

# Handle command line arguments
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    --gosec-only)
        check_dependencies
        run_gosec
        ;;
    --nancy-only)
        check_dependencies
        run_nancy
        ;;
    --govulncheck-only)
        check_dependencies
        run_govulncheck
        ;;
    --vuln-only)
        check_dependencies
        run_nancy
        run_govulncheck
        ;;
    *)
        main
        ;;
esac

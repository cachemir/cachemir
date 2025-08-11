# CacheMir CI/CD Documentation

CacheMir includes a comprehensive CI/CD pipeline built with GitHub Actions that provides automated testing, security scanning, releases, and deployment.

## 🚀 Overview

The CI/CD system consists of several automated workflows:

- **CI Workflow**: Comprehensive testing and quality checks on PRs
- **Release Workflow**: Automated versioning and binary releases
- **Docker Workflow**: Container builds and registry publishing
- **PR Comment Workflow**: Automated test result comments
- **CodeQL Workflow**: Security analysis and vulnerability scanning

## 📋 Workflows

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers**: Push to main/develop, Pull Requests, Manual dispatch

**Features**:
- **Matrix Testing**: Go 1.20, 1.21, 1.22 on Ubuntu, macOS, Windows
- **Code Quality**: golangci-lint, go vet, gofmt validation
- **Testing**: Unit tests, race detection, benchmarks
- **Coverage**: Codecov integration with coverage reports
- **Security**: Gosec security scanning
- **Artifacts**: Build binaries and upload test results

**Jobs**:
1. **Test**: Run tests across multiple Go versions and OS
2. **Lint**: Code quality and style checks
3. **Security**: Security vulnerability scanning
4. **Build**: Compile binaries and create artifacts

### 2. Release Workflow (`.github/workflows/release.yml`)

**Triggers**: Git tags (v*), Manual dispatch

**Features**:
- **Semantic Versioning**: Automatic version detection
- **Multi-platform Builds**: Linux, macOS, Windows (AMD64, ARM64)
- **Release Notes**: Auto-generated from commit history
- **GitHub Releases**: Automated release creation
- **Binary Distribution**: Cross-platform binary artifacts

**Process**:
1. **Create Release**: Generate release notes and GitHub release
2. **Build Binaries**: Cross-compile for all supported platforms
3. **Upload Assets**: Attach binaries to GitHub release

### 3. Docker Workflow (`.github/workflows/docker.yml`)

**Triggers**: Push to main, Git tags, Pull Requests, Manual dispatch

**Features**:
- **Multi-arch Builds**: AMD64 and ARM64 support
- **Container Registry**: Push to GitHub Container Registry (ghcr.io)
- **Image Tagging**: Latest, version, branch, and SHA tags
- **Security Scanning**: Trivy vulnerability scanning
- **Build Caching**: GitHub Actions cache for faster builds

**Images**:
- `ghcr.io/cachemir/cachemir:latest` - Latest main branch
- `ghcr.io/cachemir/cachemir:v1.0.0` - Version tags
- `ghcr.io/cachemir/cachemir:main-abc1234` - Branch + SHA

### 4. PR Comment Workflow (`.github/workflows/pr-comment.yml`)

**Triggers**: CI workflow completion on PRs

**Features**:
- **Test Results**: Detailed test output in PR comments
- **Coverage Reports**: Coverage percentage and trends
- **Benchmark Results**: Performance metrics
- **Status Updates**: Success/failure indicators
- **Workflow Links**: Direct links to CI runs

**Comment Format**:
```markdown
## 🚀 CI Results

**Status**: ✅ SUCCESS

### 🟢 Test Coverage
**85.2%** of code is covered by tests

### ⚡ Benchmark Results
BenchmarkGet-8    1000000    1234 ns/op    456 B/op    7 allocs/op

### 📊 Details
- **Workflow Run**: [#123](link)
- **Commit**: abc1234
- **Branch**: feature/new-feature
```

### 5. CodeQL Workflow (`.github/workflows/codeql.yml`)

**Triggers**: Push to main/develop, PRs, Weekly schedule

**Features**:
- **Static Analysis**: GitHub's semantic code analysis
- **Security Scanning**: Vulnerability detection
- **Quality Checks**: Code quality and maintainability
- **SARIF Integration**: Security findings in GitHub Security tab

## 🔧 Configuration

### Environment Variables

The workflows use these environment variables:

```yaml
# Global settings
GO_VERSION: '1.21'
REGISTRY: ghcr.io
IMAGE_NAME: ${{ github.repository }}

# Secrets (configured in repository settings)
GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}  # Automatic
CODECOV_TOKEN: ${{ secrets.CODECOV_TOKEN }}  # Optional
```

### golangci-lint Configuration

The project includes a comprehensive `.golangci.yml` configuration with:

- **30+ Linters**: Comprehensive code quality checks
- **Custom Rules**: Project-specific linting rules
- **Test Exclusions**: Relaxed rules for test files
- **Performance Focus**: Optimized for Go best practices

### Dependabot Configuration

Automated dependency updates for:

- **Go Modules**: Weekly updates on Mondays
- **GitHub Actions**: Weekly action version updates
- **Docker**: Base image updates

## 📊 Quality Gates

### Pull Request Requirements

Before merging, PRs must pass:

1. **All Tests**: Unit tests across all Go versions and platforms
2. **Linting**: golangci-lint with zero issues
3. **Security**: Gosec security scan with no high-severity issues
4. **Build**: Successful compilation on all platforms
5. **Coverage**: Maintain or improve test coverage

### Release Requirements

Releases are created when:

1. **Tag Format**: Git tag matches `v*` pattern (e.g., `v1.0.0`)
2. **CI Success**: All CI checks pass on the tagged commit
3. **Security**: No known vulnerabilities in dependencies

## 🚀 Deployment Process

### Automated Release Process

1. **Tag Creation**: Developer creates a git tag (`git tag v1.0.0`)
2. **Push Tag**: Tag is pushed to GitHub (`git push origin v1.0.0`)
3. **Trigger Release**: Release workflow automatically starts
4. **Build Artifacts**: Binaries built for all platforms
5. **Create Release**: GitHub release created with artifacts
6. **Container Build**: Docker images built and pushed
7. **Notifications**: Team notified of successful release

### Manual Release Process

1. **Go to Actions**: Navigate to GitHub Actions tab
2. **Select Release**: Choose "Release" workflow
3. **Run Workflow**: Click "Run workflow" button
4. **Enter Version**: Specify version (e.g., `v1.0.0`)
5. **Execute**: Workflow runs with manual version

### Container Deployment

```bash
# Pull latest image
docker pull ghcr.io/cachemir/cachemir:latest

# Run container
docker run -p 8080:8080 ghcr.io/cachemir/cachemir:latest

# Or specific version
docker pull ghcr.io/cachemir/cachemir:v1.0.0
docker run -p 8080:8080 ghcr.io/cachemir/cachemir:v1.0.0
```

## 📈 Monitoring and Metrics

### Build Metrics

The CI/CD system tracks:

- **Build Times**: Duration of each workflow
- **Test Results**: Pass/fail rates and trends
- **Coverage**: Code coverage trends over time
- **Security**: Vulnerability counts and severity
- **Performance**: Benchmark results and regressions

### Quality Metrics

- **Code Quality**: Linting issues and technical debt
- **Test Coverage**: Line and branch coverage percentages
- **Security Score**: Vulnerability assessment results
- **Performance**: Benchmark trends and regression detection

## 🔒 Security

### Security Scanning

Multiple layers of security scanning:

1. **CodeQL**: Static analysis for security vulnerabilities
2. **Gosec**: Go-specific security issue detection
3. **Trivy**: Container vulnerability scanning
4. **Dependabot**: Dependency vulnerability alerts

### Security Reporting

- **SARIF Integration**: Security findings in GitHub Security tab
- **Vulnerability Alerts**: Automated alerts for new vulnerabilities
- **Security Advisories**: Private vulnerability reporting
- **Dependency Updates**: Automated security patches

## 🛠️ Development Workflow

### Contributing Process

1. **Fork Repository**: Create a fork of the main repository
2. **Create Branch**: Create feature branch from main
3. **Make Changes**: Implement changes with tests
4. **Run Tests**: Ensure all tests pass locally
5. **Create PR**: Submit pull request with description
6. **CI Validation**: Automated CI checks run
7. **Code Review**: Maintainer review and feedback
8. **Merge**: PR merged after approval and CI success

### Local Development

```bash
# Run tests locally
make test

# Run linting
make lint

# Run all quality checks
make check

# Build binaries
make build

# Run benchmarks
go test -bench=. ./...
```

## 🎯 Best Practices

### Commit Messages

Use conventional commit format for automatic changelog generation:

```
feat: add new caching algorithm
fix: resolve memory leak in connection pool
docs: update API documentation
chore: update dependencies
```

### Pull Request Guidelines

- **Clear Description**: Explain what and why
- **Test Coverage**: Include tests for new functionality
- **Documentation**: Update docs for API changes
- **Breaking Changes**: Clearly mark breaking changes
- **Performance**: Consider performance implications

### Release Guidelines

- **Semantic Versioning**: Follow semver (MAJOR.MINOR.PATCH)
- **Release Notes**: Clear description of changes
- **Migration Guide**: Document breaking changes
- **Testing**: Thorough testing before release

## 🔧 Troubleshooting

### Common Issues

**CI Failures**:
- Check test output in workflow logs
- Verify code formatting with `gofmt`
- Run linting locally with `golangci-lint run`

**Release Issues**:
- Ensure tag format matches `v*` pattern
- Verify all CI checks pass before tagging
- Check GitHub token permissions

**Docker Issues**:
- Verify Dockerfile syntax
- Check base image availability
- Ensure multi-arch build compatibility

### Getting Help

- **GitHub Issues**: Report bugs and request features
- **Discussions**: Ask questions and share ideas
- **Documentation**: Check docs for detailed guides
- **Security**: Use private security advisories for vulnerabilities

The CI/CD system is designed to be robust, secure, and developer-friendly, providing comprehensive automation while maintaining high code quality and security standards.

# Security Policy

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability in TurboScript, please report it to us privately.

### How to Report

1. **DO NOT** create a public GitHub issue for security vulnerabilities
2. Email security reports to: <daison12006013@gmail.com>
3. Include the following information:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if known)

### What to Expect

- **Acknowledgment**: We will acknowledge receipt of your report within 24 hours
- **Investigation**: We will investigate and validate the vulnerability within 5 business days
- **Resolution**: We will work to fix confirmed vulnerabilities within 30 days
- **Disclosure**: We will coordinate with you on responsible disclosure timing

### Security Measures

TurboScript implements several security measures:

#### Application Security

- **Input Validation**: All user inputs are validated before processing
- **JWT Security**: Secure token handling with configurable expiration
- **Password Security**: Bcrypt hashing for password storage
- **CORS Protection**: Configurable CORS settings

#### Runtime Security

- **Sandboxed Execution**: TypeScript code runs in a controlled JavaScript VM (goja)
- **Resource Limits**: Memory and execution time limits for TypeScript code
- **Error Handling**: Secure error messages that don't leak sensitive information

#### Infrastructure Security

- **HTTPS Support**: TLS/SSL support for encrypted communication
- **Environment Variables**: Secrets managed through environment variables
- **Docker Security**: Minimal base images and non-root user execution

### Security Best Practices for Users

When deploying TurboScript:

1. **Database Configuration**
   - Use strong database passwords
   - Limit database user permissions
   - Configure network-level database access restrictions

2. **Application Configuration**
   - Set `debug: false` in production
   - Use environment variables for secrets
   - Set strong JWT secrets

3. **Deployment Security**
   - Use HTTPS in production
   - Keep dependencies updated
   - Monitor application logs
   - Regular security scans

4. **Code Security**
   - Validate all user inputs in TypeScript routes
   - Use the provided security utilities
   - Follow the principle of least privilege
   - Regular dependency audits

### Automated Security Scanning

This project uses automated security scanning:

- **Gosec**: Static analysis for Go code security issues
- **Nancy**: Vulnerability scanning for Go dependencies
- **Custom Checks**: TurboScript-specific security validations
- **CI/CD Integration**: Security checks run on every commit

To run security scans locally:

```bash
make security          # Run all security checks
make security-gosec    # Run only static analysis
make security-deps     # Check dependencies only
```

### Security Updates

Security updates are released as patch versions and include:

- CVE fixes
- Dependency updates
- Security configuration improvements
- Documentation updates

Subscribe to releases to stay informed about security updates.

### Hall of Fame

We recognize security researchers who help improve TurboScript security:

<!-- Future contributors will be listed here -->

---

Thank you for helping keep TurboScript secure! 🔒

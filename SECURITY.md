# Security Configuration Guide

## Environment Variables

ReignX uses environment variables for sensitive configuration values. **Never commit passwords, secrets, or API keys to version control.**

### Required Environment Variables

Before running ReignX, set the following environment variables:

```bash
# Database password
export DATABASE_PASSWORD="your_secure_password"

# JWT secret for token signing (generate with: openssl rand -base64 32)
export JWT_SECRET="your_random_jwt_secret"

# Encryption key for SSH password storage (generate with: openssl rand -base64 32)
export REIGNX_ENCRYPTION_KEY="your_random_encryption_key"
```

### Optional Environment Variables

```bash
# SSH password (optional, prefer using SSH keys)
export SSH_PASSWORD=""

# NATS authentication (if enabled)
export NATS_USERNAME=""
export NATS_PASSWORD=""

# Etcd authentication (if enabled)
export ETCD_USERNAME=""
export ETCD_PASSWORD=""
```

## Configuration Files

### Setup Steps

1. **Copy the example environment file:**
   ```bash
   cp .env.example .env
   ```

2. **Edit `.env` and fill in your secure values:**
   ```bash
   # Generate secure secrets
   openssl rand -base64 32  # Use for JWT_SECRET
   openssl rand -base64 32  # Use for REIGNX_ENCRYPTION_KEY

   # Edit the file
   vim .env
   ```

3. **Load environment variables:**
   ```bash
   source .env
   # or use direnv for automatic loading
   direnv allow
   ```

4. **Verify configuration:**
   ```bash
   echo $DATABASE_PASSWORD  # Should show your password
   echo $JWT_SECRET         # Should show your JWT secret
   ```

### Configuration File Security

The configuration files (`config/*.yaml`) reference environment variables using `${VAR_NAME}` syntax:

```yaml
database:
  password: ${DATABASE_PASSWORD}

security:
  jwt_secret: ${JWT_SECRET}
```

**DO NOT** replace these with hardcoded values. Always use environment variables for sensitive data.

## Production Deployment Checklist

### 1. Secrets Management

- [ ] Use environment variables for all passwords and secrets
- [ ] Generate strong random values for JWT_SECRET and REIGNX_ENCRYPTION_KEY
- [ ] Store secrets in a secure secret management system (HashiCorp Vault, AWS Secrets Manager, etc.)
- [ ] Rotate secrets regularly (at least every 90 days)

### 2. Database Security

- [ ] Use a strong database password (min 16 characters, mixed case, numbers, symbols)
- [ ] Enable SSL/TLS for database connections (`sslmode: require` or `verify-full`)
- [ ] Restrict database access to specific IP addresses
- [ ] Use separate database users with minimal required privileges
- [ ] Enable database audit logging

### 3. SSH Security

- [ ] Use SSH keys instead of passwords wherever possible
- [ ] Disable SSH password authentication in production
- [ ] Implement SSH host key verification (already enabled in ReignX)
- [ ] Rotate SSH keys periodically
- [ ] Use strong SSH key encryption (Ed25519 or RSA 4096-bit)

### 4. TLS/mTLS Configuration

- [ ] Enable TLS for all gRPC and HTTP endpoints
- [ ] Use valid TLS certificates from a trusted CA
- [ ] Enable mutual TLS (mTLS) for agent-server communication
- [ ] Configure proper certificate validation
- [ ] Set up certificate rotation automation

### 5. JWT Token Security

- [ ] Use a cryptographically secure JWT secret (min 32 bytes)
- [ ] Set appropriate token expiry times (default: 1h for access, 7d for refresh)
- [ ] Implement token refresh mechanism
- [ ] Use HTTPS only for token transmission
- [ ] Implement token revocation for logout

### 6. Access Control

- [ ] Enable RBAC (Role-Based Access Control)
- [ ] Follow principle of least privilege
- [ ] Regularly audit user permissions
- [ ] Implement strong password policies
- [ ] Enable multi-factor authentication (MFA) where possible

### 7. Network Security

- [ ] Use firewall rules to restrict access to services
- [ ] Enable rate limiting on API endpoints
- [ ] Implement DDoS protection
- [ ] Use VPN or private networks for internal communication
- [ ] Segment network by security zones

### 8. Audit and Monitoring

- [ ] Enable session recording and audit logging
- [ ] Monitor for suspicious activities
- [ ] Set up alerts for security events
- [ ] Regularly review audit logs
- [ ] Implement log retention policies

### 9. Container Security

- [ ] Use minimal base images (Alpine, Distroless)
- [ ] Scan container images for vulnerabilities
- [ ] Run containers as non-root user
- [ ] Use read-only root filesystems where possible
- [ ] Implement container resource limits

### 10. Regular Maintenance

- [ ] Keep all dependencies up to date
- [ ] Apply security patches promptly
- [ ] Perform regular security audits
- [ ] Conduct penetration testing
- [ ] Have an incident response plan

## Reporting Security Issues

If you discover a security vulnerability, please email security@reignx.io (or your organization's security contact) instead of opening a public issue.

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will acknowledge your email within 48 hours and provide a detailed response within 7 days.

## Security Features in ReignX

### Implemented
- ✅ JWT authentication with access/refresh tokens
- ✅ AES-256 encryption for SSH passwords in database
- ✅ SSH host key verification (TOFU pattern)
- ✅ Session recording for audit trail
- ✅ TLS/mTLS support for agent communication
- ✅ Password hashing with bcrypt
- ✅ SQL injection prevention (parameterized queries)

### Roadmap
- 🔄 RBAC (Role-Based Access Control)
- 🔄 Multi-factor authentication (MFA)
- 🔄 API rate limiting
- 🔄 IP whitelisting
- 🔄 Security event alerting
- 🔄 Compliance reporting (SOC 2, HIPAA, etc.)

## Best Practices

### Development
```bash
# Never commit .env files
echo ".env" >> .gitignore
echo "*.env" >> .gitignore

# Use different secrets for dev/staging/prod
# DEV: Simple passwords are OK
export DATABASE_PASSWORD="dev_password"

# PROD: Generate strong random passwords
export DATABASE_PASSWORD=$(openssl rand -base64 32)
```

### Testing
```bash
# Use separate test database and credentials
export DATABASE_PASSWORD="test_password"
export DATABASE_NAME="reignx_test"
```

### Production
```bash
# Load from secure secret manager
export DATABASE_PASSWORD=$(vault kv get -field=password secret/reignx/db)
export JWT_SECRET=$(vault kv get -field=secret secret/reignx/jwt)
export REIGNX_ENCRYPTION_KEY=$(vault kv get -field=key secret/reignx/encryption)
```

## Additional Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [Go Security Checklist](https://github.com/guardrailsio/awesome-golang-security)

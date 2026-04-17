# Git Hooks

This directory contains Git hooks to prevent security incidents.

## Available Hooks

### pre-commit

Prevents committing passwords, API keys, and other secrets to the repository.

**Features:**
- Detects hardcoded passwords and secrets
- Warns about certificate/key files
- Blocks commits containing sensitive data
- Allows environment variable references (${VAR})

**Installation:**

```bash
# Install the hook
cp hooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit

# Test the hook
git add .
git commit -m "test"  # Will check for secrets
```

**Detected Patterns:**

- `password: "xxx"` (but allows `password: "${DATABASE_PASSWORD}"`)
- `passwd: "xxx"`
- `reignx123` (old test password)
- `CHANGEME_DB_PASSWORD` (placeholder)
- API keys, secret keys, tokens (20+ alphanumeric characters)
- Certificate files (`.pem`, `.key`, `.crt`, `.p12`, `.pfx`)

**Bypassing the Hook (NOT RECOMMENDED):**

```bash
git commit --no-verify -m "commit message"
```

Only use `--no-verify` if you're absolutely certain the detected patterns are false positives.

## Best Practices

1. **Use environment variables** for all secrets
2. **Never commit** `.env`, `.secrets`, or credential files
3. **Use placeholders** like `changeme`, `CHANGE_ME`, `your_password_here` in examples
4. **Document** which environment variables are required in `.env.example`

## Related Files

- [.secrets.example](../.secrets.example) - Template for local secrets
- [.env.example](../.env.example) - Environment variable template
- [SECURITY.md](../SECURITY.md) - Security best practices
- [SECURITY_FIXES.md](../SECURITY_FIXES.md) - Past security issues

## Troubleshooting

### False Positives

If the hook incorrectly flags safe content:

1. Check if you're using placeholders like `changeme` or `your_password_here`
2. Verify environment variables are in the format `${VAR_NAME}`
3. Update the hook patterns in `hooks/pre-commit` if needed

### Hook Not Running

Make sure the hook is:
1. Copied to `.git/hooks/` (not just `hooks/`)
2. Executable: `chmod +x .git/hooks/pre-commit`
3. Not disabled: Check for `.git/hooks/pre-commit.sample`

## Contributing

When adding new patterns:
1. Add the regex pattern to the `PATTERNS` array
2. Test with sample files
3. Document in this README
4. Consider false positives

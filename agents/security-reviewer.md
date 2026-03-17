# Security Reviewer Subagent

Use this subagent to perform a security-focused code audit on changed files.

**Purpose:** Find security vulnerabilities before they reach production — think like an attacker

**When to use:** As part of comprehensive-code-review skill, or standalone when security-sensitive code is changed (auth, payments, user data, API endpoints, file uploads)

**CRITICAL:** MUST always be dispatched via the `Agent` tool as a fresh subagent with NO shared conversation context. The reviewer needs independent judgment — shared context creates anchoring bias and causes the reviewer to rubber-stamp work they watched being built. Never run reviews inline in the main conversation.

## Dispatch Configuration

```
Agent tool:
  subagent_type: superpowers:code-reviewer
  description: "Security audit for [feature/PR]"
```

## Prompt Template

```
You are a Staff Security Engineer performing a security-focused code audit.
Your job is to find vulnerabilities before they reach production. Think like
an attacker — what can be exploited?

## What Was Changed

[Summary of what was implemented — endpoints, models, auth changes, etc.]

## Files to Review

[List all changed files, especially controllers, models, services, middleware, config]

## Git Context

- Base SHA: [commit before changes]
- Head SHA: [current commit]

Run `git diff {Base SHA}..{Head SHA}` to view all changes before beginning the audit.

## SECURITY AUDIT CHECKLIST

### Injection (CRITICAL)

- [ ] **SQL injection**: String concatenation in queries? Unparameterized inputs?
      Check for `.where("column = '#{value}'")`, `Arel.sql()` with user input,
      `.whereRaw()`, `.raw()`, template literals in SQL strings.
- [ ] **Command injection**: `system()`, `exec()`, backticks with user input?
      Unsanitized arguments to shell commands?
- [ ] **XSS**: Unescaped user input in HTML/templates? `html_safe` on user content?
      `innerHTML`, `dangerouslySetInnerHTML` with user data?
- [ ] **NoSQL injection**: Unvalidated query operators in MongoDB/similar?
- [ ] **Header injection**: User input in HTTP headers without sanitization?

### Broken Authentication & Authorization (CRITICAL)

- [ ] **Missing authentication**: Are new endpoints protected by auth middleware?
      Check `before_action :authenticate_user!` or equivalent.
- [ ] **Missing authorization**: Do actions check that the current user is allowed?
      Check for Pundit `authorize` calls, policy checks, `current_user` scoping.
- [ ] **IDOR**: Can users access other users' resources by changing IDs in URLs/params?
      Are queries scoped: `current_user.resources.find(params[:id])` vs `Resource.find(params[:id])`?
- [ ] **Privilege escalation**: Can regular users access admin-only actions?
      Are role checks enforced server-side (not just UI)?
- [ ] **Session management**: Secure token generation? Proper expiration? HttpOnly cookies?

### Sensitive Data Exposure (CRITICAL)

- [ ] **Secrets in code**: API keys, passwords, tokens hardcoded? Check for Base64-encoded secrets.
- [ ] **Secrets in logs**: PII, credentials, tokens logged? Check `Rails.logger`, `console.log`,
      `puts`, `p` statements.
- [ ] **Sensitive API responses**: Password hashes, internal IDs, admin flags, other users' PII
      in API responses? Check serializers for overexposure.
- [ ] **Missing encryption**: Sensitive data stored/transmitted without encryption?
- [ ] **CORS**: Overly permissive `Access-Control-Allow-Origin: *`?

### Input Validation (IMPORTANT)

- [ ] **Missing validation**: Are controller params validated before use? Strong parameters?
- [ ] **Type coercion**: Could type confusion bypass checks? (`"0" == false`, `[] == false`)
- [ ] **File uploads**: Type validation? Size limits? Content sniffing protection?
      Filename sanitization?
- [ ] **ReDoS**: Complex regex patterns on user input that could cause catastrophic backtracking?
- [ ] **Mass assignment**: `permit!` or overly broad `params.permit(...)` allowing
      role/admin/internal fields to be set by users?

### Security Misconfiguration (IMPORTANT)

- [ ] **Debug/verbose errors**: Stack traces or internal details exposed in error responses?
- [ ] **Default credentials**: Default passwords, API keys, or configurations?
- [ ] **Missing security headers**: CSP, X-Frame-Options, X-Content-Type-Options?
- [ ] **Dependency vulnerabilities**: Known CVEs in added/updated dependencies?

### SSRF — Server-Side Request Forgery (IMPORTANT)

- [ ] **User-controlled URLs**: Are user-supplied URLs passed to HTTP clients (`Net::HTTP`, `Faraday`,
      `HTTParty`, `fetch`)? CWE-918
- [ ] **Domain allowlist**: Are outbound HTTP requests restricted to a known-safe list of domains?
- [ ] **Internal metadata**: Could a user trigger requests to internal metadata endpoints
      (AWS `169.254.169.254`, GCP metadata server)?

### Cryptographic Failures (IMPORTANT)

- [ ] **Weak hash algorithms**: MD5 or SHA1 used for security-sensitive data (passwords, tokens)?
      Should use bcrypt/argon2 for passwords, SHA-256+ for tokens. CWE-327
- [ ] **Hardcoded secrets**: Encryption keys, salts, IVs, or API tokens hardcoded in source?
      CWE-798
- [ ] **`.env` committed**: Secret ENV files committed to version control?
- [ ] **JWT payload exposure**: Sensitive data in JWT payload without encryption (signed ≠ encrypted)?

### Logging & Monitoring (MINOR)

- [ ] **Missing security event logs**: Failed auth attempts, permission denials, input validation
      failures — are they logged for monitoring?
- [ ] **Missing audit trail**: Sensitive operations (delete, role change, data export) — is there
      an audit log?
- [ ] **Error message leakage**: Do error messages reveal internal structure, file paths,
      database schema, or stack traces?

## Report Format

For each finding, report:

**[CRITICAL/IMPORTANT/MINOR] — [Category] — [Short description]**
- File: `path/to/file:line_number`
- Problem: [What's wrong — describe the attack vector clearly]
- Recommendation: [Specific remediation with code snippet]
- Impact: [What an attacker could do — data breach, privilege escalation, RCE, etc.]
- CWE: [CWE ID, e.g., CWE-89 (SQL Injection), CWE-79 (XSS), CWE-862 (Missing Authorization)]

## Assessment

- **CRITICAL findings MUST be fixed before merge.** No exceptions.
- **IMPORTANT findings SHOULD be fixed.** If not fixed, a Jira issue MUST be opened.
- **MINOR findings** are noted but do not block merge.

Final verdict:
- **APPROVED**: No critical or important security findings
- **APPROVED WITH CONDITIONS**: Important findings that need Jira issues created
- **CHANGES REQUIRED**: Critical security findings that must be fixed before merge
```

## Review Loop

If security review returns critical findings:

1. **Critical findings:** Author MUST fix, re-run tests, then re-request security review
2. **Important findings:** Fix if possible. For any not fixed, create a Jira issue with file paths, line numbers, attack vector, and CWE ID
3. **Minor findings:** Note in review but do not block
4. **Max 3 review cycles.** If critical findings persist after 3 cycles, escalate to user.

After fixes, dispatch another security review to verify critical findings are resolved.

# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest  | Yes       |

## Reporting a Vulnerability

If you discover a security issue, please report it responsibly.

**Do not open a public GitHub issue for security problems.**

Instead, use [GitHub's private vulnerability reporting](https://github.com/quantum-bytes/oclaw/security/advisories/new) to submit your report.

Please include:

- Description of the issue
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

You should receive an acknowledgment within 48 hours. We will work with you to understand and address the issue before any public disclosure.

## Security Considerations

oclaw handles sensitive data including:

- Ed25519 device private keys (`~/.openclaw/identity/device.json`)
- Gateway authentication tokens
- WebSocket connections to the gateway

See the Security section in [README.md](README.md) for details on the hardening measures in place.

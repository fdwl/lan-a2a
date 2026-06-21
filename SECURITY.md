# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability within LanA2A, please send an email to the maintainers. All security vulnerabilities will be promptly addressed.

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to the repository owner.

When reporting, please include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

## Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial assessment**: Within 1 week
- **Fix or mitigation**: Depends on severity, typically within 2 weeks

## Security Best Practices

When using LanA2A in production:

- **LAN Only**: LanA2A is designed for trusted local networks. Do not expose agents to the public internet without additional security measures.
- **No Authentication**: The current protocol does not include authentication. Any device on the LAN can connect.
- **No Encryption**: WebSocket connections are unencrypted (ws://, not wss://). For sensitive data, consider running on an isolated network.
- **File Downloads**: Received files are saved to `~/.lan-agent-bus/downloads/`. Ensure adequate disk space and access controls.

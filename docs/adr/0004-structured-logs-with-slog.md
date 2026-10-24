---
status: accepted
---

# Structured logs with log/slog in the save server

The save server of the WASM image (`images/wasm/server.go`) logged with `log.Printf`.
The agent-readiness criterion `structured_logging` asks for logs a tool can
parse; operators read them with `docker logs`.

## Considered Options

- `log/slog` with the JSON handler on stdout: in the standard library, no
  dependency, one JSON object per line.
- `log/slog` with the text handler: rejected, key=value lines are easier to read
  but harder to filter.
- zap or zerolog: rejected, a dependency for a server of 150 lines.

## Consequences

- Log lines are JSON with stable keys (`msg`, `name`, `err`, `addr`).
- Revisit if the server grows request logging or metrics.

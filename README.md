# Echo Server (echosvr)

A highly configurable, lightweight Mock Echo Server for testing HTTP/WebSocket traffic.
It echoes requests and can be dynamically configured via `config.yml` to return specific status codes, headers, and bodies for specific routes.

## Usage

```bash
# Run locally
make run

# Run via Docker
docker run -p 58080:58080 -p 58081:58081 -v ./config.yml:/app/config.yml ghcr.io/xvlet/echosvr:latest

# Configuration
# Edit config.yml to add your mock routes.
```

By default, it listens on port `58080`.

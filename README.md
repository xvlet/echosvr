<p align="center">
  <img src="https://img.shields.io/badge/echosvr-Mock%20Server-4A90D9?style=for-the-badge&logo=server&logoColor=white" alt="echosvr banner">
</p>

<h1 align="center">⚡ Echosvr — Mock Echo Server</h1>
<h3 align="center">A Highly Configurable, Lightweight Mock Server for HTTP/WebSocket</h3>

<p align="center">
  <b>Simulate latency, mock responses, and test your systems effortlessly.</b><br>
  Write your config. Spin it up. Test with confidence.
</p>

<p align="center">
  <a href="https://github.com/xvlet/echosvr"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version"></a>
  <img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=for-the-badge" alt="Platform">
  <a href="README.ko-KR.md"><img src="https://img.shields.io/badge/Lang-한국어-red?style=for-the-badge" alt="Korean"></a>
</p>

---

## Overview

**Echosvr** is a highly configurable, lightweight Mock Echo Server for testing HTTP and WebSocket traffic.
It echoes requests and can be dynamically configured via `config.yml` to return specific status codes, headers, bodies, and even artificial latency for specific routes.

It was originally built as a companion tool for [VJM (Vegeta-JMeter Engine)](https://github.com/xvlet/vjm) to act as a target backend for load testing, but it can be used for any general-purpose mocking.

### Why Echosvr?

- ✅ **Zero-Config Catch-all Echo**: No need to map every route! By default, it acts as a catch-all for both HTTP and WebSocket, echoing all requests (regardless of path).
- ✅ **Dynamic Routing (Optional)**: Only add routes to `config.yml` when you need to mock specific behaviors like forcing errors (500), simulating latency, or injecting custom headers/bodies.
- ✅ **HTTP & WebSocket Support**: Listen on separate ports for HTTP and WebSocket traffic.
- ✅ **Mock Errors & Latency**: Force specific status codes, headers, and inject artificial delays.
- ✅ **Cross-Platform**: Run locally on any OS or deploy instantly via Docker.
- ✅ **Lightweight**: Fast startup, low resource usage.

---

## Key Features

<table>
<tr><td><b>HTTP Echoing</b></td><td>Defaults to echoing the request body back to the client with a 200 OK status if no specific route is matched.</td></tr>
<tr><td><b>WebSocket Support</b></td><td>Supports echoing messages over WebSockets on configurable paths.</td></tr>
<tr><td><b>Dynamic Responses</b></td><td>Mock custom JSON/Text responses, inject HTTP headers, and force specific status codes.</td></tr>
<tr><td><b>Latency Simulation</b></td><td>Inject artificial delays (<code>delay_ms</code>) to test timeout handling and slow networks.</td></tr>
<tr><td><b>Advanced Logging</b></td><td>Configurable log rotation, file size limits, and log levels.</td></tr>
</table>

---

## Quick Start

### 1. Run via Docker (Recommended)

```bash
# Run via Docker (with default config)
docker run --rm -p 58080:58080 -p 58081:58081 ghcr.io/xvlet/echosvr:latest

# Run via Docker (with custom config override)
docker run --rm -p 58080:58080 -p 58081:58081 -v $(pwd)/config.yml:/app/config.yml ghcr.io/xvlet/echosvr:latest
```

### 2. Run Locally (Source)

```bash
# Clone and run
git clone https://github.com/xvlet/echosvr.git
cd echosvr
make run
```

---

## Installation

You can install **Echosvr** using one of the following methods.

### 1. Quick Install Scripts
The easiest way to install the latest release is by using the provided installation scripts for your operating system.

**macOS / Linux / AIX (Shell)**
```bash
curl -fsSL https://raw.githubusercontent.com/xvlet/echosvr/master/install.sh | sh
```

**Windows (PowerShell)**
```powershell
powershell -ExecutionPolicy Bypass -c "irm https://raw.githubusercontent.com/xvlet/echosvr/master/install.ps1 | iex"
```

### 2. Using Go (go install)
If you have Go installed, you can easily install **Echosvr** via `go install`:
```bash
go install github.com/xvlet/echosvr@latest
```

### 3. Download Pre-built Binary
Download the latest pre-built release from the [Releases page](https://github.com/xvlet/echosvr/releases) and extract it to your `$PATH`.

---

## Configuration

**Echosvr** is configured via `config.yml`. By default, it looks for this file in the current directory.

```yaml
server:
  port: 58080
  
  websocket:
    port: 58081
    paths:
      - "/ws"
    routes:
      - path: "/ws/mock/auth"
        handshake_status_code: 401
      - path: "/ws/mock/welcome"
        initial_message: '{"type":"welcome","message":"Connected!"}'
      - path: "/ws/mock/delay"
        delay_ms: 500
      - path: "/ws/mock/disconnect"
        disconnect_after_msgs: 3
        disconnect_after_sec: 10
      
  logging:
    use: true
    file_name: "logs/echo.log"
    level: "debug"

  routes:
    - path: "/api/mock"
      method: "GET,POST"
      status_code: 200
      delay_ms: 500
      response_headers:
        "X-Mock-Status": "Active"
      response_body: '{"message": "Mock response"}'
    
    - path: "/test/error"
      method: "ANY"
      status_code: 500
      response_body: "Internal Server Error Simulation"
```

### WebSocket Route Options

| Field | Type | Default | Description |
|---|---|---|---|
| `path` | string | — | WebSocket path to match |
| `handshake_status_code` | int | 0 (pass) | Return this HTTP status code instead of upgrading (e.g. 401, 403) |
| `initial_message` | string | — | Message sent to client immediately after connection |
| `delay_ms` | int | 0 | Artificial delay (ms) before echoing each message |
| `disconnect_after_msgs` | int | 0 | Close connection after receiving N messages |
| `disconnect_after_sec` | int | 0 | Close connection after N seconds |

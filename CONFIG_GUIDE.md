# Echosvr Configuration Guide

`echosvr` is a flexible mock testing server that either returns incoming HTTP/WebSocket requests exactly as they are (Echo), or manipulates the response in a user-desired format.
You can configure ports, logging, and custom routing (returning errors, adding delay time, etc.) via the `config.yml` file.

---

## Table of Contents

1. [server — Basic Server Configuration](#1-server--basic-server-configuration)
2. [logging — Log Configuration](#2-logging--log-configuration)
3. [routes — Routing Configuration (Mock API)](#3-routes--routing-configuration-mock-api)

---

## 1. `server` — Basic Server Configuration

Configures the server's operating port, WebSocket port, and transaction tracking.

```yaml
server:
  port: 58080
  websocket:
    port: 58081
    paths:
      - "/ws"
      - "/ws/test"
  transaction_id_header: "tran_gid"
```

| Item | Description |
|---|---|
| `port` | The port number the HTTP echo server will listen on (e.g., `58080`) |
| `websocket.port` | The port number the WebSocket server will listen on. If set differently from the HTTP port, it runs on a separate port. |
| `websocket.paths` | A list of URL paths that allow WebSocket connections |
| `transaction_id_header` | The HTTP header name to use for transaction tracking (e.g., `tran_gid`). If this header exists, it is used as the request ID when printing logs. |

---

## 2. `logging` — Log Configuration

Configures the server's log output method. It supports rolling file logs and automatic deletion using the `zap` logger and `file-rotatelogs`.

```yaml
server:
  logging:
    use: true
    file_name: "logs/echo.log"
    max_size_mb: 100
    max_history: 3
    level: "debug"
    rotation_time: 24
    pattern: "[%D:23][%G:36][%L:5][%C:14,5] %M"
```

| Item | Description |
|---|---|
| `use` | Whether to enable the logger (`true` / `false`) |
| `file_name` | The log file path to output (e.g., `logs/echo.log`). A symlink pointing to the latest log is created upon execution. |
| `max_size_mb` | The maximum size of a single log file (in MB) |
| `max_history` | The maximum number of log files to keep (in days) |
| `level` | The log level to output (`debug`, `info`, `warn`, `error`, etc.) |
| `rotation_time` | Log file rotation period (in hours, `24` = 1 day) |
| `pattern` | Log output format (customizable with date, time, code line, etc.) |

---

## 3. `routes` — Routing Configuration (Mock API)

Defines forced error returns, delay times (Delay), and custom header/body responses for requests entering specified paths. This is extremely useful for performance testing or resilience testing.

```yaml
server:
  routes:
    # 1. Intentional Delay Response Test
    - path: "/api/slow"
      method: "GET"
      delay_ms: 500
      status_code: 200
      response_body: "This response was delayed by 500ms"

    # 2. HTTP 500 Error and Forced Custom Header Return
    - path: "/api/error"
      method: "GET,POST"
      status_code: 500
      response_headers:
        "X-Error-Code": "ERR-999"
      response_body: '{"error": "Internal Server Error Simulation"}'

    # 3. Allow All Methods (ANY) and Return Specific JSON
    - path: "/api/auth/login"
      # Omitting the method allows all HTTP methods.
      status_code: 200
      response_headers:
        "Content-Type": "application/json"
      response_body: '{"status": "success", "token": "mock-token-12345"}'
```

### Route Configuration Field Descriptions

| Field | Description |
|---|---|
| `path` | The URL path to apply Mocking to (Required) |
| `method` | HTTP methods to allow (comma-separated, e.g., `GET`, `POST,PUT`). If not specified or if `ANY` is used, all methods are allowed. |
| `response_headers` | (Optional) A map of custom HTTP headers to return to the client |
| `status_code` | (Optional) The HTTP status code to return (Default: `200`) |
| `response_body` | (Optional) The text or JSON payload to return (If not set, echoes back the client's body exactly as received) |
| `delay_ms` | (Optional) Forced delay time in milliseconds (ms) before responding to the client |

### 💡 Behavior Summary
1. Requests to unknown paths not in the configuration or unspecified methods default to `200 OK`, echoing the request body exactly as is.
2. If `response_body` is set, it returns the configured fixed string/JSON instead of the echo functionality.
3. Using `delay_ms`, you can effectively verify Timeout and Connection Drop scenarios.

---

## How to Run

```bash
# Local Source Build and Run
make all
./build/echosvr

# Docker-based Run
docker run -p 58080:58080 -p 58081:58081 -v ./config.yml:/app/config.yml ghcr.io/xvlet/echosvr
```

<p align="center">
  <img src="https://img.shields.io/badge/echosvr-Mock%20Server-4A90D9?style=for-the-badge&logo=server&logoColor=white" alt="echosvr banner">
</p>

<h1 align="center">⚡ Echosvr — Mock Echo Server</h1>
<h3 align="center">HTTP/WebSocket 통신 테스트를 위한 경량화 Mock 서버</h3>

<p align="center">
  <b>지연 시간(Latency) 시뮬레이션, Mock 응답 생성 등 백엔드 환경을 쉽게 구성하세요.</b><br>
  Write your config. Spin it up. Test with confidence.
</p>

<p align="center">
  <a href="https://github.com/xvlet/echosvr"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version"></a>
  <img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=for-the-badge" alt="Platform">
</p>

---

## 개요

**Echosvr**은 HTTP 및 WebSocket 트래픽 테스트를 위한 설정 가능하고 가벼운 Mock Echo Server입니다.
수신된 요청을 그대로 에코(Echo)하며, `config.yml`을 통해 특정 경로(Route)에 대해 상태 코드, 헤더, 본문(Body), 인위적 지연(Latency) 등을 동적으로 설정하여 응답할 수 있습니다.

원래는 [VJM (Vegeta-JMeter Engine)](https://github.com/xvlet/vjm) 부하 테스트의 대상 백엔드로 사용하기 위해 개발되었으나, 일반적인 Mock 서버 용도로도 폭넓게 활용할 수 있습니다.

### Why Echosvr?

- ✅ **Zero-Config 기본 에코**: 경로를 일일이 지정할 필요가 없습니다! 기본적으로 HTTP 및 WebSocket 트래픽에 대한 캐치올(Catch-all) 역할을 하며, 모든 요청(경로 무관)을 수신된 그대로 에코(Echo)합니다.
- ✅ **동적 라우팅 설정 (선택 사항)**: 오류(500), 지연(Latency) 시뮬레이션, 특정 헤더/Body 모킹 등 특수한 동작이 필요한 경우에만 `config.yml`에 라우팅을 추가하면 됩니다.
- ✅ **HTTP & WebSocket 지원**: HTTP와 WebSocket 트래픽을 각각 독립적인 포트에서 처리
- ✅ **에러 및 지연 시뮬레이션**: 특정 상태 코드 반환, 커스텀 헤더 주입, 인위적인 응답 지연(Latency) 부여
- ✅ **크로스 플랫폼**: 로컬(OS 무관) 또는 Docker를 통해 즉시 배포 및 실행 가능
- ✅ **경량성**: 빠른 시작 시간과 매우 낮은 시스템 리소스 사용량

---

## 주요 기능

<table>
<tr><td><b>HTTP Echoing</b></td><td>별도로 설정된 경로가 없는 경우, 수신한 요청의 Body를 200 OK 상태와 함께 그대로 클라이언트에게 반환합니다.</td></tr>
<tr><td><b>WebSocket 지원</b></td><td>설정된 경로에서 WebSocket 기반 메시지 에코(Echo)를 지원합니다.</td></tr>
<tr><td><b>동적 응답 구성</b></td><td>커스텀 JSON/Text 응답 구성, HTTP 헤더 주입, 상태 코드 강제 설정이 가능합니다.</td></tr>
<tr><td><b>지연(Latency) 시뮬레이션</b></td><td>인위적인 응답 지연(<code>delay_ms</code>)을 주입하여 타임아웃 핸들링 및 느린 네트워크 상황을 테스트할 수 있습니다.</td></tr>
<tr><td><b>고급 로깅 지원</b></td><td>로그 로테이션, 파일 크기 제한, 로그 레벨 지정 등 설정이 가능합니다.</td></tr>
</table>

---

## 빠른 시작

### 1. Docker로 실행 (권장)

```bash
# 기본 설정으로 Docker 실행
docker run --rm -p 58080:58080 -p 58081:58081 ghcr.io/xvlet/echosvr:latest

# 사용자 정의 config.yml 설정을 덮어씌워 Docker 실행
docker run --rm -p 58080:58080 -p 58081:58081 -v $(pwd)/config.yml:/app/config.yml ghcr.io/xvlet/echosvr:latest
```

### 2. 로컬 실행 (Source)

```bash
# 클론 및 실행
git clone https://github.com/xvlet/echosvr.git
cd echosvr
make run
```

---

## 설치

사용자의 환경에 맞춰 아래 방법 중 하나로 설치할 수 있습니다.

### 1. 간편 설치 스크립트
가장 쉽고 빠르게 최신 버전을 설치하는 방법입니다. 아래 명령어를 실행하면 OS와 아키텍처에 맞는 최신 릴리즈를 자동으로 다운로드하고 설치합니다.

**macOS / Linux / AIX (Shell)**
```bash
curl -fsSL https://raw.githubusercontent.com/xvlet/echosvr/master/install.sh | sh
```

**Windows (PowerShell)**
```powershell
powershell -ExecutionPolicy Bypass -c "irm https://raw.githubusercontent.com/xvlet/echosvr/master/install.ps1 | iex"
```

### 2. Go 환경이 설치된 경우 (go install)
Go가 설치된 환경에서는 아래 명령어로 쉽게 설치할 수 있습니다.
```bash
go install github.com/xvlet/echosvr@latest
```

### 3. 사전 빌드된 바이너리 다운로드 (Pre-built Binary)
- [Release 페이지](https://github.com/xvlet/echosvr/releases)에서 최신 바이너리를 다운로드하고 환경변수 `$PATH`에 등록하세요.

---

## 환경 설정 (Configuration)

**Echosvr**은 `config.yml` 파일을 통해 설정할 수 있습니다. 기본적으로 실행 위치(현재 디렉토리)에서 이 파일을 찾습니다.

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

| 필드 | 타입 | 기본값 | 설명 |
|---|---|---|---|
| `path` | string | — | 매칭할 WebSocket 경로 |
| `handshake_status_code` | int | 0 (pass) | WebSocket 업그레이드 대신 반환할 HTTP 상태 코드 (예: 401, 403) |
| `initial_message` | string | — | 연결 직후 클라이언트에게 전송할 초기 메시지 |
| `delay_ms` | int | 0 | 각 메시지 에코 전 인위적인 지연 시간 (ms) |
| `disconnect_after_msgs` | int | 0 | 지정된 N개의 메시지를 수신한 후 연결 종료 |
| `disconnect_after_sec` | int | 0 | 연결 후 N초가 지나면 연결 종료 |

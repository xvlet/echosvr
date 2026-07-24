# Echosvr 설정 가이드

`echosvr`는 수신된 HTTP/WebSocket 요청을 그대로 응답(Echo)으로 돌려주거나, 사용자가 원하는 형태로 응답을 조작할 수 있는 유연한 모의(Mock) 테스트 서버입니다.
`config.yml` 파일을 통해 포트, 로깅, 커스텀 라우팅(에러 반환, 지연 시간 추가 등)을 설정할 수 있습니다.

---

## 목차

1. [server — 서버 기본 설정](#1-server--서버-기본-설정)
2. [logging — 로그 설정](#2-logging--로그-설정)
3. [routes — 라우팅 설정 (Mock API)](#3-routes--라우팅-설정-mock-api)

---

## 1. `server` — 서버 기본 설정

서버의 작동 포트, WebSocket 포트, 트랜잭션 추적 등을 설정합니다.

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

| 항목 | 설명 |
|---|---|
| `port` | HTTP 에코 서버가 수신 대기할 포트 번호 (예: `58080`) |
| `websocket.port` | WebSocket 서버가 수신 대기할 포트 번호. HTTP 포트와 다르게 설정하면 별도 포트로 구동됩니다. |
| `websocket.paths` | WebSocket 연결을 허용할 URL 경로 목록 |
| `transaction_id_header` | 트랜잭션 추적에 사용할 HTTP 헤더명(예: `tran_gid`). 해당 헤더가 있으면 로그 출력 시 요청 ID로 사용합니다. |

---

## 2. `logging` — 로그 설정

서버의 로그 출력 방식을 설정합니다. `zap` 로거와 `file-rotatelogs`를 사용하여 롤링 파일 로그 및 자동 삭제 기능을 지원합니다.

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

| 항목 | 설명 |
|---|---|
| `use` | 로거 활성화 여부 (`true` / `false`) |
| `file_name` | 출력할 로그 파일 경로 (예: `logs/echo.log`). 실행 시 최신 로그를 가리키는 symlink가 생성됩니다. |
| `max_size_mb` | 단일 로그 파일의 최대 크기 (MB) |
| `max_history` | 보관할 로그 파일의 최대 개수 (일 단위) |
| `level` | 출력할 로그 레벨 (`debug`, `info`, `warn`, `error` 등) |
| `rotation_time` | 로그 파일 로테이션 주기 (시간 단위, `24` = 1일) |
| `pattern` | 로그 출력 포맷 (날짜, 시간, 코드 라인 등 커스터마이징 가능) |

---

## 3. `routes` — 라우팅 설정 (Mock API)

지정된 경로(Path)로 들어오는 요청에 대해 강제 에러 반환, 지연 시간(Delay), 커스텀 헤더 및 바디 응답 등을 정의합니다. 성능 테스트나 장애 복원력 테스트(Resilience Test) 시 매우 유용합니다.

```yaml
server:
  routes:
    # 1. 고의적인 지연(Latency) 응답 테스트
    - path: "/api/slow"
      method: "GET"
      delay_ms: 500
      status_code: 200
      response_body: "This response was delayed by 500ms"

    # 2. HTTP 500 에러 및 커스텀 헤더 강제 반환
    - path: "/api/error"
      method: "GET,POST"
      status_code: 500
      response_headers:
        "X-Error-Code": "ERR-999"
      response_body: '{"error": "Internal Server Error Simulation"}'

    # 3. 모든 메서드(ANY) 허용 및 특정 JSON 반환
    - path: "/api/auth/login"
      # method 생략 시 모든 HTTP 메서드를 허용합니다.
      status_code: 200
      response_headers:
        "Content-Type": "application/json"
      response_body: '{"status": "success", "token": "mock-token-12345"}'
```

### 라우트 설정 필드 설명

| 필드 | 설명 |
|---|---|
| `path` | Mocking을 적용할 URL 경로 (필수) |
| `method` | 허용할 HTTP 메서드 (`GET`, `POST,PUT` 등 콤마 구분). 지정하지 않거나 `ANY` 사용 시 모든 메서드 허용 |
| `response_headers` | (선택) 클라이언트에게 반환할 커스텀 HTTP 헤더 맵(Map) |
| `status_code` | (선택) 반환할 HTTP 상태 코드 (기본값: `200`) |
| `response_body` | (선택) 반환할 텍스트 또는 JSON 페이로드 (설정하지 않으면 클라이언트가 보낸 Body를 그대로 Echo) |
| `delay_ms` | (선택) 클라이언트에게 응답하기 전에 강제로 지연시킬 밀리초(ms) 시간 |

### 💡 동작 방식 요약
1. 설정에 없는 알 수 없는 경로나 설정되지 않은 메서드로 요청 시 기본적으로 요청 바디를 그대로 Echo 하는 `200 OK`로 동작합니다.
2. `response_body`를 설정하면 에코(Echo) 기능 대신 설정된 고정 문자열/JSON을 반환합니다.
3. `delay_ms`를 이용해 Timeout 및 Connection Drop 시나리오를 효과적으로 검증할 수 있습니다.

---

## 실행 방법

```bash
# 로컬 소스 빌드 및 실행
make all
./build/echosvr

# 도커(Docker) 기반 실행
docker run -p 58080:58080 -p 58081:58081 -v ./config.yml:/app/config.yml ghcr.io/xvlet/echosvr
```

package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gopkg.in/yaml.v3"

	"github.com/xvlet/echosvr/custom"
	"github.com/xvlet/echosvr/pkg/logger"

	"golang.org/x/net/websocket"
)

type RouteConfig struct {
	Path            string            `yaml:"path"`
	Method          string            `yaml:"method"`
	ResponseHeaders map[string]string `yaml:"response_headers"`
	StatusCode      int               `yaml:"status_code"`
	ResponseBody    string            `yaml:"response_body"`
	DelayMs         int               `yaml:"delay_ms"`
}

type WebsocketConfig struct {
	Port  int      `yaml:"port"`
	Paths []string `yaml:"paths"`
}

type Config struct {
	Server struct {
		Port                int             `yaml:"port"`
		Websocket           WebsocketConfig `yaml:"websocket"`
		TransactionIDHeader string          `yaml:"transaction_id_header"`
		Routes              []RouteConfig   `yaml:"routes"`
		Logging             logger.Config   `yaml:"logging"`
	} `yaml:"server"`
}

func main() {
	// Load config
	config := Config{}
	configFile, err := os.ReadFile("config.yml")
	if err != nil {
		fmt.Printf("Warning: failed to read config.yml, using default port 58080 and routes: %v\n", err)
		config.Server.Port = 58080
	} else {
		err = yaml.Unmarshal(configFile, &config)
		if err != nil {
			fmt.Printf("Warning: failed to parse config.yml, using default port 58080: %v\n", err)
			config.Server.Port = 58080
		}
	}

	// Initialize Logger
	err = logger.InitLogger(config.Server.Logging)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
	}
	defer logger.Sync()

	var startupReqID string
	b := make([]byte, 16)
	if _, err := rand.Read(b); err == nil {
		b[6] = (b[6] & 0x0f) | 0x40
		b[8] = (b[8] & 0x3f) | 0x80
		startupReqID = fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	} else {
		startupReqID = "startup"
	}

	if logger.S != nil {
		logger.WithReqID(startupReqID).Infof("\x1b[38;5;45m▶ HTTP Server\x1b[0m")
		logger.WithReqID(startupReqID).Infof("Loaded configuration: port=%d, routes=%d", config.Server.Port, len(config.Server.Routes))
	} else {
		fmt.Println("\x1b[38;5;45m▶ HTTP Server\x1b[0m")
	}

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			b := make([]byte, 16)
			_, _ = rand.Read(b)
			b[6] = (b[6] & 0x0f) | 0x40
			b[8] = (b[8] & 0x3f) | 0x80
			return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
		},
	}))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogURI:    true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			reqId := getReqID(c, config.Server.TransactionIDHeader)
			log := logger.WithReqID(reqId)
			if log != nil {
				log.Debugf("[ECHO] %-7s [REQUEST]     uri=%s, status=%d", fmt.Sprintf("[%s]", c.Request().Method), v.URI, v.Status)
			} else if os.Getenv("DEBUG") == "Y" {
				fmt.Printf("REQUEST: uri=%s, status=%d\n", v.URI, v.Status)
			}
			return nil
		},
	}))
	e.Use(middleware.Recover())

	if len(config.Server.Routes) > 0 {
		for _, r := range config.Server.Routes {
			route := r // localized copy for closure
			if logger.S != nil {
				logger.WithReqID(startupReqID).Infof("Registering route: [%s] %s", route.Method, route.Path)
			} else {
				fmt.Printf("Registering route: [%s] %s\n", route.Method, route.Path)
			}

			methods := strings.Split(route.Method, ",")
			for _, m := range methods {
				m = strings.TrimSpace(m)

				// Intercept for custom handlers
				if h := custom.GetCustomHandler(m, route.Path); h != nil {
					e.Add(m, route.Path, h)
					continue
				}

				// Dynamic Routes from Config
				switch m {
				case "", "ANY", "*":
					e.Any(route.Path, func(c echo.Context) error {
						return handleAny(c, config.Server.TransactionIDHeader, route)
					})
				default:
					e.Add(m, route.Path, func(c echo.Context) error {
						return handleAny(c, config.Server.TransactionIDHeader, route)
					})
				}
			}
		}
	}

	// Always provide a catch-all fallback for undefined routes
	if logger.S != nil {
		logger.WithReqID(startupReqID).Infof("Registering wildcard route: \x1b[93m[ANY] /* (Catch-all)\x1b[0m")
	} else {
		fmt.Println("Registering wildcard route: \x1b[93m[ANY] /* (Catch-all)\x1b[0m")
	}
	e.Any("/*", func(c echo.Context) error {
		return handleAny(c, config.Server.TransactionIDHeader, RouteConfig{})
	})

	// WebSocket Echo Route
	if logger.S != nil {
		logger.WithReqID(startupReqID).Infof("")
		logger.WithReqID(startupReqID).Infof("\x1b[38;5;45m▶ WebSocket Server\x1b[0m")
	} else {
		fmt.Println()
		fmt.Println("\x1b[38;5;45m▶ WebSocket Server\x1b[0m")
	}
	wsPaths := config.Server.Websocket.Paths
	wsServer := websocket.Server{
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			for {
				var msg string
				if err := websocket.Message.Receive(ws, &msg); err != nil {
					fmt.Printf("[WS] Disconnected or error: %v\n", err)
					break
				}
				fmt.Printf("[WS] Received payload (len %d): %q\n", len(msg), msg)
				if err := websocket.Message.Send(ws, msg); err != nil {
					break
				}
			}
		}),
		Handshake: func(config *websocket.Config, req *http.Request) error {
			// Accept all origins to prevent 403 Forbidden on cross-origin/remote requests
			return nil
		},
	}
	wsHandler := echo.WrapHandler(wsServer)
	if len(wsPaths) == 0 {
		wsPaths = []string{"/ws"} // default fallback
	}
	wsPort := config.Server.Websocket.Port

	if wsPort > 0 && wsPort != config.Server.Port {
		// Run WebSocket on a separate port
		wsEcho := echo.New()
		wsEcho.HideBanner = true
		wsEcho.HidePort = true
		if logger.S != nil {
			logger.WithReqID(startupReqID).Infof("Loaded configuration: port=%d, routes=%d", wsPort, len(wsPaths))
		}
		for _, wsPath := range wsPaths {
			if logger.S != nil {
				logger.WithReqID(startupReqID).Infof("Registering route: [%s]", wsPath)
			}
			wsEcho.GET(wsPath, wsHandler)
		}
		go func() {
			// Output directly with the exact same format and color (ANSI Green) as the Echo framework
			fmt.Printf("⇨ websocket server started on \x1b[32m[::]:%d\x1b[0m\n", wsPort)

			lc := net.ListenConfig{
				Control: func(network, address string, c syscall.RawConn) error {
					var err error
					_ = c.Control(func(fd uintptr) {
						err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
						if err != nil {
							return
						}
						err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, 15, 1) // SO_REUSEPORT
					})
					return err
				},
			}
			l, err := lc.Listen(context.Background(), "tcp", fmt.Sprintf(":%d", wsPort))
			if err != nil {
				wsEcho.Logger.Fatal(err)
			}
			wsEcho.Listener = l
			wsEcho.Logger.Fatal(wsEcho.StartServer(wsEcho.Server))
		}()
	} else {
		// Run WebSocket on the same HTTP port
		if logger.S != nil {
			logger.WithReqID(startupReqID).Infof("Loaded configuration: port=%d, routes=%d", config.Server.Port, len(wsPaths))
		} else {
			fmt.Printf("Loaded configuration: port=%d, routes=%d\n", config.Server.Port, len(wsPaths))
		}
		for _, wsPath := range wsPaths {
			if logger.S != nil {
				logger.WithReqID(startupReqID).Infof("Registering route: [%s]", wsPath)
			} else {
				fmt.Printf("Registering route: [%s]\n", wsPath)
			}
			e.GET(wsPath, wsHandler)
		}
	}

	// Start server
	address := fmt.Sprintf(":%d", config.Server.Port)
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var err error
			_ = c.Control(func(fd uintptr) {
				err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				if err != nil {
					return
				}
				err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, 15, 1) // SO_REUSEPORT
			})
			return err
		},
	}
	l, err := lc.Listen(context.Background(), "tcp", address)
	if err != nil {
		e.Logger.Fatal(err)
	}
	e.Listener = l
	e.Logger.Fatal(e.StartServer(e.Server))
}

// reflectHeaders copies request headers to response headers, excluding hop-by-hop ones.
func reflectHeaders(c echo.Context, transactionIDHeader string) {
	// Standard hop-by-hop and protocol-managed headers to avoid mirroring
	restricted := map[string]bool{
		"Content-Length":      true,
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
		"Host":                true,
	}

	isDebug := os.Getenv("DEBUG") == "Y"
	reqId := getReqID(c, transactionIDHeader)
	log := logger.WithReqID(reqId)

	for k, v := range c.Request().Header {
		if _, isRestricted := restricted[k]; !isRestricted {
			for _, val := range v {
				if len(val) > 0 {
					c.Response().Header().Add(k, val)
					if log != nil {
						log.Debugf("[ECHO] %-7s [Header]      %s: [%s]", fmt.Sprintf("[%s]", c.Request().Method), k, val)
					} else if isDebug {
						fmt.Printf("[MOCK-ECHO-SERVER] Reflected Header: %s=%s\n", k, val)
					}
				}
			}
		}
	}
}

// handleAny echoes back the request dynamically based on the method
func handleAny(c echo.Context, transactionIDHeader string, route RouteConfig) error {
	isDebug := os.Getenv("DEBUG") == "Y"
	req := c.Request()
	method := req.Method
	query := req.URL.RawQuery
	reqId := getReqID(c, transactionIDHeader)
	log := logger.WithReqID(reqId)

	// Simulate delay if configured
	if route.DelayMs > 0 {
		if log != nil {
			log.Debugf("[ECHO] %-7s Sleeping for %d ms", fmt.Sprintf("[%s]", method), route.DelayMs)
		} else if isDebug {
			fmt.Printf("[MOCK-ECHO-SERVER] %-7s Sleeping for %d ms\n", fmt.Sprintf("[%s]", method), route.DelayMs)
		}
		time.Sleep(time.Duration(route.DelayMs) * time.Millisecond)
	}

	// Log and reflect headers
	reflectHeaders(c, transactionIDHeader)

	// Apply custom response headers from config
	for k, v := range route.ResponseHeaders {
		c.Response().Header().Set(k, v)
		if log != nil {
			log.Debugf("[ECHO] %-7s [Header]      %s: [%s] (Configured)", fmt.Sprintf("[%s]", method), k, v)
		} else if isDebug {
			fmt.Printf("[MOCK-ECHO-SERVER] %-7s Added Config Header: %s=%s\n", fmt.Sprintf("[%s]", method), k, v)
		}
	}

	if log != nil {
		if query != "" {
			log.Debugf("[ECHO] %-7s [QueryString] [%s]", fmt.Sprintf("[%s]", method), query)
		}
	} else if isDebug && query != "" {
		fmt.Printf("[MOCK-ECHO-SERVER] %-7s [QueryString] [%s]\n", fmt.Sprintf("[%s]", method), query)
	}

	var body []byte
	var err error
	// Limit request body to 10MB to prevent OOM
	body, err = io.ReadAll(io.LimitReader(req.Body, 10*1024*1024))
	if err == nil {
		if len(body) > 0 {
			if log != nil {
				log.Debugf("[ECHO] %-7s [Body]        [%s]", fmt.Sprintf("[%s]", method), string(body))
			} else if isDebug {
				fmt.Printf("[MOCK-ECHO-SERVER] %-7s [Body]        [%s]\n", fmt.Sprintf("[%s]", method), string(body))
			}
		}
	} else {
		if log != nil {
			log.Errorf("[ECHO] %-7s Error reading body: %v", fmt.Sprintf("[%s]", method), err)
		} else if isDebug {
			fmt.Printf("[MOCK-ECHO-SERVER] %-7s Error reading body: %v\n", fmt.Sprintf("[%s]", method), err)
		}
	}

	msg := c.Param("*")
	if msg == "" {
		msg = "N/A"
	}

	statusCode := http.StatusOK
	if route.StatusCode != 0 {
		statusCode = route.StatusCode
	}

	responseBody := string(body)
	if responseBody == "" && (method == "GET" || method == "DELETE") {
		if query != "" {
			decodedQuery, err := url.QueryUnescape(query)
			if err == nil {
				responseBody = decodedQuery
			} else {
				responseBody = query
			}
		} else {
			responseBody = msg
		}
	}
	if route.ResponseBody != "" {
		responseBody = route.ResponseBody
	}

	// If custom headers don't define Content-Type, echo sets it to text/plain by default for c.String()
	// Let's use c.Blob to respect user's headers or echo's defaults
	return c.Blob(statusCode, c.Response().Header().Get(echo.HeaderContentType), []byte(responseBody))
}

// getReqID extracts the request ID, preferring the configured transaction ID header
func getReqID(c echo.Context, transactionIDHeader string) string {
	if transactionIDHeader != "" {
		for k, v := range c.Request().Header {
			if strings.EqualFold(k, transactionIDHeader) && len(v) > 0 && v[0] != "" {
				return v[0]
			}
		}
	}
	reqId := c.Response().Header().Get(echo.HeaderXRequestID)
	if reqId == "" {
		reqId = c.Request().Header.Get(echo.HeaderXRequestID)
	}
	return reqId
}

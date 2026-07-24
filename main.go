package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
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

	if logger.S != nil {
		logger.S.Infof("Loaded configuration: port=%d, routes=%d", config.Server.Port, len(config.Server.Routes))
	}

	// Create Echo instance
	e := echo.New()

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

	// Dynamic Routes from Config
	if len(config.Server.Routes) > 0 {
		for _, r := range config.Server.Routes {
			route := r // localized copy for closure
			if logger.S != nil {
				logger.S.Infof("Registering route: [%s] %s (ResponseHeaders: %v)", route.Method, route.Path, route.ResponseHeaders)
			} else {
				fmt.Printf("Registering route: [%s] %s (ResponseHeaders: %v)\n", route.Method, route.Path, route.ResponseHeaders)
			}

			methods := strings.Split(route.Method, ",")
			for _, m := range methods {
				m = strings.TrimSpace(m)

				// Intercept for custom handlers
				if route.Path == "/test/api/login" && m == "POST" {
					e.POST(route.Path, custom.LoginHandler)
					continue
				}
				if route.Path == "/test/api/profile" && m == "GET" {
					e.GET(route.Path, custom.ProfileHandler)
					continue
				}

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
	e.Any("/*", func(c echo.Context) error {
		return handleAny(c, config.Server.TransactionIDHeader, RouteConfig{})
	})

	// WebSocket Echo Route
	wsPaths := config.Server.Websocket.Paths
	wsHandler := echo.WrapHandler(websocket.Handler(func(ws *websocket.Conn) {
		_, _ = io.Copy(ws, ws)
	}))
	if len(wsPaths) == 0 {
		wsPaths = []string{"/ws"} // default fallback
	}
	wsPort := config.Server.Websocket.Port

	if wsPort > 0 && wsPort != config.Server.Port {
		// Run WebSocket on a separate port
		wsEcho := echo.New()
		wsEcho.HideBanner = true
		wsEcho.HidePort = true
		for _, wsPath := range wsPaths {
			wsEcho.GET(wsPath, wsHandler)
		}
		go func() {
			if logger.S != nil {
				logger.S.Infof("Starting WebSocket server on port: %d, paths: %v", wsPort, wsPaths)
			}

			// Output directly with the exact same format and color (ANSI Green) as the Echo framework
			fmt.Printf("⇨ websocket server started on \x1b[32m[::]:%d\x1b[0m\n", wsPort)

			wsEcho.Logger.Fatal(wsEcho.Start(fmt.Sprintf(":%d", wsPort)))
		}()
	} else {
		// Run WebSocket on the same HTTP port
		if logger.S != nil {
			logger.S.Infof("Starting WebSocket server on HTTP port: %d, paths: %v", config.Server.Port, wsPaths)
		} else {
			fmt.Printf("Starting WebSocket server on HTTP port: %d, paths: %v\n", config.Server.Port, wsPaths)
		}
		for _, wsPath := range wsPaths {
			e.GET(wsPath, wsHandler)
		}
	}

	// Start server
	address := fmt.Sprintf(":%d", config.Server.Port)
	e.Logger.Fatal(e.Start(address))
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

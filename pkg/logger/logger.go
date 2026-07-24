package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	L *zap.Logger
	S *zap.SugaredLogger
)

type Config struct {
	Use          bool   `yaml:"use"`
	FileName     string `yaml:"file_name"`
	MaxSizeMB    int    `yaml:"max_size_mb"`
	MaxHistory   int    `yaml:"max_history"`
	Level        string `yaml:"level"`
	RotationTime int    `yaml:"rotation_time"`
	Pattern      string `yaml:"pattern"`
}

func InitLogger(cfg Config) error {
	if !cfg.Use || strings.ToLower(cfg.Level) == "off" {
		L = zap.NewNop()
		S = L.Sugar()
		return nil
	}

	// Ensure log directory exists
	logDir := filepath.Dir(cfg.FileName)
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Parse level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zap.InfoLevel
	}

	// Rotate logs config
	maxAge := time.Duration(cfg.MaxHistory) * 24 * time.Hour
	if maxAge == 0 {
		maxAge = 7 * 24 * time.Hour
	}
	rotationTime := time.Duration(cfg.RotationTime) * time.Hour
	if rotationTime == 0 {
		rotationTime = 24 * time.Hour
	}

	ext := filepath.Ext(cfg.FileName)
	baseName := strings.TrimSuffix(cfg.FileName, ext)
	fileNamePattern := baseName + "_%Y-%m-%d" + ext

	opts := []rotatelogs.Option{
		rotatelogs.WithMaxAge(maxAge),
		rotatelogs.WithRotationTime(rotationTime),
		rotatelogs.WithRotationSize(int64(cfg.MaxSizeMB) * 1024 * 1024),
		rotatelogs.WithLinkName(cfg.FileName),
	}

	rl, err := rotatelogs.New(fileNamePattern, opts...)
	if err != nil {
		return fmt.Errorf("failed to create rotatelogs: %w", err)
	}

	// Also output to console
	consoleSyncer := zapcore.AddSync(os.Stdout)
	fileSyncer := zapcore.AddSync(rl)

	patternStr := cfg.Pattern
	if patternStr == "" {
		patternStr = "[%D:23][%G:36][%L:5][%C:24,5] %M"
	}
	parsedPattern := parsePattern(patternStr)

	core := zapcore.NewTee(
		&customCore{LevelEnabler: level, out: consoleSyncer, pattern: parsedPattern},
		&customCore{LevelEnabler: level, out: fileSyncer, pattern: parsedPattern},
	)

	L = zap.New(core, zap.AddCaller())
	S = L.Sugar()

	return nil
}

func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}

func WithReqID(reqId string) *zap.SugaredLogger {
	if L == nil {
		return nil
	}
	return L.With(zap.String("reqId", reqId)).Sugar()
}

type customCore struct {
	zapcore.LevelEnabler
	out     zapcore.WriteSyncer
	fields  map[string]string
	pattern []patternNode
}

func (c *customCore) With(fields []zapcore.Field) zapcore.Core {
	newFields := make(map[string]string)
	for k, v := range c.fields {
		newFields[k] = v
	}
	for _, f := range fields {
		if f.Type == zapcore.StringType {
			newFields[f.Key] = f.String
		}
	}
	return &customCore{
		LevelEnabler: c.LevelEnabler,
		out:          c.out,
		fields:       newFields,
		pattern:      c.pattern,
	}
}

func (c *customCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *customCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	reqID := c.fields["reqId"]

	var sb strings.Builder
	for _, node := range c.pattern {
		if !node.isPlaceholder {
			sb.WriteString(node.value)
			continue
		}

		switch node.value {
		case "D":
			date := ent.Time.Format("2006-01-02T15:04:05.000+09:00")
			if node.width1 > 0 && len(date) > node.width1 {
				date = date[:node.width1]
			}
			sb.WriteString(date)
		case "L":
			level := ent.Level.CapitalString()
			if node.width1 > 0 {
				format := fmt.Sprintf("%%-%ds", node.width1)
				level = fmt.Sprintf(format, level)
				if len(level) > node.width1 {
					level = level[:node.width1]
				}
			}
			sb.WriteString(level)
		case "C":
			callerStr := ""
			if node.width1 > 0 || node.width2 > 0 {
				callerStr = strings.Repeat(" ", node.width1+1+node.width2)
			}
			if ent.Caller.Defined {
				file := filepath.Base(ent.Caller.File)
				name := strings.TrimSuffix(file, filepath.Ext(file))
				if node.width1 > 0 && len(name) > node.width1 {
					name = name[:node.width1]
				}
				if node.width1 > 0 && node.width2 > 0 {
					format := fmt.Sprintf("%%-%ds:%%5d", node.width1)
					callerStr = fmt.Sprintf(format, name, ent.Caller.Line)
				} else if node.width1 > 0 {
					format := fmt.Sprintf("%%-%ds", node.width1)
					callerStr = fmt.Sprintf(format, name)
				}
			}
			sb.WriteString(callerStr)
		case "G":
			gidStr := reqID
			if node.width1 > 0 {
				if len(gidStr) > node.width1 {
					gidStr = gidStr[:node.width1]
				}
				format := fmt.Sprintf("%%-%ds", node.width1)
				gidStr = fmt.Sprintf(format, gidStr)
			}
			sb.WriteString(gidStr)
		case "M":
			sb.WriteString(ent.Message)
		}
	}
	sb.WriteString("\n")
	_, err := c.out.Write([]byte(sb.String()))
	return err
}

func (c *customCore) Sync() error {
	return c.out.Sync()
}

type patternNode struct {
	isPlaceholder bool
	value         string
	width1        int
	width2        int
}

func parsePattern(s string) []patternNode {
	var nodes []patternNode
	i := 0
	for i < len(s) {
		if s[i] == '%' && i+1 < len(s) {
			node := patternNode{isPlaceholder: true}
			char := s[i+1]
			node.value = string(char)
			i += 2
			if i < len(s) && s[i] == ':' {
				i++
				start := i
				for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == ',') {
					i++
				}
				parts := strings.Split(s[start:i], ",")
				if len(parts) > 0 {
					_, _ = fmt.Sscanf(parts[0], "%d", &node.width1)
				}
				if len(parts) > 1 {
					_, _ = fmt.Sscanf(parts[1], "%d", &node.width2)
				}
			}
			nodes = append(nodes, node)
		} else {
			start := i
			for i < len(s) && s[i] != '%' {
				i++
			}
			nodes = append(nodes, patternNode{isPlaceholder: false, value: s[start:i]})
		}
	}
	return nodes
}

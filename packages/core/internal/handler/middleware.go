package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func getLogLevel(statusCode int) zapcore.Level {
	switch {
	case statusCode >= 500:
		return zapcore.ErrorLevel
	case statusCode >= 400:
		return zapcore.WarnLevel
	case statusCode >= 300:
		return zapcore.InfoLevel
	default:
		return zapcore.DebugLevel
	}
}

func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Read the request body
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
		}

		// Restore the request body for future use
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Create a response writer wrapper to capture the response status
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		duration := time.Since(start)
		status := blw.Status()
		logLevel := getLogLevel(status)

		logFunc := logger.Debug
		switch logLevel {
		case zapcore.ErrorLevel:
			logFunc = logger.Error
		case zapcore.WarnLevel:
			logFunc = logger.Warn
		case zapcore.InfoLevel:
			logFunc = logger.Info
		}

		// json parse
		requestBodyField := cleanJSONRaw(string(bodyBytes))
		responseBodyField := cleanJSONRaw(blw.body.String())

		// 어떤 항목을 로그 메시지에 포함시킬지 결정
		logFields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", duration),
		}
		logFields = append(logFields, zap.Any("request_body", requestBodyField.Interface))
		logFields = append(logFields, zap.Any("response_body", responseBodyField.Interface))

		logFunc("HTTP Request", logFields...)
	}
}

// bodyLogWriter is a custom response writer that captures the response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func (w *bodyLogWriter) Status() int {
	return w.ResponseWriter.Status()
}

func cleanJSONRaw(s string) zap.Field {
	var data json.RawMessage
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		// JSON 파싱에 실패하면 원본 문자열을 일반 문자열로 반환
		return zap.String("body", s)
	}

	return zap.Any("body", json.RawMessage(data))
}

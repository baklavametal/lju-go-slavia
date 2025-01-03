package middlewares

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/baklavametal/lju-go-slavia/pkg/constants"
	"github.com/baklavametal/lju-go-slavia/pkg/logging"
)

var (
	HiddenRequestHeaders = map[string]struct{}{
		"authorization": {},
		"cookie":        {},
		"set-cookie":    {},
		"x-auth-token":  {},
		"x-csrf-token":  {},
		"x-xsrf-token":  {},
	}
	HiddenResponseHeaders = map[string]struct{}{
		"set-cookie": {},
	}

	RequestBodyMaxSize  = 64 * 1024 // 64KB
	ResponseBodyMaxSize = 64 * 1024 // 64KB
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger, _ := logging.Get(c)
		level := slog.LevelInfo

		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		host := c.Request.Host
		route := c.FullPath()
		ip := c.ClientIP()
		referer := c.Request.Referer()
		userAgent := c.Request.UserAgent()

		params := map[string]string{}
		for _, p := range c.Params {
			params[p.Key] = p.Value
		}

		br := newBodyReader(c.Request.Body, RequestBodyMaxSize, true)
		c.Request.Body = br
		bw := newBodyWriter(c.Writer, ResponseBodyMaxSize, true)
		c.Writer = bw

		requestID := c.GetHeader(string(constants.HeaderRequestIDKey))
		if requestID == "" {
			requestID = uuid.New().String()
			c.Header(string(constants.HeaderRequestIDKey), requestID)
		}

		requestAttributes := []any{
			slog.Time("time", start.UTC()),
			slog.String("path", path),
			slog.String("query", query),
			slog.String("method", method),
			slog.String("host", host),
			slog.String("route", route),
			slog.String("ip", ip),
			slog.String("referer", referer),
			slog.String("user-agent", userAgent),
			slog.Any("params", params),
			slog.Any("headers", extractRequestHeaders(c)),
			slog.Any("body", br.body.String()),
		}

		attributes := []slog.Attr{
			slog.String(string(constants.LogRequestIDKey), requestID),
			slog.Group("request", requestAttributes...),
		}

		logger.LogAttrs(
			c.Request.Context(),
			level,
			"Incoming request",
			attributes...,
		)

		logger = logger.With(slog.String("req_id", requestID))
		c.Set(string(constants.ContextLoggerKey), logger)

		c.Next()

		status := c.Writer.Status()
		end := time.Now()
		latency := end.Sub(start)

		responseAttributes := []any{
			slog.Time("time", end.UTC()),
			slog.Duration("latency", latency),
			slog.Int("status", status),
			slog.Any("headers", extractResponseHeaders(c)),
			slog.Any("body", bw.body.String()),
		}

		attributes = []slog.Attr{
			slog.Group("response", responseAttributes...),
		}

		msg := "Request completed"

		if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
			level = slog.LevelWarn
			msg = fmt.Sprintf("Request failed with error: %s", c.Errors.String())
		} else if status >= http.StatusInternalServerError {
			level = slog.LevelError
			msg = fmt.Sprintf("Request failed with error: %s", c.Errors.String())
		}

		logger.LogAttrs(
			c.Request.Context(),
			level,
			msg,
			attributes...,
		)
	}
}

func extractRequestHeaders(c *gin.Context) []any {
	headers := []any{}
	for k, v := range c.Request.Header {
		if _, found := HiddenRequestHeaders[strings.ToLower(k)]; found {
			continue
		}
		headers = append(headers, slog.Any(k, v))
	}
	return headers
}

func extractResponseHeaders(c *gin.Context) []any {
	headers := []any{}
	for k, v := range c.Writer.Header() {
		if _, found := HiddenResponseHeaders[strings.ToLower(k)]; found {
			continue
		}
		headers = append(headers, slog.Any(k, v))
	}
	return headers
}

var _ http.ResponseWriter = (*bodyWriter)(nil)
var _ http.Flusher = (*bodyWriter)(nil)
var _ http.Hijacker = (*bodyWriter)(nil)

type bodyWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	maxSize int
	bytes   int
}

// implements gin.ResponseWriter
func (w *bodyWriter) Write(b []byte) (int, error) {
	if w.body != nil {
		if w.body.Len()+len(b) > w.maxSize {
			w.body.Write(b[:w.maxSize-w.body.Len()])
		} else {
			w.body.Write(b)
		}
	}

	w.bytes += len(b) //nolint:staticcheck
	return w.ResponseWriter.Write(b)
}

func newBodyWriter(writer gin.ResponseWriter, maxSize int, recordBody bool) *bodyWriter {
	var body *bytes.Buffer
	if recordBody {
		body = bytes.NewBufferString("")
	}

	return &bodyWriter{
		ResponseWriter: writer,
		body:           body,
		maxSize:        maxSize,
		bytes:          0,
	}
}

type bodyReader struct {
	io.ReadCloser
	body    *bytes.Buffer
	maxSize int
	bytes   int
}

// implements io.Reader
func (r *bodyReader) Read(b []byte) (int, error) {
	n, err := r.ReadCloser.Read(b)
	if r.body != nil {
		if r.body.Len()+n > r.maxSize {
			r.body.Write(b[:r.maxSize-r.body.Len()])
		} else {
			r.body.Write(b[:n])
		}
	}
	r.bytes += n
	return n, err
}

func newBodyReader(reader io.ReadCloser, maxSize int, recordBody bool) *bodyReader {
	var body *bytes.Buffer
	if recordBody {
		body = bytes.NewBufferString("")
	}

	return &bodyReader{
		ReadCloser: reader,
		body:       body,
		maxSize:    maxSize,
		bytes:      0,
	}
}

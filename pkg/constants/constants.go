package constants

// HeaderKey is a key for a header
type HeaderKey string

const (
	HeaderRequestIDKey HeaderKey = "X-Request-Id"
)

// LogKey is a key for a log
type LogKey string

const (
	LogRequestIDKey LogKey = "req_id"
)

// ContextKey is a key for context
type ContextKey string

const (
	ContextLoggerKey    ContextKey = "lju-logger"
	ContextRequestIDKey ContextKey = "lju-request-id"
)

// Env is an environment type
const (
	ENV_DEV  = "dev"
	ENV_PROD = "prod"
)

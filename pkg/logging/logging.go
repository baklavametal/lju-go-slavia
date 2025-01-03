package logging

import (
	"context"
	"os"

	"log/slog"

	"github.com/lmittmann/tint"

	"github.com/baklavametal/lju-go-slavia/pkg/constants"
)

var env = constants.ENV_PROD

func Get(ctx context.Context, args ...any) (*slog.Logger, context.Context) {
	ctxLogger := ctx.Value(string(constants.ContextLoggerKey))
	if ctxLogger != nil {
		return ctxLogger.(*slog.Logger).With(args...), ctx
	}
	if env == constants.ENV_PROD {
		logger, ctx := NewProdLogger(ctx)
		return logger.With(args...), ctx
	}

	logger, ctx := NewDevLogger(ctx)
	return logger.With(args...), ctx
}

func NewDevLogger(ctx context.Context) (*slog.Logger, context.Context) {
	env = constants.ENV_DEV
	logger := slog.New(tint.NewHandler(os.Stdout, nil))
	ctx = context.WithValue(ctx, string(constants.ContextLoggerKey), logger)
	return logger, ctx
}

func NewProdLogger(ctx context.Context) (*slog.Logger, context.Context) {
	env = constants.ENV_PROD
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx = context.WithValue(ctx, string(constants.ContextLoggerKey), logger)
	return logger, ctx
}

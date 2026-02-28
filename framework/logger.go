package framework

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitLogger() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	var config zap.Config
	if env == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var err error
	Log, err = config.Build()
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(Log)
}

func FromContext(ctx context.Context) *zap.Logger {
	if reqID, ok := ctx.Value("request_id").(string); ok {
		return Log.With(zap.String("request_id", reqID))
	}
	return Log
}

package logger

import (
	"awesomeEval/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitLogger(cfg *config.Config) error {
	logConf := cfg.Log

	level := zapcore.InfoLevel
	_ = level.Set(logConf.Level)

	// 修改 EncoderConfig，使用 ISO8601 时间格式
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder // ts 输出 ISO8601 字符串

	logCfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      logConf.Development,
		Encoding:         logConf.Encoding,
		OutputPaths:      logConf.OutputPaths,
		ErrorOutputPaths: logConf.ErrorOutputPaths,
		EncoderConfig:    encoderCfg,
	}

	logger, err := logCfg.Build()
	if err != nil {
		return err
	}

	Log = logger
	return nil
}

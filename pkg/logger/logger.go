package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Leopold1975/banners_control/internal/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

const (
	InfoLevel    = "info"
	DebugLevel   = "debug"
	JSONEncoding = "json"
)

func New(cfg config.Logger) (Logger, error) {
	var logLvl zapcore.Level

	switch cfg.Level {
	case InfoLevel:
		logLvl = zap.InfoLevel
	case DebugLevel:
		logLvl = zap.DebugLevel
	default:
		log.Fatal("unexpected log level")
	}

	config := zap.Config{ //nolint:exhaustruct
		Level:    zap.NewAtomicLevelAt(logLvl),
		Encoding: JSONEncoding,
		EncoderConfig: zapcore.EncoderConfig{ //nolint:exhaustruct
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.TimeEncoderOfLayout("2006 Jan 02 15:04:05 -0700 MST"),

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,

			LineEnding: "\n",
		},
		OutputPaths:      append([]string{"stdout"}, cfg.Output...),
		ErrorOutputPaths: append([]string{"stderr"}, cfg.ErrOutput...),
	}

	core, err := getCore(logLvl, config)
	if err != nil {
		return Logger{}, err
	}

	logg := zap.New(core, zap.AddCaller())

	return Logger{logg.Sugar()}, nil
}

func getCore(logLvl zapcore.Level, config zap.Config) (zapcore.Core, error) { //nolint:ireturn
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	levelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zap.ErrorLevel && lvl >= logLvl
	})

	ws, err := toMultiSyncer(config.OutputPaths)
	if err != nil {
		return nil, err
	}

	wsErr, err := toMultiSyncer(config.ErrorOutputPaths)
	if err != nil {
		return nil, err
	}

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(config.EncoderConfig), ws, levelEnabler),
		zapcore.NewCore(zapcore.NewJSONEncoder(config.EncoderConfig), wsErr, highPriority),
	)

	return core, nil
}

func toMultiSyncer(files []string) (zapcore.WriteSyncer, error) { //nolint:ireturn
	w := make([]zapcore.WriteSyncer, 0, len(files))

	for _, f := range files {
		switch f {
		case "stderr":
			w = append(w, zapcore.AddSync(os.Stderr))
		case "stdout":
			w = append(w, zapcore.AddSync(os.Stdout))
		default:
			if err := os.MkdirAll(filepath.Dir(f), os.ModePerm); err != nil {
				return nil, fmt.Errorf("mkdir error %w", err)
			}

			file, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
			if err != nil {
				return nil, fmt.Errorf("open file error %w", err)
			}

			w = append(w, zapcore.AddSync(file))
		}
	}

	ws := zapcore.NewMultiWriteSyncer(w...)

	return ws, nil
}

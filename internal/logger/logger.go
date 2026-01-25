package logger

import (
	"fmt"
	"log/slog"
	"os"
)

type logger struct {
	logger *slog.Logger
	shutUp bool
}

type Conf struct {
	Debug bool
}

func Create(conf Conf) (*logger, error) {
	newLogger := new(logger)
	var programLevel = new(slog.LevelVar) /* Info by default */

	if conf.Debug {
		programLevel.Set(slog.LevelDebug)
	}

	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})

	l := slog.New(h)

	newLogger.logger = l

	return newLogger, nil
}

func (l *logger) ShutUp() {
	l.shutUp = true
}

func (l *logger) Speak() {
	l.shutUp = false
}

func (l *logger) Debug(fmtStr string, vals ...any) {
	if l.shutUp {
		return
	}
	l.logger.Debug(format(fmtStr, vals))
}

func (l *logger) Info(fmtStr string, vals ...any) {
	if l.shutUp {
		return
	}
	l.logger.Info(format(fmtStr, vals))
}

func (l *logger) Warning(fmtStr string, vals ...any) {
	if l.shutUp {
		return
	}
	l.logger.Warn(format(fmtStr, vals))
}

func (l *logger) Error(fmtStr string, vals ...any) {
	if l.shutUp {
		return
	}
	l.logger.Error(format(fmtStr, vals))
}

func (l *logger) Refresh() error {
	return nil
}

func format(fmtStr string, a []any) string {
	if len(a) == 0 {
		return fmtStr
	}

	return fmt.Sprintf(fmtStr, a...)
}

package util

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
)

// NewLogger creates a new logger instance with log file rotation enabled.
func NewLogger() *logrus.Logger {

	log := logrus.New()

	logFile := &lumberjack.Logger{
		Filename:   "/home/jenil/Documents/logs/pluginlogs/app.log",
		MaxSize:    50,
		MaxBackups: 5,
		MaxAge:     7,
		Compress:   true,
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	log.SetOutput(multiWriter)

	log.SetLevel(logrus.InfoLevel)

	log.SetFormatter(&logrus.TextFormatter{

		FullTimestamp: true,
	})

	return log
}

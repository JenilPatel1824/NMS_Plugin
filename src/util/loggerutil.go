package util

import (
	"github.com/sirupsen/logrus"
	"os"
)

// NewLogger initializes and returns a new instance of a logrus.Logger with predefined configurations.
func NewLogger() *logrus.Logger {

	log := logrus.New()

	log.Out = os.Stdout

	log.SetLevel(logrus.InfoLevel)

	log.SetFormatter(&logrus.TextFormatter{

		FullTimestamp: true,
	})

	return log
}

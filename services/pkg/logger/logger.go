package logger

import (
	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	"os"
)

var Logger logrus.Logger

func Init() {
	Logger = logrus.Logger{
		Out:   os.Stderr,
		Level: logrus.DebugLevel,
		Formatter: &formatter.Formatter{
			TimestampFormat: "[2006-01-02 15:04:05]",
			HideKeys:        true,
			NoColors:        false,
		},
	}
}

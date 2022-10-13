package log

import (
	"github.com/sirupsen/logrus"
)

type Logger interface {
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Warn(v ...interface{})
	Warnf(format string, v ...interface{})
}

func New(debug bool) Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
	if debug {
		l.SetLevel(logrus.DebugLevel)
	}
	return l
}

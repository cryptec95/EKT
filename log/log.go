package log

import (
	"sync"

	"github.com/EducationEKT/EKT/xlog"
)

var once sync.Once
var l xlog.XLog

func InitLog(logPath string) {
	once.Do(func() {
		l = xlog.NewDailyLog(logPath)
	})
}

func Debug(msg string, args ...interface{}) {
	l.Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	l.Info(msg, args...)
}

func Error(msg string, args ...interface{}) {
	l.Error(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	l.Warn(msg, args...)
}

func Crit(msg string, args ...interface{}) {
	l.Crit(msg, args...)
}

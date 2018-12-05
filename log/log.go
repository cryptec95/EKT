package log

import (
	"runtime"
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

func LogErr(err error) {
	if err != nil {
		Error("There is an error, %v .", err)
	}
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

func PrintStack(source string) {
	var buf [4096]byte
	runtime.Stack(buf[:], false)
	l.Crit("Panic occured at %s, %s", source, string(buf[:]))
}

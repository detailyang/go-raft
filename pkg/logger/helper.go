package logger

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

func abbrfile(s string) string {
	if strings.Contains(s, "/") {
		t := strings.Split(s, "/")
		if len(t) > 2 {
			s = strings.Join(t[len(t)-2:len(t)], "/")
		}
	}

	return s
}

func abbrfn(s string) string {
	if strings.Contains(s, ".") {
		t := strings.Split(s, ".")
		if len(t) > 2 {
			s = strings.Join(t[len(t)-1:len(t)], ".")
		}
	}

	return s
}

func callstack(entry *logrus.Entry) *logrus.Entry {
	if pc, file, line, ok := runtime.Caller(2); ok {
		fn := runtime.FuncForPC(pc).Name()
		return entry.WithField("ctx", fmt.Sprintf("%s@%s:%d", abbrfn(fn), abbrfile(file), line))
	} else {
		return entry
	}
}

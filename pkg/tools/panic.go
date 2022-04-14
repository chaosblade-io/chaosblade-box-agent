package tools

import (
	"runtime"

	"github.com/sirupsen/logrus"
)

func PanicPrintStack() {
	if err := recover(); err != nil {
		buf := make([]byte, 1<<11)
		length := runtime.Stack(buf, true)
		logrus.WithField("panic-error", err).Warnf("panic stack: %s", string(buf[:length]))
	}
}

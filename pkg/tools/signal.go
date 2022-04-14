package tools

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

type ShutdownHook interface {
	Shutdown()
}

func Hold(hooks ...ShutdownHook) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	buf := make([]byte, 1<<20)
	for {
		switch <-sig {
		case syscall.SIGINT, syscall.SIGTERM:
			logrus.Warningln("received SIGINT/SIGTERM, exit")
			for _, hook := range hooks {
				if hook == nil {
					continue
				}
				hook.Shutdown()
			}
			return
		case syscall.SIGQUIT:
			for _, hook := range hooks {
				if hook == nil {
					continue
				}
				hook.Shutdown()
			}
			len := runtime.Stack(buf, true)
			logrus.Warningf("received SIGQUIT\n%s\n", buf[:len])
		}
	}
}

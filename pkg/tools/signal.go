/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

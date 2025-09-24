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

package monitor

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/transport"
)

const (
	monitorTimeIntervalSec = 10
)

type monitor struct {
	checker
	transportClient *transport.TransportClient
}

type monitorAction struct {
	needStop  bool
	needStart bool
	needExit  bool
	reason    string
}

type checker interface {
	check() monitorAction
}

var (
	instance *monitor
	cLock    sync.Mutex
)

func (this *monitorAction) recover() {
	this.needStart = false
	this.needStop = false
	this.needExit = false
	this.reason = ""
}

func GetMonitorInstance(transportClient *transport.TransportClient) *monitor {
	if instance != nil {
		return instance
	}

	cLock.Lock()
	defer cLock.Unlock()

	if instance != nil {
		return instance
	}

	instance = &monitor{
		checker:         &defaultChecker{},
		transportClient: transportClient,
	}

	return instance
}

func (this *monitor) Start() {
	go this.doMonitor()
}

func (this *monitor) doMonitor() {
	logrus.Infof("starting monitor:%s", time.Now())

	for {
		action := this.check()
		if action.needStop {
			infoMsg := fmt.Sprintf("monitor exception[%s], stop", action.reason)
			this.StopWithReason(infoMsg)
		}

		if action.needExit {
			logrus.Warnf("monitor error[%s], exit", action.reason)
			if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
				logrus.Warnf("the monitor send SIGTERM signal to self fail:%s", err)
				os.Exit(5)
			}
		}

		if action.needStart {
			infoMsg := fmt.Sprintf("recover[%s], start", action.reason)
			this.StartWithReason(infoMsg)
		}

		time.Sleep(time.Second * monitorTimeIntervalSec)
	}
}

func (this *monitor) StopWithReason(reason string) {
	logrus.Warningf("[Controller] send stop event to server, reason: %s", reason)
	go this.sendEventToServer("stop", reason)
	// todo 因为现在没有数据收集，所以不需要controller.stop，去关停数据收集相关的controller
}

func (this *monitor) StartWithReason(reason string) {
	logrus.Infof("[Controller] send start event to server, reason: %s", reason)
	go this.sendEventToServer("start", reason)
	// todo 因为现在没有数据收集，所以不需要controller.start，去开启数据收集相关的controller
}

func (this *monitor) sendEventToServer(event, reason string) {
	uri, ok := transport.TransportUriMap[transport.API_EVENT]
	if !ok {
		return
	}
	request := transport.NewRequest()
	request.AddParam("event", event).AddParam("reason", reason)
	_, err := this.transportClient.Invoke(uri, request, true)
	if err != nil {
		logrus.Warningf("[Monitor] send %s event with %s reason to server error %s.", event, reason, err.Error())
	} else {
		logrus.Infof("[Monitor] send %s event with %s reason to server successfully.", event, reason)
	}
}

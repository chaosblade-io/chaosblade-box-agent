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
	"bytes"
	"container/list"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/conn/heartbeat"
)

type defaultChecker struct {
}

var hbAlreadyStoped = false

var hbStopThreshold = 12
var hbStartThreshold = 3

func (*defaultChecker) check() monitorAction {
	action := monitorAction{}
	action.recover()

	//todo: 这里暂时没有cpu\memory的兜底策略，因为暂时不收集process数据

	checkHeartBeat(&action)
	if action.needStop {
		return action
	}
	if action.needExit {
		return action
	}

	if action.needStart {
		hbAlreadyStoped = false
	}

	//finally
	return action
}

func checkHeartBeat(action *monitorAction) {
	hbFailContinuousCount := 0
	hbSuccContinuousCount := 0
	walkerList := list.New()
	heartbeat.HBSnapshotList.ForeachReverse(func(v interface{}) error {
		if hbSnapshot, ok := v.(heartbeat.HBSnapshot); ok {

			walkerList.PushBack(hbSnapshot)

			if !hbSnapshot.Success {
				hbFailContinuousCount++
				hbSuccContinuousCount = 0
			} else {
				hbSuccContinuousCount++
				hbFailContinuousCount = 0
			}

			if hbFailContinuousCount == hbStopThreshold && !hbAlreadyStoped {
				hbAlreadyStoped = true
				action.recover()
				action.needStop = true
				action.reason = "stop because of heartbeat"
				printWalkerList(action.reason, walkerList)
				return errors.New("nolog")
			}

			if hbFailContinuousCount == hbStopThreshold && hbAlreadyStoped {
				return errors.New("nolog")
			}

			if hbSuccContinuousCount == hbStartThreshold && hbAlreadyStoped {
				action.recover()
				action.needStart = true
				printWalkerList("can start because of heartbeat", walkerList)
				return errors.New("nolog")
			}

			if hbSuccContinuousCount == hbStartThreshold && !hbAlreadyStoped {
				return errors.New("nolog")
			}
		}

		return nil
	}, true)
}

func printWalkerList(info string, l *list.List) {
	var buf bytes.Buffer
	buf.WriteString(info)
	buf.WriteString(", walker list is : ")
	for element := l.Front(); element != nil; element = element.Next() {
		buf.WriteString(fmt.Sprintf("%v|", element.Value))
	}
	logrus.Warn(buf.String())
}

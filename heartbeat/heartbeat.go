/*
 * Copyright 1999-2021 Alibaba Group Holding Ltd.
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

package heartbeat

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/meta"
	"github.com/chaosblade-io/chaos-agent/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

//httpClientConfig defines heartbeat configuration
type Config struct {
	Period time.Duration
}

type heartbeat struct {
	period time.Duration
	*transport.Transport
}

type HBSnapshot struct {
	Success bool
}

func (beat *heartbeat) record(success bool) {
	HBSnapshotList.Put(HBSnapshot{
		Success: success,
	})
}

var HBSnapshotList, _ = tools.NewLimitedSortList(26)

//New heartbeat
func New(config Config, trans *transport.Transport) *heartbeat {
	handler := &GetPingHandler().InterceptorRequestHandler
	trans.RegisterHandler(transport.Ping, handler)
	return &heartbeat{
		period:    config.Period,
		Transport: trans,
	}
}

//Start heartbeat service
func (beat *heartbeat) Start() *heartbeat {
	ticker := time.NewTicker(beat.period)
	go func() {
		defer tools.PrintPanicStack()
		for range ticker.C {
			request := transport.NewRequest()

			uri := transport.NewUri(transport.HttpHandlerHeartbeat)
			if meta.IsHostMode() {
				request.AddHeader(tools.AppInstanceKeyName, meta.Info.ApplicationInstance)
				request.AddHeader(tools.AppGroupKeyName, meta.Info.ApplicationGroup)
			}
			beat.sendHeartbeat(uri, request)
		}
	}()
	log.WithFields(log.Fields{
		"ver":         meta.Info.Version,
		"appInstance": meta.Info.ApplicationInstance,
		"appGroup":    meta.Info.ApplicationGroup,
	}).Infoln("[heartbeat] start successfully")
	return nil
}

// sendHeartbeat
func (beat *heartbeat) sendHeartbeat(uri transport.Uri, request *transport.Request) {
	response, err := beat.Invoke(uri, request)
	if err != nil {
		log.Errorln("[heartbeat] send failed.", err)
		beat.record(false)
		return
	}
	if !response.Success {
		log.Errorf("[heartbeat] send failed. %+v", response)
		beat.record(false)
		return
	}
	log.WithFields(log.Fields{
		"ver":         meta.Info.Version,
		"appInstance": meta.Info.ApplicationInstance,
		"appGroup":    meta.Info.ApplicationGroup,
	}).Debugln("[heartbeat] success")
	beat.record(true)
}

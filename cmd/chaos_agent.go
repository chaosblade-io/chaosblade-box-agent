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

package main

import (
	"bytes"
	"net/http"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/conn"
	closer "github.com/chaosblade-io/chaos-agent/conn/close"
	"github.com/chaosblade-io/chaos-agent/conn/connect"
	"github.com/chaosblade-io/chaos-agent/conn/heartbeat"
	"github.com/chaosblade-io/chaos-agent/conn/metric"
	"github.com/chaosblade-io/chaos-agent/metricreport"
	"github.com/chaosblade-io/chaos-agent/pkg/helm3"
	chaoshttp "github.com/chaosblade-io/chaos-agent/pkg/http"
	"github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/pkg/log"
	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
	api2 "github.com/chaosblade-io/chaos-agent/web/api"
	"github.com/chaosblade-io/chaos-agent/web/handler/litmuschaos"
)

var pidFile = "/var/run/chaos.pid"

func main() {
	options.NewOptions()
	log.InitLog(&options.Opts.LogConfig)

	options.Opts.SetOthersByFlags()

	// new transport newConn
	clientInstance, err := chaoshttp.NewHttpClient(options.Opts.TransportConfig)
	if err != nil {
		logrus.Errorf("create transport client instance failed, err: %s", err.Error())
		handlerErr(err)
	}
	transportClient := transport.NewTransportClient(clientInstance)
	transport.InitTransprotUri()

	// k8s
	k8sInstance := kubernetes.GetInstance()

	// registry report metric
	reportMetricConfigMap := metricreport.New(k8sInstance, transportClient)
	reportMetricConfigMap.InitMetricConfig()

	// helm
	buf := new(bytes.Buffer)
	h := helm3.GetHelmInstance(litmuschaos.LitmusHelmName, litmuschaos.LitmusHelmNamespace, buf)

	// conn to server
	connectClient := connect.NewClientConnectHandler(transportClient)
	heartbeatClient := heartbeat.NewClientHeartbeatHandler(options.Opts.HeartbeatConfig, transportClient)
	metricClient := metric.NewClientMetricHandler(transportClient, reportMetricConfigMap)
	newConn := conn.NewConn()
	newConn.Register(transport.API_REGISTRY, connectClient)
	newConn.Register(transport.API_HEARTBEAT, heartbeatClient)
	newConn.Register(transport.API_METRIC, metricClient)
	newConn.Start()

	// new api
	api := api2.NewAPI()
	err = api.Register(transportClient, k8sInstance, h)
	if err != nil {
		logrus.Errorf("register api failed, err: %s", err.Error())
		handlerErr(err)
	}

	// listen server
	go func() {
		defer tools.PanicPrintStack()
		err := http.ListenAndServe(":"+options.Opts.Port, nil)
		if err != nil {
			logrus.Warningln("Start http server failed")
			handlerErr(err)
		}
	}()

	handlerSuccess()

	closeClient := closer.NewClientCloseHandler(transportClient)
	tools.Hold(closeClient)
}

func handlerSuccess() {
	pid := os.Getpid()
	err := writePid(pid)
	if err != nil {
		logrus.Panic("write pid: ", pidFile, " failed. ", err)
	}
}

func handlerErr(err error) {
	if err == nil {
		return
	}
	logrus.Warningf("start agent failed because of %v", err)
	writePid(-1)
	logrus.Errorf("chaos agent will exit")
	os.Exit(1)
}

func writePid(pid int) error {
	file, err := os.OpenFile(pidFile, os.O_RDWR|os.O_CREATE, 0o755)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strconv.Itoa(pid))
	return err
}

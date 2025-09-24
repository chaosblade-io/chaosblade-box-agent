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

package metric

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/metricreport"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type ClientMetricHandler struct {
	// metricReportConfig options.MetricReportConfig
	reportMetricConfigMap *metricreport.ReportMetricConfigMap
	transportClient       *transport.TransportClient
}

// config options.MetricReportConfig,
func NewClientMetricHandler(transportClient *transport.TransportClient, reportMetricConfigMap *metricreport.ReportMetricConfigMap) *ClientMetricHandler {
	return &ClientMetricHandler{
		// metricReportConfig: config,

		reportMetricConfigMap: reportMetricConfigMap,
		transportClient:       transportClient,
	}
}

func (cmh *ClientMetricHandler) Start() error {
	// metricreport.ReportMetricConfigDatas.RLock()
	// defer metricreport.ReportMetricConfigDatas.RUnlock()
	cmh.reportMetricConfigMap.RLock()
	defer cmh.reportMetricConfigMap.RUnlock()

	// metricreport.ReportMetricConfigDatas.ReportMetricConfig
	for metricName, reportMetricConfigData := range cmh.reportMetricConfigMap.ReportMetricConfig {
		if !reportMetricConfigData.Enable {
			continue
		}

		logrus.Infof("[metric] report %s metric, start!", metricName)
		ticker := time.NewTicker(reportMetricConfigData.Period)
		go func() {
			defer tools.PanicPrintStack()
			for range ticker.C {
				// metricreport.ReportMetricConfigDatas.RLock()
				go reportMetricConfigData.Report()
			}
		}()
		reportMetricConfigData.Ticker = ticker
		reportMetricConfigData.Enable = true
		cmh.reportMetricConfigMap.ReportMetricConfig[metricName] = reportMetricConfigData
	}

	return nil
}

// Stop for monitor
func (cmh *ClientMetricHandler) Stop(stopCh chan bool) error {
	cmh.reportMetricConfigMap.RLock()
	defer cmh.reportMetricConfigMap.RUnlock()

	for metricName := range cmh.reportMetricConfigMap.ReportMetricConfig {
		if err := cmh.reportMetricConfigMap.CloseEnable(metricName); err != nil {
			return err
		}
	}

	return nil
}

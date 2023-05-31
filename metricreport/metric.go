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

package metricreport

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/chaosblade-io/chaos-agent/collector/kubernetes"
	k8s "github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type MetricCollector interface {
	Report()
}

//var ReportMetricConfigDatas *ReportMetricConfigMap

type ReportMetricConfigMap struct {
	sync.RWMutex
	K8sChannel *k8s.Channel
	Transport  *transport.TransportClient

	ReportMetricConfig map[string]ReportMetricConfig
}

type ReportMetricConfig struct {
	MetricCollector

	// enable flag, 提供给server设置
	Enable bool
	Period time.Duration

	Ticker *time.Ticker
}

const DefaultReportMetricPeriod = 10 * time.Second

func New(k8sChannel *k8s.Channel, trans *transport.TransportClient) *ReportMetricConfigMap {
	return &ReportMetricConfigMap{
		K8sChannel:         k8sChannel,
		Transport:          trans,
		ReportMetricConfig: make(map[string]ReportMetricConfig, 0),
	}
	//return ReportMetricConfigDatas
}

func (rmc *ReportMetricConfigMap) InitMetricConfig() {
	// pod
	opts := metav1.ListOptions{
		//FieldSelector: "spec.nodeName=" + LocalNodeName,
	}
	serviceCollector := kubernetes.NewServiceCollector(rmc.Transport, rmc.K8sChannel, opts)
	podCollector := kubernetes.NewPodCollector(rmc.Transport, rmc.K8sChannel, serviceCollector, opts)
	podMetricConfig := ReportMetricConfig{podCollector, options.Opts.PodMetricFlag, DefaultReportMetricPeriod, nil} // todo for test true
	rmc.MetricRegistry(transport.K8sPod, podMetricConfig)
}

func (rmc *ReportMetricConfigMap) MetricRegistry(metricName string, config ReportMetricConfig) {
	rmc.Lock()
	defer rmc.Unlock()

	if config.MetricCollector == nil {
		logrus.Warnf("[metric report] %s, registry collector is nil", metricName)
		return
	}

	rmc.ReportMetricConfig[metricName] = config
}

func (rmc *ReportMetricConfigMap) CloseEnable(metricName string) error {
	rmc.Lock()
	defer rmc.Unlock()

	reportConfig, ok := rmc.ReportMetricConfig[metricName]
	if !ok {
		return nil
	}

	if reportConfig.Ticker == nil {
		return nil
	}

	reportConfig.Enable = false
	reportConfig.Ticker.Stop()
	rmc.ReportMetricConfig[metricName] = reportConfig
	return nil
}

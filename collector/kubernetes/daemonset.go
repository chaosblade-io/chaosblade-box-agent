/*
 * Copyright 2025 The ChaosBlade Authors
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

package kubernetes

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type DaemonsetInfo struct {
	CommonInfo
	Namespace              string `json:"namespace,omitempty"`
	CurrentNumberScheduled int32  `json:"currentNumberScheduled,omitempty"`
	DesiredNumberScheduled int32  `json:"desiredNumberScheduled,omitempty"`
	NumberAvailable        int32  `json:"numberAvailable,omitempty"`
	NumberMisscheduled     int32  `json:"numberMisscheduled,omitempty"`
	NumberReady            int32  `json:"int32,omitempty"`
	ObservedGeneration     int64  `json:"observedGeneration,omitempty"`
	UpdatedNumberScheduled int32  `json:"updatedNumberScheduled,omitempty"`
	UpdateStrategy         string `json:"updateStrategy,omitempty"`
}

type DaemonsetCollector struct {
	*K8sBaseCollector
	opts metav1.ListOptions
}

func NewDaemonsetCollector(trans *transport.TransportClient, k8sChannel *kubernetes.Channel, opts metav1.ListOptions) *DaemonsetCollector {
	uri, ok := transport.TransportUriMap[transport.API_K8S_DAEMONSET]
	if !ok {
		return nil
	}
	collector := createK8sBaseCollector(kubernetes.DaemonsetResource, k8sChannel, trans, uri)
	daemonsetCollector := &DaemonsetCollector{
		K8sBaseCollector: collector,
		opts:             opts,
	}
	return daemonsetCollector
}

func (collector *DaemonsetCollector) Report() {
	if collector.indexer == nil {
		if collector.k8sChannel.ClientSet == nil {
			logrus.Warnf("[DAEMONSET REPORT] k8s client not enable")
			return
		}

		collector.indexer = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &v1.DaemonSet{}, collector.opts, createDaemonSetListWatch)
	}

	infos, err := collector.getDaemonsetInfo()
	if err != nil {
		logrus.Warnf("[DAEMONSET REPORT] get daemonset info failed, %v", err)
		return
	}
	collector.reportK8sMetric(metav1.NamespaceAll, true, infos, len(infos))
	collector.reportNotExistResource()
}

// getDaemonsetInfo
func (collector *DaemonsetCollector) getDaemonsetInfo() ([]*DaemonsetInfo, error) {
	list := collector.indexer.List()
	logrus.Debugf("[DAEMONSET REPORT] daemonset list len: %v", len(list))
	daemonsets := make([]*DaemonsetInfo, 0)
	for _, dm := range list {
		d := dm.(*v1.DaemonSet)
		daemonsetInfo := &DaemonsetInfo{
			CommonInfo: CommonInfo{
				Uid:         string(d.UID),
				Name:        d.Name,
				CreatedTime: d.CreationTimestamp.Format(time.RFC3339Nano),
				Labels:      d.Labels,
				Exist:       true,
			},
			Namespace:              d.Namespace,
			CurrentNumberScheduled: d.Status.CurrentNumberScheduled,
			DesiredNumberScheduled: d.Status.DesiredNumberScheduled,
			NumberAvailable:        d.Status.NumberAvailable,
			NumberMisscheduled:     d.Status.NumberMisscheduled,
			NumberReady:            d.Status.NumberReady,
			UpdatedNumberScheduled: d.Status.UpdatedNumberScheduled,
			UpdateStrategy:         string(d.Spec.UpdateStrategy.Type),
		}
		// handle increment
		daemonsetInfo = collector.handleDaemonsetIncrement(daemonsetInfo)
		daemonsets = append(daemonsets, daemonsetInfo)
	}
	return daemonsets, nil
}

// 处理 Daemonset 增量
func (collector *DaemonsetCollector) handleDaemonsetIncrement(daemonsetInfo *DaemonsetInfo) *DaemonsetInfo {
	collector.IdentifierLock.Lock()
	defer collector.IdentifierLock.Unlock()
	sumData, err := tools.Md5sumData(daemonsetInfo)
	if err == nil {
		if v, ok := collector.identifiers[daemonsetInfo.Uid]; ok {
			if v.Md5 == sumData && v.Cid != "" {
				// 如果相等，说明数据没变，则只上报关键数据
				daemonsetInfo = &DaemonsetInfo{
					CommonInfo: CommonInfo{
						Uid:   v.Uid,
						Exist: true,
						Cid:   v.Cid,
					},
				}
			} else {
				// 如果存在相同的 UID，但是 md5 不一致，则需要更新 md5，同时上报全量数据
				v.Md5 = sumData
			}
			v.Curr = true
		} else {
			collector.identifiers[daemonsetInfo.Uid] = &ResourceIdentifier{
				Uid: daemonsetInfo.Uid,
				Md5: sumData,
				// 当前是否存在；
				Curr: true,
				name: daemonsetInfo.Name,
			}
		}
	}
	return daemonsetInfo
}

func (collector *DaemonsetCollector) reportNotExistResource() {
	// old Daemonsets
	daemonsets := make([]*DaemonsetInfo, 0)
	collector.IdentifierLock.Lock()
	daemonsetIdentifiers := collector.identifiers
	logrus.Debugf("daemonsetIdentifiers len: %d", len(daemonsetIdentifiers))
	if daemonsetIdentifiers != nil {
		for k, v := range daemonsetIdentifiers {
			if v.Curr {
				v.Curr = false
			} else {
				if v.Cid != "" {
					daemonsets = append(daemonsets, &DaemonsetInfo{
						CommonInfo: CommonInfo{
							Uid:   v.Uid,
							Cid:   v.Cid,
							Exist: false,
						},
					})
				}
				logrus.Debugf("daemonsetIdentifiers delete: %s", v.Uid)
				delete(daemonsetIdentifiers, k)
			}
		}
		collector.IdentifierLock.Unlock()
	}
	collector.reportK8sMetric(metav1.NamespaceAll, false, daemonsets, len(daemonsets))
}

func createDaemonSetListWatch(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AppsV1().DaemonSets(ns).List(context.TODO(), options)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AppsV1().DaemonSets(ns).Watch(context.TODO(), options)
		},
	}
}

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
	"errors"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type NamespaceCollector struct {
	K8sBaseCollector
	opts metav1.ListOptions
}

type NamespaceInfo struct {
	CommonInfo
}

func NewNamespaceCollector(trans *transport.TransportClient, k8sChannel *kubernetes.Channel, opts metav1.ListOptions) *NamespaceCollector {
	uri, ok := transport.TransportUriMap[transport.API_K8S_NAMESPACE]
	if !ok {
		return nil
	}
	collector := createK8sBaseCollector(kubernetes.NodeResource, k8sChannel, trans, uri)

	return &NamespaceCollector{
		collector,
		opts,
	}
}

func (collector *NamespaceCollector) Report() {
	if collector.indexer == nil {
		if collector.k8sChannel.ClientSet == nil {
			logrus.Warnf("[NAMESPACE REPORT] k8s client not enable")
			return
		}

		collector.indexer = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &v1.Namespace{}, collector.opts, createNamespaceListWatch)
	}

	namespaces, err := collector.getNamespaceInfo()
	if err != nil {
		logrus.Errorf("[NAMESPACE REPORT] get namespace failed, %v", err)
		return
	}

	collector.reportK8sMetric(metav1.NamespaceAll, true, namespaces, len(namespaces))
	collector.reportNotExistResource()
}

func (collector *NamespaceCollector) getNamespaces() ([]string, error) {
	if collector.indexer == nil {
		return nil, errors.New("namespace index is nil")
	}
	list := collector.indexer.List()
	collector.indexer.ListKeys()
	namespaces := make([]string, 0)
	for _, ns := range list {
		namespace := ns.(*v1.Namespace)
		spaceName := strings.TrimSpace(namespace.GetName())
		namespaces = append(namespaces, spaceName)
	}
	return namespaces, nil
}

func (collector *NamespaceCollector) getNamespaceInfo() ([]*NamespaceInfo, error) {
	list := collector.indexer.List()
	namespaces := make([]*NamespaceInfo, 0)
	for _, ns := range list {
		namespace := ns.(*v1.Namespace)
		namespaceInfo := &NamespaceInfo{
			CommonInfo: CommonInfo{
				Uid:         string(namespace.GetUID()),
				Name:        namespace.GetName(),
				CreatedTime: namespace.GetCreationTimestamp().Format(time.RFC3339Nano),
				Labels:      namespace.GetLabels(),
				Exist:       true,
			},
		}
		namespaceInfo = collector.handleNamespaceIncrement(namespaceInfo)
		namespaces = append(namespaces, namespaceInfo)
	}
	return namespaces, nil
}

func (collector *NamespaceCollector) handleNamespaceIncrement(namespaceInfo *NamespaceInfo) *NamespaceInfo {
	collector.IdentifierLock.Lock()
	defer collector.IdentifierLock.Unlock()
	sumData, err := tools.Md5sumData(namespaceInfo)
	if err == nil {
		if v, ok := collector.identifiers[namespaceInfo.Uid]; ok {
			if v.Md5 == sumData && v.Cid != "" {
				// 如果相等，说明数据没变，则只上报关键数据
				namespaceInfo = &NamespaceInfo{
					CommonInfo{
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
			collector.identifiers[namespaceInfo.Uid] = &ResourceIdentifier{
				Uid:  namespaceInfo.Uid,
				Md5:  sumData,
				Curr: true,
				name: namespaceInfo.Name,
			}
		}
	}
	return namespaceInfo
}

func (collector *NamespaceCollector) reportNotExistResource() {
	// old pods
	namespaces := make([]*NamespaceInfo, 0)
	collector.IdentifierLock.Lock()
	identifiers := collector.identifiers
	logrus.Debugf("[NAMESPACE REPORT] namespaceIdentifiers len: %d", len(identifiers))
	if identifiers != nil {
		for k, v := range identifiers {
			if v.Curr {
				v.Curr = false
			} else {
				if v.Cid != "" {
					namespaces = append(namespaces, &NamespaceInfo{
						CommonInfo: CommonInfo{
							Uid:   v.Uid,
							Cid:   v.Cid,
							Exist: false,
						},
					})
				}
				logrus.Debugf("[NAMESPACE REPORT] namespaceIdentifiers delete: %s", v.Uid)
				delete(identifiers, k)
			}
		}
	}
	collector.IdentifierLock.Unlock()
	collector.reportK8sMetric(metav1.NamespaceAll, false, namespaces, len(namespaces))
}

func createNamespaceListWatch(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Namespaces().List(context.TODO(), options)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Namespaces().Watch(context.TODO(), options)
		},
	}
}

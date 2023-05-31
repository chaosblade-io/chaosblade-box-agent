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

package kubernetes

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type ServiceInfo struct {
	CommonInfo
	Namespace  string            `json:"namespace,omitempty"`
	ClusterIp  string            `json:"clusterIp,omitempty"`
	ExternalIp string            `json:"externalIp,omitempty"`
	Ports      []string          `json:"ports,omitempty"`
	Type       string            `json:"type,omitempty"`
	Selector   map[string]string `json:"selector,omitempty"`
}

type ServiceCollector struct {
	K8sBaseCollector
	namespaceCollector *NamespaceCollector
	SelectorLock       sync.Mutex
	opts               metav1.ListOptions
}

func NewServiceCollector(trans *transport.TransportClient, k8sChannel *kubernetes.Channel, opts metav1.ListOptions) *ServiceCollector {
	uri, ok := transport.TransportUriMap[transport.API_K8S_SERVICE]
	if !ok {
		logrus.Warnf("service collector, get uri failed!")
		//return nil
	}
	collector := createK8sBaseCollector(kubernetes.NodeResource, k8sChannel, trans, uri)

	return &ServiceCollector{
		K8sBaseCollector: collector,
		SelectorLock:     sync.Mutex{},
		opts:             opts,
	}
}

func (collector *ServiceCollector) Report() {
	if collector.indexer == nil {
		// 需要构建reflector
		if collector.k8sChannel.ClientSet == nil {
			logrus.Warnf("[SERVICE REPORT] k8s client not enable")
			return
		}

		collector.indexer = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &v1.Service{}, collector.opts, createServiceListWatch)
	}

	infos, _, err := collector.getServiceInfo()
	if err != nil {
		logrus.Errorf("[SERVICE REPORT] get service failed, %v", err)
	}

	collector.reportK8sMetric(metav1.NamespaceAll, true, infos, len(infos))
	collector.reportNotExistResource()

}

func (collector *ServiceCollector) SetSelector() {
	if collector.indexer == nil {
		// 需要构建reflector
		if collector.k8sChannel.ClientSet == nil {
			logrus.Warnf("[SERVICE REPORT] k8s client not enable")
			return
		}

		collector.indexer = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &v1.Service{}, collector.opts, createServiceListWatch)
	}

	_, selectors, err := collector.getServiceInfo()
	if err != nil {
		logrus.Errorf("[SERVICE REPORT] get service in 'ssssss' namespace failed, %v", err)
	}
	collector.SelectorLock.Lock()
	collector.selectors = selectors
	collector.SelectorLock.Unlock()

}

// getServiceInfo
func (collector *ServiceCollector) getServiceInfo() ([]*ServiceInfo, []func(node podNode), error) {
	list := collector.indexer.List()
	logrus.Debugf("[SERVICE REPORT] get services from indexer, len: %d, list keys: %v", len(list), collector.indexer.ListKeys())
	var services = make([]*ServiceInfo, 0)
	selectors := make([]func(node podNode), 0)
	for _, srv := range list {
		s := srv.(*v1.Service)
		serviceInfo := &ServiceInfo{
			CommonInfo: CommonInfo{
				Uid:         string(s.UID),
				Name:        s.Name,
				CreatedTime: s.CreationTimestamp.Format(time.RFC3339Nano),
				Labels:      s.Labels,
				Exist:       true,
			},
			Namespace: s.Namespace,
			ClusterIp: s.Spec.ClusterIP,
			Ports:     servicePortsToString(s.Spec.Ports),
			Type:      string(s.Spec.Type),
			Selector:  s.Spec.Selector,
		}
		serviceInfo.ExternalIp = getServiceExternalIP(s, true)
		// handle increment
		serviceInfo = collector.handleServiceIncrement(serviceInfo)
		services = append(services, serviceInfo)
		// add selector
		if s.Spec.Selector != nil {
			selector := labels.SelectorFromSet(s.Spec.Selector)
			selectors = append(selectors, selectorMatch(
				s.Namespace, selector, kubernetes.ServiceResource, string(s.UID),
			))
		}
	}
	return services, selectors, nil
}

func (collector *ServiceCollector) handleServiceIncrement(serviceInfo *ServiceInfo) *ServiceInfo {
	collector.IdentifierLock.Lock()
	defer collector.IdentifierLock.Unlock()
	sumData, err := tools.Md5sumData(serviceInfo)
	if err == nil {
		if v, ok := collector.identifiers[serviceInfo.Uid]; ok {
			if v.Md5 == sumData && v.Cid != "" {
				// 如果相等，说明数据没变，则只上报关键数据
				serviceInfo = &ServiceInfo{
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
			collector.identifiers[serviceInfo.Uid] = &ResourceIdentifier{
				Uid: serviceInfo.Uid,
				Md5: sumData,
				// 当前是否存在；
				Curr: true,
				name: serviceInfo.Name,
			}
		}
	}
	return serviceInfo
}

func (collector *ServiceCollector) reportNotExistResource() {
	// old service
	var services = make([]*ServiceInfo, 0)
	collector.IdentifierLock.Lock()
	serviceIdentifiers := collector.identifiers
	logrus.Debugf("serviceIdentifiers len: %d", len(serviceIdentifiers))
	if serviceIdentifiers != nil {
		for k, v := range serviceIdentifiers {
			if v.Curr {
				v.Curr = false
			} else {
				if v.Cid != "" {
					services = append(services, &ServiceInfo{
						CommonInfo: CommonInfo{
							Uid:   v.Uid,
							Cid:   v.Cid,
							Exist: false,
						},
					})
				}
				logrus.Debugf("serviceIdentifiers delete: %s", v.Uid)
				delete(serviceIdentifiers, k)
			}
		}
	}
	collector.IdentifierLock.Unlock()
	collector.reportK8sMetric(metav1.NamespaceAll, false, services, len(services))
}

func createServiceListWatch(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Services(ns).List(context.TODO(), options)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Services(ns).Watch(context.TODO(), options)
		},
	}
}

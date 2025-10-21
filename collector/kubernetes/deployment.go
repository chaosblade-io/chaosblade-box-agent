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

type DeploymentInfo struct {
	CommonInfo
	Namespace           string `json:"namespace,omitempty"`
	AvailableReplicas   int32  `json:"availableReplicas,omitempty"`
	Replicas            int32  `json:"replicas,omitempty"`
	ObservedGeneration  int64  `json:"observedGeneration,omitempty"`
	ReadyReplicas       int32  `json:"readyReplicas,omitempty"`
	UpdatedReplicas     int32  `json:"updatedReplicas,omitempty"`
	Strategy            string `json:"strategy,omitempty"`
	UnavailableReplicas int32  `json:"unavailableReplicas,omitempty"`
}

type DeploymentCollector struct {
	*K8sBaseCollector
	opts metav1.ListOptions
}

func NewDeploymentCollector(trans *transport.TransportClient, k8sChannel *kubernetes.Channel, opts metav1.ListOptions) *DeploymentCollector {
	uri, ok := transport.TransportUriMap[transport.API_K8S_DEPLOYMNT]
	if !ok {
		return nil
	}
	k8sBaseCollector := createK8sBaseCollector(kubernetes.DeploymentResource, k8sChannel, trans, uri)

	return &DeploymentCollector{
		K8sBaseCollector: k8sBaseCollector,
		opts:             opts,
	}
}

func (collector *DeploymentCollector) Report() {
	if collector.indexer == nil {
		// 需要构建reflector
		if collector.k8sChannel.ClientSet == nil {
			logrus.Warnf("[DEPLOYMENT REPORT] k8s client not enable")
			return
		}

		collector.indexer = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &v1.Deployment{},
			collector.opts, createDeploymentListWatch)
	}
	infos, err := collector.getDeploymentInfo()
	if err != nil {
		logrus.Errorf("[DEPLOYMENT REPORT] get deployment failed, %v", err)
		return
	}
	collector.reportK8sMetric(metav1.NamespaceAll, true, infos, len(infos))
	collector.reportNotExistResource()
}

// getDeploymentInfo
func (collector *DeploymentCollector) getDeploymentInfo() ([]*DeploymentInfo, error) {
	list := collector.indexer.List()
	logrus.Debugf("[DEPLOYMENT REPORT] get deployments from lister, size: %d", len(list))
	// deployments
	deployments := make([]*DeploymentInfo, 0)
	for _, dm := range list {
		d := dm.(*v1.Deployment)
		deploymentInfo := &DeploymentInfo{
			CommonInfo: CommonInfo{
				Uid:         string(d.UID),
				Name:        d.Name,
				CreatedTime: d.CreationTimestamp.Format(time.RFC3339Nano),
				Labels:      d.Labels,
				Exist:       true,
			},
			Namespace:           d.Namespace,
			AvailableReplicas:   d.Status.AvailableReplicas,
			Replicas:            *d.Spec.Replicas,
			ObservedGeneration:  d.Status.ObservedGeneration,
			ReadyReplicas:       d.Status.ReadyReplicas,
			UpdatedReplicas:     d.Status.UpdatedReplicas,
			UnavailableReplicas: d.Status.UnavailableReplicas,
			Strategy:            string(d.Spec.Strategy.Type),
		}

		// handle increment
		deploymentInfo = collector.handleDeploymentIncrement(deploymentInfo)
		deployments = append(deployments, deploymentInfo)
	}
	return deployments, nil
}

// 处理 deployment 增量
func (collector *DeploymentCollector) handleDeploymentIncrement(deploymentInfo *DeploymentInfo) *DeploymentInfo {
	collector.IdentifierLock.Lock()
	defer collector.IdentifierLock.Unlock()
	sumData, err := tools.Md5sumData(deploymentInfo)
	if err == nil {
		if v, ok := collector.identifiers[deploymentInfo.Uid]; ok {
			if v.Md5 == sumData && v.Cid != "" {
				// 如果相等，说明数据没变，则只上报关键数据
				deploymentInfo = &DeploymentInfo{
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
			collector.identifiers[deploymentInfo.Uid] = &ResourceIdentifier{
				Uid: deploymentInfo.Uid,
				Md5: sumData,
				// 当前是否存在；
				Curr: true,
				name: deploymentInfo.Name,
			}
		}
	}
	return deploymentInfo
}

func (collector *DeploymentCollector) reportNotExistResource() {
	// old deployments
	deployments := make([]*DeploymentInfo, 0)
	collector.IdentifierLock.Lock()
	deployIdentifiers := collector.identifiers
	logrus.Debugf("[DEPLOYMENT REPORT] deployIdentifiers len: %d", len(deployIdentifiers))
	if deployIdentifiers != nil {
		for k, v := range deployIdentifiers {
			if v.Curr {
				v.Curr = false
			} else {
				if v.Cid != "" {
					deployments = append(deployments, &DeploymentInfo{
						CommonInfo: CommonInfo{
							Uid:   v.Uid,
							Cid:   v.Cid,
							Exist: false,
						},
					})
				}
				logrus.Debugf("[DEPLOYMENT REPORT] deployIdentifiers delete: %s", v.Uid)
				delete(deployIdentifiers, k)
			}
		}
	}
	collector.IdentifierLock.Unlock()
	collector.reportK8sMetric(metav1.NamespaceAll, false, deployments, len(deployments))
}

func createDeploymentListWatch(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AppsV1().Deployments(ns).List(context.TODO(), options)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AppsV1().Deployments(ns).Watch(context.TODO(), options)
		},
	}
}

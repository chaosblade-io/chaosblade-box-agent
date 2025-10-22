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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type ReplicaSetCollector struct {
	*K8sBaseCollector
	opts metav1.ListOptions
}

type ReplicaSetInfo struct {
	CommonInfo
	Namespace          string `json:"namespace,omitempty"`
	AvailableReplicas  int32  `json:"availableReplicas,omitempty"`
	Replicas           int32  `json:"replicas,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
	ReadyReplicas      int32  `json:"readyReplicas,omitempty"`
	DeploymentUid      string `json:"deploymentUid,omitempty"`
}

func NewReplicasetCollector(trans *transport.TransportClient, k8sChannel *kubernetes.Channel, opts metav1.ListOptions) *ReplicaSetCollector {
	uri, ok := transport.TransportUriMap[transport.API_K8S_REPLICASET]
	if !ok {
		return nil
	}
	collector := createK8sBaseCollector(kubernetes.NodeResource, k8sChannel, trans, uri)

	return &ReplicaSetCollector{
		K8sBaseCollector: collector,
		opts:             opts,
	}
}

func (collector *ReplicaSetCollector) Report() {
	if collector.indexer == nil {
		// 需要构建reflector
		if collector.k8sChannel.ClientSet == nil {
			logrus.Warnf("[REPLICASET REPORT] k8s client not enable")
			return
		}

		collector.indexer = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &appsv1.ReplicaSet{}, collector.opts, createReplicaSetListWatch)
	}

	infos, err := collector.getReplicaSetInfo()
	if err != nil {
		logrus.Errorf("[REPLICASET REPORT] get replicaSet failed, %v", err)
		return
	}
	collector.reportK8sMetric(metav1.NamespaceAll, true, infos, len(infos))
	collector.reportNotExistResource()
}

// getReplicaSetInfo
func (collector *ReplicaSetCollector) getReplicaSetInfo() ([]*ReplicaSetInfo, error) {
	list := collector.indexer.List()
	logrus.Debugf("[REPLICASET REPORT] get replicasets from lister, size: %d", len(list))
	replicaSets := make([]*ReplicaSetInfo, 0)
	for _, r := range list {
		rs := r.(*appsv1.ReplicaSet)
		replicaSetInfo := &ReplicaSetInfo{
			CommonInfo: CommonInfo{
				Uid:         string(rs.UID),
				Name:        rs.Name,
				CreatedTime: rs.CreationTimestamp.Format(time.RFC3339Nano),
				Labels:      rs.Labels,
				Exist:       true,
			},
			Namespace:          rs.Namespace,
			AvailableReplicas:  rs.Status.AvailableReplicas,
			Replicas:           rs.Status.Replicas,
			ObservedGeneration: rs.Status.ObservedGeneration,
			ReadyReplicas:      rs.Status.ReadyReplicas,
		}
		// 获取 deploymentUid
		for _, ref := range rs.OwnerReferences {
			if ref.Kind == "Deployment" {
				replicaSetInfo.DeploymentUid = string(ref.UID)
			}
		}
		// md5
		replicaSetInfo = collector.handleReplicaSetIncrement(replicaSetInfo)
		replicaSets = append(replicaSets, replicaSetInfo)
	}
	return replicaSets, nil
}

// 处理 rs 增量
func (collector *ReplicaSetCollector) handleReplicaSetIncrement(replicaSetInfo *ReplicaSetInfo) *ReplicaSetInfo {
	collector.IdentifierLock.Lock()
	defer collector.IdentifierLock.Unlock()
	sumData, err := tools.Md5sumData(replicaSetInfo)
	if err == nil {
		if v, ok := collector.identifiers[replicaSetInfo.Uid]; ok {
			if v.Md5 == sumData && v.Cid != "" {
				// 如果相等，说明数据没变，则只上报关键数据
				replicaSetInfo = &ReplicaSetInfo{
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
			collector.identifiers[replicaSetInfo.Uid] = &ResourceIdentifier{
				Uid: replicaSetInfo.Uid,
				Md5: sumData,
				// 当前是否存在；
				Curr: true,
				name: replicaSetInfo.Name,
			}
		}
	}
	return replicaSetInfo
}

func (collector *ReplicaSetCollector) reportNotExistResource() {
	// old rs
	replicaSets := make([]*ReplicaSetInfo, 0)
	collector.IdentifierLock.Lock()
	rsIdentifiers := collector.identifiers
	logrus.Debugf("[REPLICASET REPORT] rsIdentifiers len: %d", len(rsIdentifiers))
	if rsIdentifiers != nil {
		for k, v := range rsIdentifiers {
			if v.Curr {
				v.Curr = false
			} else {
				if v.Cid != "" {
					replicaSets = append(replicaSets, &ReplicaSetInfo{
						CommonInfo: CommonInfo{
							Uid:   v.Uid,
							Cid:   v.Cid,
							Exist: false,
						},
					})
				}
				logrus.Debugf("[REPLICASET REPORT] replicaSets delete: %s", v.Uid)
				delete(rsIdentifiers, k)
			}
		}
	}
	collector.IdentifierLock.Unlock()
	collector.reportK8sMetric(metav1.NamespaceAll, false, replicaSets, len(replicaSets))
}

func createReplicaSetListWatch(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AppsV1().ReplicaSets(ns).List(context.TODO(), options)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AppsV1().ReplicaSets(ns).Watch(context.TODO(), options)
		},
	}
}

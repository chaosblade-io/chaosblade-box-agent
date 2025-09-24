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

type NodeCapacity struct {
	Cpu    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

type NodeAddress struct {
	Address     string `json:"address,omitempty"`
	AddressType string `json:"type,omitempty"`
}

type VirtualNodeInfo struct {
	CommonInfo
	Role          string            `json:"role,omitempty"`
	ClusterId     string            `json:"clusterId,omitempty"`
	ClusterName   string            `json:"clusterName,omitempty"`
	NodeInfo      v1.NodeSystemInfo `json:"nodeInfo,omitempty"`
	NodeCapacity  NodeCapacity      `json:"capacity,omitempty"`
	NodeAddresses []v1.NodeAddress  `json:"addresses,omitempty"`
	Pods          []PodInfo         `json:"pods,omitempty"`
}

type VirtualNodeCollector struct {
	K8sBaseCollector
	nodeIndex cache.Indexer
	podIndex  map[string]cache.Indexer
	opts      metav1.ListOptions
}

func NewVirtualNodeCollector(trans *transport.TransportClient, k8sChannel *kubernetes.Channel, opts metav1.ListOptions) *VirtualNodeCollector {
	uri, ok := transport.TransportUriMap[transport.API_K8S_VIRTUAL_NODE]
	if !ok {
		return nil
	}
	collector := createK8sBaseCollector(kubernetes.NodeResource, k8sChannel, trans, uri)

	virtualNodeCollector := &VirtualNodeCollector{
		K8sBaseCollector: collector,
		podIndex:         map[string]cache.Indexer{},
		opts:             opts,
	}

	virtualNodeCollector.secondIdentifiers = make(map[string]*ResourceIdentifier, 0)
	return virtualNodeCollector
}

func (collector *VirtualNodeCollector) Report() {
	if collector.indexer == nil {
		// 需要构建reflector
		if collector.k8sChannel.ClientSet == nil {
			logrus.Warnf("[VIRTUALNODE REPORT] k8s client not enable")
			return
		}

		collector.indexer = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &v1.Node{}, collector.opts, createVirtualnodeListWatch)
	}

	nodes, err := collector.getVirtualNodeInfo()
	if err != nil {
		logrus.Errorf("[VIRTUALNODE REPORT] get virtual node failed, %v", err)
		return
	}
	collector.reportK8sMetric(metav1.NamespaceAll, true, nodes, len(nodes))
	collector.reportNotExistResource()
}

// getNodeInfo
func (collector *VirtualNodeCollector) getVirtualNodeInfo() ([]*VirtualNodeInfo, error) {
	list := collector.nodeIndex.List()
	logrus.Debugf("[VIRTUALNODE REPORT] get virtualnodes from lister, size: %d, listkey : %v", len(list), collector.nodeIndex.ListKeys())
	nodes := make([]*VirtualNodeInfo, 0)
	for _, n := range list {
		node := n.(*v1.Node)
		roles := findNodeRoles(node)

		nodeInfo := &VirtualNodeInfo{
			CommonInfo: CommonInfo{
				Uid:         string(node.UID),
				Name:        node.Name,
				CreatedTime: node.CreationTimestamp.Format(time.RFC3339Nano),
				Labels:      node.Labels,
				Exist:       true,
			},
			NodeInfo:      node.Status.NodeInfo,
			NodeCapacity:  getNodeCapacity(node.Status.Allocatable),
			NodeAddresses: node.Status.Addresses,
		}
		if len(roles) > 0 {
			nodeInfo.Role = strings.Join(roles, ",")
		} else {
			nodeInfo.Role = "<none>"
		}
		nodeInfo = collector.handleVirtualNodeIncrement(nodeInfo)
		pods, err := collector.getPods(node)
		if err != nil {
			logrus.Errorf("[VIRTUALNODE REPORT] get pods on %s virtual node failed, %v", node.Name, err)
		} else {
			nodeInfo.Pods = pods
		}
		nodes = append(nodes, nodeInfo)
	}
	return nodes, nil
}

func getNodeCapacity(lists v1.ResourceList) NodeCapacity {
	cpuValue := lists["cpu"]
	memoryValue := lists["memory"]
	capacity := NodeCapacity{
		Cpu:    cpuValue.String(),
		Memory: memoryValue.String(),
	}
	return capacity
}

func (collector *VirtualNodeCollector) getPods(node *v1.Node) ([]PodInfo, error) {
	// reflector for every node
	if _, ok := collector.podIndex[node.Name]; !ok {
		listopts := metav1.ListOptions{
			FieldSelector: "spec.nodeName=" + node.Name,
		}

		collector.podIndex[node.Name] = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &v1.Node{}, listopts, createVirtualnodepodListWatch)
	}
	// get list from indexer
	list := collector.podIndex[node.Name].List()
	listkey := collector.podIndex[node.Name].ListKeys()
	logrus.Debugf("[VIRTUALNODE REPORT] get pods in node : %s, pod len: %d, pod keys : %v", node.Name, len(list), listkey)
	pods := make([]PodInfo, 0)
	for _, p := range list {
		pod := p.(*v1.Pod)
		status := getPodState(pod)
		// pod uid
		podUid := string(pod.UID)
		if hash, ok := pod.Annotations["kubernetes.io/config.hash"]; ok {
			podUid = hash
		}
		podInfo := PodInfo{
			CommonInfo: CommonInfo{
				Uid:         podUid,
				Name:        pod.Name,
				CreatedTime: pod.CreationTimestamp.Format(time.RFC3339Nano),
				Labels:      pod.Labels,
				Exist:       true,
			},
			Namespace:    pod.Namespace,
			Ip:           pod.Status.PodIP,
			RestartCount: getPodRestartCount(pod),
			State:        status,
		}
		podInfo = collector.handlePodInfoIncrement(&podInfo)
		pods = append(pods, podInfo)
	}
	return pods, nil
}

// 处理 virtual 增量
func (collector *VirtualNodeCollector) handleVirtualNodeIncrement(virtualNodeInfo *VirtualNodeInfo) *VirtualNodeInfo {
	sumData, err := tools.Md5sumData(virtualNodeInfo)
	if err == nil {
		if v, ok := collector.identifiers[virtualNodeInfo.Uid]; ok {
			if v.Md5 == sumData && v.Cid != "" {
				// 如果相等，说明数据没变，则只上报关键数据
				virtualNodeInfo = &VirtualNodeInfo{
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
			collector.identifiers[virtualNodeInfo.Uid] = &ResourceIdentifier{
				Uid: virtualNodeInfo.Uid,
				Md5: sumData,
				// 当前是否存在；
				Curr: true,
				name: virtualNodeInfo.Name,
			}
		}
	}
	return virtualNodeInfo
}

func (collector *VirtualNodeCollector) reportNotExistResource() {
	// old pods
	pods := make([]PodInfo, 0)
	podIdentifiers := collector.secondIdentifiers
	logrus.Debugf("[VIRTUALNODE REPORT] virtualPodIdentifiers len: %d", len(podIdentifiers))
	if podIdentifiers != nil {
		for k, v := range podIdentifiers {
			if v.Curr {
				v.Curr = false
			} else {
				if v.Cid != "" {
					pods = append(pods, PodInfo{
						CommonInfo: CommonInfo{
							Uid:   v.Uid,
							Cid:   v.Cid,
							Exist: false,
						},
					})
				}
				logrus.Debugf("[VIRTUALNODE REPORT] virtualPodIdentifiers delete: %s", v.Uid)
				delete(podIdentifiers, k)
			}
		}
	}

	nodes := make([]*VirtualNodeInfo, 0)
	identifiers := collector.identifiers
	logrus.Debugf("[VIRTUALNODE REPORT] virtualNodeIdentifiers len: %d", len(identifiers))
	if identifiers != nil {
		for k, v := range identifiers {
			if v.Curr {
				v.Curr = false
			} else {
				if v.Cid != "" {
					nodes = append(nodes, &VirtualNodeInfo{
						CommonInfo: CommonInfo{
							Uid:   v.Uid,
							Cid:   v.Cid,
							Exist: false,
						},
						Pods: pods,
					})
				}
				logrus.Debugf("[VIRTUALNODE REPORT] virtualNodeIdentifiers delete: %s", v.Uid)
				delete(identifiers, k)
			}
		}
	}
	collector.reportK8sMetric(metav1.NamespaceAll, true, nodes, len(nodes))
}

func (collector *VirtualNodeCollector) handlePodInfoIncrement(podInfo *PodInfo) PodInfo {
	sumData, err := tools.Md5sumData(podInfo)
	if err == nil {
		if v, ok := collector.secondIdentifiers[podInfo.Uid]; ok {
			if v.Md5 == sumData && v.Cid != "" {
				// 如果相等，说明数据没变，则只上报关键数据
				podInfo = &PodInfo{
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
			collector.secondIdentifiers[podInfo.Uid] = &ResourceIdentifier{
				Uid: podInfo.Uid,
				Md5: sumData,
				// 当前是否存在；
				Curr: true,
				name: podInfo.Name,
			}
		}
	}
	return *podInfo
}

func createVirtualnodeListWatch(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Nodes().List(context.TODO(), options)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Nodes().Watch(context.TODO(), options)
		},
	}
}

func createVirtualnodepodListWatch(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Pods(ns).List(context.TODO(), options)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Pods(ns).Watch(context.TODO(), options)
		},
	}
}

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
	"fmt"
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
	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type PodCollector struct {
	*K8sBaseCollector
	serviceCollector *ServiceCollector

	opts metav1.ListOptions
}

func NewPodCollector(trans *transport.TransportClient, k8sChannel *kubernetes.Channel,
	serviceCollector *ServiceCollector, opts metav1.ListOptions,
) *PodCollector {
	uri, ok := transport.TransportUriMap[transport.API_K8S_POD]
	if !ok {
		return nil
	}
	collector := createK8sBaseCollector(kubernetes.PodResource, k8sChannel, trans, uri)
	return &PodCollector{
		K8sBaseCollector: collector,
		serviceCollector: serviceCollector,
		opts:             opts,
	}
}

func (collector *PodCollector) Report() {
	if collector.indexer == nil {
		// 需要构建reflector
		if collector.k8sChannel.ClientSet == nil {
			logrus.Warnf("[POD REPORT] k8s client not enable")
			return
		}

		collector.indexer = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &v1.Pod{}, collector.opts, createPodListWatch)
	}

	collector.serviceCollector.SetSelector()
	collector.setAgentExternalIp()
	selectors := collector.getSelectors()
	infos, err := collector.getPodInfo(selectors)
	if err != nil {
		logrus.Warnf("[POD REPORT] get pod info failed, %v", err)
		return
	}
	collector.reportK8sMetric(metav1.NamespaceAll, true, infos, len(infos))
	collector.reportNotExistResource()
}

// getSelectors, is empty
func (collector *PodCollector) getSelectors() []func(node podNode) {
	collector.serviceCollector.SelectorLock.Lock()
	defer collector.serviceCollector.SelectorLock.Unlock()
	return collector.serviceCollector.selectors
}

func (collector *PodCollector) getPodInfo(selectors []func(node podNode)) ([]*PodInfo, error) {
	podList := collector.indexer.List()
	logrus.Debugf("[POD REPORT] get pods from indexer len : %v", len(podList))
	pods := make([]*PodInfo, 0)
	for _, p := range podList {
		pod, ok := p.(*v1.Pod)
		if !ok {
			continue
		}
		status := getPodState(pod)
		// pod uid
		podUid := string(pod.UID)
		if hash, ok := pod.Annotations["kubernetes.io/config.hash"]; ok {
			podUid = hash
		}
		podInfo := &PodInfo{
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
		podInfo = collector.handleOwnerReferences(podInfo, pod.OwnerReferences)
		// handle increment
		podInfo = collector.handlePodIncrement(podInfo)
		pods = append(pods, podInfo)
		// 建立 pod 与 service 等的关系
		for idx := range selectors {
			selectors[idx](podInfo)
		}
	}
	return pods, nil
}

// 处理 pod 增量
func (collector *PodCollector) handlePodIncrement(podInfo *PodInfo) *PodInfo {
	collector.IdentifierLock.Lock()
	defer collector.IdentifierLock.Unlock()
	sumData, err := tools.Md5sumData(podInfo)
	if err == nil {
		if v, ok := collector.identifiers[podInfo.Uid]; ok {
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
			collector.identifiers[podInfo.Uid] = &ResourceIdentifier{
				Uid: podInfo.Uid,
				Md5: sumData,
				// 当前是否存在；
				Curr: true,
				name: podInfo.Name,
			}
		}
	}
	return podInfo
}

func (collector *PodCollector) reportNotExistResource() {
	// old pods
	collector.IdentifierLock.Lock()
	pods := make([]*PodInfo, 0)
	podIdentifiers := collector.identifiers
	logrus.Debugf("podIdentifiers len: %d", len(podIdentifiers))
	if podIdentifiers != nil {
		for k, v := range podIdentifiers {
			if v.Curr {
				v.Curr = false
			} else {
				if v.Cid != "" {
					pods = append(pods, &PodInfo{
						CommonInfo: CommonInfo{
							Uid:   v.Uid,
							Cid:   v.Cid,
							Exist: false,
						},
					})
				}
				logrus.Debugf("podIdentifiers delete: %s", v.Uid)
				delete(podIdentifiers, k)
			}
		}
	}
	collector.IdentifierLock.Unlock()
	collector.reportK8sMetric(metav1.NamespaceAll, false, pods, len(pods))
}

func (collector *PodCollector) handleOwnerReferences(info *PodInfo, references []metav1.OwnerReference) *PodInfo {
	for _, reference := range references {
		switch reference.Kind {
		case "ReplicaSet":
			info.ReplicaSetUid = string(reference.UID)
		case "DaemonSet":
			info.DaemonsetUid = string(reference.UID)
		}
	}
	return info
}

// NodeUnreachablePodReason is the reason on a pod when its state cannot be confirmed as kubelet is unresponsive
// on the node it is (was) running.
var NodeUnreachablePodReason = "NodeLost"

func getPodState(pod *v1.Pod) string {
	readyContainers := 0
	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}
	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
				readyContainers++
			}
		}
		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			reason = "Running"
		}
	}
	if pod.DeletionTimestamp != nil && pod.Status.Reason == NodeUnreachablePodReason {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}
	return reason
}

func createPodListWatch(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Pods(ns).List(context.TODO(), options)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Pods(ns).Watch(context.TODO(), options)
		},
	}
}

// PodInfo
type PodInfo struct {
	CommonInfo
	Namespace     string `json:"namespace,omitempty"`
	Ip            string `json:"ip,omitempty"`
	RestartCount  int32  `json:"restartCount,omitempty"`
	State         string `json:"state,omitempty"`
	DaemonsetUid  string `json:"daemonsetUid,omitempty"`
	DaemonsetCid  string `json:"daemonsetCid,omitempty"`
	ServiceUid    string `json:"serviceUid,omitempty"`
	ServiceCid    string `json:"serviceCid,omitempty"`
	DeploymentUid string `json:"deploymentUid,omitempty"`
	DeploymentCid string `json:"deploymentCid,omitempty"`
	ReplicaSetUid string `json:"replicasetUid,omitempty"`
	ReplicaSetCid string `json:"replicasetCid,omitempty"`
}

func (pi *PodInfo) namespace() string {
	return pi.Namespace
}

func (pi *PodInfo) labels() map[string]string {
	return pi.Labels
}

func (pi *PodInfo) addLink(resource string, uid string) {
	// 获取缓存的 cid
	switch resource {
	case kubernetes.ServiceResource:
		pi.ServiceUid = uid
	case kubernetes.DeploymentResource:
		pi.DeploymentUid = uid
	case kubernetes.ReplicaSetResource:
		pi.ReplicaSetUid = uid
	case kubernetes.DaemonsetResource:
		pi.DaemonsetUid = uid
	}
}

func (collector *PodCollector) setAgentExternalIp() {
	serviceInfos, _, err := collector.serviceCollector.getServiceInfo()
	if err != nil {
		logrus.Warnf("get service info failed, err: %v", err)
		return
	}

	for _, serviceInfo := range serviceInfos {
		if serviceInfo.Name != DefaultAgentServiceName {
			continue
		}

		externalIp := serviceInfo.ExternalIp
		if externalIp == "" {
			logrus.Debugf("Service %s has empty ExternalIP, skip setting", DefaultAgentServiceName)
			continue
		}

		// 处理无效IP
		invalidIPs := []string{"<none>", "<pending>", "<unknown>"}
		for _, invalid := range invalidIPs {
			if externalIp == invalid {
				logrus.Warnf("Service %s has invalid ExternalIP: %s, skip setting", DefaultAgentServiceName, externalIp)
				return
			}
		}

		// 处理多个IP的情况，取第一个有效IP
		ips := strings.Split(externalIp, ",")
		if len(ips) > 0 {
			validIP := strings.TrimSpace(ips[0])
			if validIP != "" {
				// 再次检查是否为无效IP（可能出现在逗号分隔的列表中）
				isInvalid := false
				for _, invalid := range invalidIPs {
					if validIP == invalid {
						isInvalid = true
						break
					}
				}
				if !isInvalid {
					options.Opts.Ip = validIP
					if len(ips) > 1 {
						logrus.Infof("Set agent ExternalIP to: %s (from %d IPs: %s)", validIP, len(ips), externalIp)
					} else {
						logrus.Infof("Set agent ExternalIP to: %s", validIP)
					}
					return
				}
			}
		}

		logrus.Warnf("Service %s ExternalIP is invalid or empty: %s", DefaultAgentServiceName, externalIp)
	}
}

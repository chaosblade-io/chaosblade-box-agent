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

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type NodeInfo struct {
	Uid         string `json:"uid"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	ClusterId   string `json:"clusterId"`
	ClusterName string `json:"clusterName"`
}

type NodeCollector struct {
	K8sBaseCollector
	opts metav1.ListOptions
}

func NewNodeCollector(trans *transport.TransportClient, k8sChannel *kubernetes.Channel, opts metav1.ListOptions) *NodeCollector {
	uri, ok := transport.TransportUriMap[transport.API_K8S_NODE]
	if !ok {
		return nil
	}
	collector := createK8sBaseCollector(kubernetes.NodeResource, k8sChannel, trans, uri)

	return &NodeCollector{
		collector,
		opts,
	}
}

func (collector *NodeCollector) Report() {
	if collector.indexer == nil {
		// 需要构建reflector
		if collector.k8sChannel.ClientSet == nil {
			logrus.Warnf("[NODE REPORT] k8s client not enable")
			return
		}

		collector.indexer = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &v1.Node{}, collector.opts, createNodeListWatch)
	}

	nodes, err := collector.getNodeInfo()
	if err != nil {
		logrus.Errorf("[NODE REPORT] get node failed, %v", err)
		return
	}
	collector.reportK8sMetric(metav1.NamespaceAll, true, nodes, len(nodes))
}

// getNodeInfo
func (collector *NodeCollector) getNodeInfo() ([]*NodeInfo, error) {
	list := collector.indexer.List()
	logrus.Debugf("[NODE REPORT] get nodes from lister, size: %d, listkey : %v", len(list), collector.indexer.ListKeys())
	nodes := make([]*NodeInfo, 0)
	for _, n := range list {
		node := n.(*v1.Node)

		roles := findNodeRoles(node)
		nodeInfo := &NodeInfo{
			Uid:       string(node.UID),
			Name:      node.Name,
			ClusterId: options.Opts.ClusterId,
			// ClusterName: node.ClusterName,
		}
		nodes = append(nodes, nodeInfo)
		if len(roles) > 0 {
			nodeInfo.Role = strings.Join(roles, ",")
		} else {
			nodeInfo.Role = "<none>"
		}
		break
	}
	return nodes, nil
}

// labelNodeRolePrefix is a label prefix for node roles
// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
const (
	labelNodeRolePrefix = "node-role.kubernetes.io/"
	// nodeLabelRole specifies the role of a node
	nodeLabelRole = "kubernetes.io/role"
)

// findNodeRoles returns the roles of a given node.
func findNodeRoles(node *v1.Node) []string {
	roles := sets.NewString()
	for k, v := range node.Labels {
		switch {
		case strings.HasPrefix(k, labelNodeRolePrefix):
			if role := strings.TrimPrefix(k, labelNodeRolePrefix); len(role) > 0 {
				roles.Insert(role)
			}
		case k == nodeLabelRole && v != "":
			roles.Insert(v)
		}
	}
	return roles.List()
}

func createNodeListWatch(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Nodes().List(context.TODO(), options)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Nodes().Watch(context.TODO(), options)
		},
	}
}

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
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/chaosblade-io/chaos-agent/pkg/options"
)

const (
	PodResource         = "pods"
	ServiceResource     = "services"
	DeploymentResource  = "deployments"
	DaemonsetResource   = "daemonsets"
	NamespaceResource   = "namespaces"
	ReplicaSetResource  = "replicasets"
	NodeResource        = "nodes"
	IngressResource     = "ingresses"
	VirtualNodeResource = "virtualNodes"
)

var (
	channel *Channel
	once    sync.Once
)

type Channel struct {
	ClientSet *kubernetes.Clientset
}

func GetInstance() *Channel {
	once.Do(
		func() {
			if channel == nil {
				channel = &Channel{}
			}

			clientset, err := NewK8sClient()
			if err != nil {
				logrus.Warningf("create k8s client err, %s", err.Error())
				return
			}
			channel = &Channel{
				ClientSet: clientset,
			}
		},
	)
	return channel
}

func NewK8sClient() (*kubernetes.Clientset, error) {
	defaultClusterId := "default-cluster"
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	if clusterConfig.Host != "" {
		defaultClusterId = fmt.Sprintf("API_SERVER_%s", clusterConfig.Host)
	}

	options.Opts.SetClusterIdIfNotPresent(defaultClusterId)
	clientset, err := kubernetes.NewForConfig(clusterConfig)
	return clientset, err
}

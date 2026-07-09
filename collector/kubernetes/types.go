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
	"errors"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	errLabelsAllowListFormat = errors.New("invalid format, metric=[label1,label2,labeln...],metricN=[]")
	AllListNs                = NamespaceList{metav1.NamespaceAll}
)

type NamespaceList []string

func (n *NamespaceList) String() string {
	return strings.Join(*n, ",")
}

func (n *NamespaceList) Set(value string) error {
	splitNamespaces := strings.Split(value, ",")
	for _, ns := range splitNamespaces {
		ns = strings.TrimSpace(ns)
		if len(ns) != 0 {
			*n = append(*n, ns)
		}
	}
	return nil
}

type listWatchFunc func(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher

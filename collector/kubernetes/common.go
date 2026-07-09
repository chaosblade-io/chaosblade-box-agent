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
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
)

func servicePortsToString(ports []v1.ServicePort) []string {
	ps := make([]string, 0)
	for _, port := range ports {
		// "32242->80/TCP/http"
		ps = append(ps, fmt.Sprintf("%d->%d/%s/%s", port.NodePort, port.Port, port.Protocol, port.Name))
	}
	return ps
}

func getServiceExternalIP(svc *v1.Service, wide bool) string {
	switch svc.Spec.Type {
	case v1.ServiceTypeClusterIP:
		if len(svc.Spec.ExternalIPs) > 0 {
			return strings.Join(svc.Spec.ExternalIPs, ",")
		}
		return "<none>"
	case v1.ServiceTypeNodePort:
		if len(svc.Spec.ExternalIPs) > 0 {
			return strings.Join(svc.Spec.ExternalIPs, ",")
		}
		return "<none>"
	case v1.ServiceTypeLoadBalancer:
		lbIps := loadBalancerStatusStringer(svc.Status.LoadBalancer, wide)
		if len(svc.Spec.ExternalIPs) > 0 {
			results := []string{}
			if len(lbIps) > 0 {
				results = append(results, strings.Split(lbIps, ",")...)
			}
			results = append(results, svc.Spec.ExternalIPs...)
			return strings.Join(results, ",")
		}
		if len(lbIps) > 0 {
			return lbIps
		}
		return "<pending>"
	case v1.ServiceTypeExternalName:
		return svc.Spec.ExternalName
	}
	return "<unknown>"
}

const (
	loadBalancerWidth = 16
	// DefaultAgentServiceName is the default service name for chaos-agent
	DefaultAgentServiceName = "chaos-agent"
)

// loadBalancerStatusStringer behaves mostly like a string interface and converts the given status to a string.
// `wide` indicates whether the returned value is meant for --o=wide output. If not, it's clipped to 16 bytes.
func loadBalancerStatusStringer(s v1.LoadBalancerStatus, wide bool) string {
	ingress := s.Ingress
	result := sets.NewString()
	for i := range ingress {
		if ingress[i].IP != "" {
			result.Insert(ingress[i].IP)
		} else if ingress[i].Hostname != "" {
			result.Insert(ingress[i].Hostname)
		}
	}

	r := strings.Join(result.List(), ",")
	if !wide && len(r) > loadBalancerWidth {
		r = r[0:(loadBalancerWidth-3)] + "..."
	}
	return r
}

func selectorMatch(namespace string, selector labels.Selector, resource, uid string) func(podNode) {
	return func(c podNode) {
		// 如果不是全量数据，则 podNode 中不存在 namespace，所以不会记 link 关系
		if namespace == c.namespace() && selector.Matches(labels.Set(c.labels())) {
			c.addLink(resource, uid)
		}
	}
}

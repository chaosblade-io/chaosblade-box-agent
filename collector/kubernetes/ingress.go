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
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type IngressInfo struct {
	CommonInfo
	Namespace   string               `json:"Namespace,omitempty"`
	Address     string               `json:"Address,omitempty"`
	Annotations map[string]string    `json:"Annotations,omitempty"`
	Tls         []v1beta1.IngressTLS `json:"Tls,omitempty"`
	Rules       []IngressRule        `json:"Rules,omitempty"`
}

type IngressCollector struct {
	K8sBaseCollector
	opts metav1.ListOptions
}

func NewIngressCollector(trans *transport.TransportClient, k8sChannel *kubernetes.Channel, opts metav1.ListOptions) *IngressCollector {
	uri, ok := transport.TransportUriMap[transport.API_K8S_INGRESS]
	if !ok {
		return nil
	}
	collector := createK8sBaseCollector(kubernetes.IngressResource, k8sChannel, trans, uri)
	return &IngressCollector{
		K8sBaseCollector: collector,
		opts:             opts,
	}
}

func (collector *IngressCollector) Report() {
	if collector.indexer == nil {
		// 需要构建reflector
		if collector.k8sChannel.ClientSet == nil {
			logrus.Warnf("[INGRESS REPORT] k8s client not enable")
			return
		}

		collector.indexer = reflectorPreNamespace(AllListNs, collector.k8sChannel.ClientSet, collector.Ctx, &v1beta1.Ingress{},
			collector.opts, createIngressListWatch)
	}

	infos, err := collector.getIngressInfo()
	if err != nil {
		logrus.Warnf("[INGRESS REPORT] get daemonset info failed, %v", err)
		return
	}
	collector.reportK8sMetric(metav1.NamespaceAll, true, infos, len(infos))
	collector.reportNotExistResource()
}

func (collector *IngressCollector) reportNotExistResource() {
	// old rs
	ingresses := make([]*IngressInfo, 0)
	collector.IdentifierLock.Lock()
	ingIdentifiers := collector.identifiers
	logrus.Debugf("[INGRESS REPORT] ingIdentifiers len: %d", len(ingIdentifiers))
	if ingIdentifiers != nil {
		for k, v := range ingIdentifiers {
			if v.Curr {
				v.Curr = false
			} else {
				// 如果不存在，
				if v.Cid != "" {
					ingresses = append(ingresses, &IngressInfo{
						CommonInfo: CommonInfo{
							Uid:   v.Uid,
							Cid:   v.Cid,
							Exist: false,
						},
					})
				}
				logrus.Debugf("[INGRESS REPORT] ingressIdentifiers delete: %s", v.Uid)
				delete(ingIdentifiers, k)
			}
		}
	}
	collector.IdentifierLock.Unlock()
	collector.reportK8sMetric(metav1.NamespaceAll, false, ingresses, len(ingresses))
}

// getIngressInfo
func (collector *IngressCollector) getIngressInfo() ([]*IngressInfo, error) {
	list := collector.indexer.List()
	logrus.Debugf("[INGRESS REPORT] get ingress from lister, size: %d", len(list))
	ingresses := make([]*IngressInfo, 0)
	for _, i := range list {
		ing := i.(*v1beta1.Ingress)
		ingressInfo := &IngressInfo{
			CommonInfo: CommonInfo{
				Uid:         string(ing.UID),
				Name:        ing.Name,
				CreatedTime: ing.CreationTimestamp.Format(time.RFC3339Nano),
				Labels:      ing.Labels,
				Exist:       true,
			},
			Namespace:   ing.Namespace,
			Tls:         ing.Spec.TLS,
			Annotations: ing.Annotations,
		}
		// generate address
		balancerIngresses := ing.Status.LoadBalancer.Ingress
		if len(balancerIngresses) > 0 {
			ipSet := tools.NewSet()

			for _, b := range balancerIngresses {
				if b.IP != "" {
					ipSet.Add(b.IP)
				} else if b.Hostname != "" {
					ipSet.Add(b.Hostname)
				}
			}
			ingressInfo.Address = strings.Join(ipSet.StringKeys(), ",")
		}
		ingressInfo.Rules = collector.generateIngressRules(ing)
		// handle increment
		ingressInfo = collector.handleIngressIncrement(ingressInfo)
		ingresses = append(ingresses, ingressInfo)
	}
	return ingresses, nil
}

func (collector *IngressCollector) generateIngressRules(ingress *v1beta1.Ingress) []IngressRule {
	rules := make([]IngressRule, 0)
	for _, r := range ingress.Spec.Rules {
		rule := IngressRule{Host: r.Host}
		paths := make([]HTTPIngressPath, 0)
		if r.HTTP != nil {
			for _, p := range r.HTTP.Paths {
				path := HTTPIngressPath{Path: p.Path}
				path.Backend = IngressBackend{
					ServiceName: p.Backend.ServiceName,
					ServicePort: getServicePort(p.Backend.ServicePort),
					// get service uid from service cache
					ServiceUid: collector.getServiceUidByName(p.Backend.ServiceName),
				}
				paths = append(paths, path)
				// log error
				if path.Backend.ServiceUid == "" {
					logrus.Warningf("%s ingress cannot get the service uid, service name: %s", ingress.Name, p.Backend.ServiceName)
				}
			}
		}
		httpRule := &HTTPIngressRuleValue{
			Paths: paths,
		}
		rule.IngressRuleValue.HTTP = httpRule
		rules = append(rules, rule)
	}
	return rules
}

// handleIngressIncrement
func (collector *IngressCollector) handleIngressIncrement(ingressInfo *IngressInfo) *IngressInfo {
	collector.IdentifierLock.Lock()
	defer collector.IdentifierLock.Unlock()
	sumData, err := tools.Md5sumData(ingressInfo)
	if err == nil {
		if v, ok := collector.identifiers[ingressInfo.Uid]; ok {
			if v.Md5 == sumData && v.Cid != "" {
				// 如果相等，说明数据没变，则只上报关键数据
				ingressInfo = &IngressInfo{
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
			collector.identifiers[ingressInfo.Uid] = &ResourceIdentifier{
				Uid: ingressInfo.Uid,
				Md5: sumData,
				// 当前是否存在；
				Curr: true,
				name: ingressInfo.Name,
			}
		}
	}
	return ingressInfo
}

type IngressRule struct {
	// Host is the fully qualified domain name of a network host, as defined
	// by RFC 3986. Note the following deviations from the "host" part of the
	// URI as defined in the RFC:
	// 1. IPs are not allowed. Currently an IngressRuleValue can only apply to the
	//	  IP in the Spec of the parent Ingress.
	// 2. The `:` delimiter is not respected because ports are not allowed.
	//	  Currently the port of an Ingress is implicitly :80 for http and
	//	  :443 for https.
	// Both these may change in the future.
	// Incoming requests are matched against the host before the IngressRuleValue.
	// If the host is unspecified, the Ingress routes all traffic based on the
	// specified IngressRuleValue.
	// +optional
	Host string `json:"host,omitempty" protobuf:"bytes,1,opt,name=host"`
	// IngressRuleValue represents a rule to route requests for this IngressRule.
	// If unspecified, the rule defaults to a http catch-all. Whether that sends
	// just traffic matching the host to the default backend or all traffic to the
	// default backend, is left to the controller fulfilling the Ingress. Http is
	// currently the only supported IngressRuleValue.
	// +optional
	IngressRuleValue `json:",inline,omitempty" protobuf:"bytes,2,opt,name=ingressRuleValue"`
}

type IngressRuleValue struct {
	// TODO:
	// 1. Consider renaming this resource and the associated rules so they
	// aren't tied to Ingress. They can be used to route intra-cluster traffic.
	// 2. Consider adding fields for ingress-type specific global options
	// usable by a loadbalancer, like http keep-alive.

	// +optional
	HTTP *HTTPIngressRuleValue `json:"http,omitempty" protobuf:"bytes,1,opt,name=http"`
}

// HTTPIngressRuleValue is a list of http selectors pointing to backends.
// In the example: http://<host>/<path>?<searchpart> -> backend where
// where parts of the url correspond to RFC 3986, this resource will be used
// to match against everything after the last '/' and before the first '?'
// or '#'.
type HTTPIngressRuleValue struct {
	// A collection of paths that map requests to backends.
	Paths []HTTPIngressPath `json:"paths" protobuf:"bytes,1,rep,name=paths"`
	// TODO: Consider adding fields for ingress-type specific global
	// options usable by a loadbalancer, like http keep-alive.
}

// HTTPIngressPath associates a path regex with a backend. Incoming urls matching
// the path are forwarded to the backend.
type HTTPIngressPath struct {
	// Path is an extended POSIX regex as defined by IEEE Std 1003.1,
	// (i.e this follows the egrep/unix syntax, not the perl syntax)
	// matched against the path of an incoming request. Currently it can
	// contain characters disallowed from the conventional "path"
	// part of a URL as defined by RFC 3986. Paths must begin with
	// a '/'. If unspecified, the path defaults to a catch all sending
	// traffic to the backend.
	// +optional
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`

	// Backend defines the referenced service endpoint to which the traffic
	// will be forwarded to.
	Backend IngressBackend `json:"backend" protobuf:"bytes,2,opt,name=backend"`
}

// IngressBackend describes all endpoints for a given service and port.
type IngressBackend struct {
	// Specifies the name of the referenced service.
	ServiceName string `json:"serviceName" protobuf:"bytes,1,opt,name=serviceName"`

	// Specifies the port of the referenced service.
	ServicePort string `json:"servicePort" protobuf:"bytes,2,opt,name=servicePort"`

	// Added for chaos service
	ServiceUid string `json:"serviceUid"`
}

func createIngressListWatch(kubeClient clientset.Interface, ns string, options metav1.ListOptions) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.ExtensionsV1beta1().Ingresses(ns).List(context.TODO(), options)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.ExtensionsV1beta1().Ingresses(ns).Watch(context.TODO(), options)
		},
	}
}

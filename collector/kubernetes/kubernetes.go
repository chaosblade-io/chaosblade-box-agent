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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type K8sBaseCollector struct {
	resourceName string
	indexer      cache.Indexer
	k8sChannel   *kubernetes.Channel

	transport *transport.TransportClient

	Ctx context.Context

	identifiers       map[string]*ResourceIdentifier
	secondIdentifiers map[string]*ResourceIdentifier
	IdentifierLock    sync.Mutex

	// for pod link
	selectors []func(node podNode)

	// report
	uri           transport.Uri
	ReportHandler string
}

var LocalNodeName string

func createK8sBaseCollector(resourceName string, k8sChannel *kubernetes.Channel, transport *transport.TransportClient, uri transport.Uri) K8sBaseCollector {
	return K8sBaseCollector{
		resourceName:      resourceName,
		k8sChannel:        k8sChannel,
		transport:         transport,
		Ctx:               context.TODO(),
		identifiers:       make(map[string]*ResourceIdentifier, 0),
		secondIdentifiers: make(map[string]*ResourceIdentifier, 0),
		IdentifierLock:    sync.Mutex{},

		uri:           uri,
		ReportHandler: uri.HandlerName,
	}
}

func (collector *K8sBaseCollector) ResourceName() string {
	return collector.resourceName
}

// getServiceUidByName returns the service uid
func (collector *K8sBaseCollector) getServiceUidByName(serviceName string) string {
	for k, v := range collector.identifiers {
		if v.name == serviceName {
			return k
		}
	}
	return ""
}

// report to server
func (collector *K8sBaseCollector) reportK8sMetric(namespace string, isExists bool, resource interface{}, size int) {
	if size == 0 {
		return
	}
	request := transport.NewRequest()
	// pods
	bytes, err := json.Marshal(resource)
	if err != nil {
		logrus.Warningf("marshal k8s %s err, %s", collector.ResourceName(), err.Error())
	} else {
		request.AddParam(collector.ResourceName(), string(bytes))
	}

	logrus.Debugf("kubernetes %s resource in %s request: %+v", collector.ReportHandler, namespace, request)
	uri := collector.uri
	// 开启压缩
	uri.CompressVersion = fmt.Sprintf("%d", transport.AllCompress)
	response, err := collector.transport.Invoke(uri, request, true) // todo 这里看下是否用这个struct
	if err != nil {
		collector.resetIdentifierCache()
		logrus.Warningf("Report kubernetes %s infos err: %v", collector.ReportHandler, err)
		return
	}
	if !response.Success {
		collector.resetIdentifierCache()
		logrus.Warningf("Report kubernetes %s infos failed: %v", collector.ReportHandler, response.Error)
		return
	}
	if isExists {
		result := response.Result
		logrus.Debugf("Report kubernetes resource %s response %+v", collector.ReportHandler, result)
		v, ok := result.(map[string]interface{})
		if !ok {
			collector.resetIdentifierCache()
			logrus.Warningf("kubernetes %s response is not map[string]", collector.ReportHandler)
			return
		}
		// 对 virtual node 单独做处理
		if collector.ReportHandler == transport.K8sVirtualNode {
			virtualNodeCids := v[kubernetes.VirtualNodeResource]
			if virtualNodeCids != nil {
				vnCids, ok := virtualNodeCids.(map[string]interface{})
				if !ok {
					collector.resetIdentifierCache()
					logrus.Warningf("kubernetes %s response is not map[string]", collector.ReportHandler)
					return
				}
				for key, value := range vnCids {
					// add cid to cache
					if vi, ok := collector.identifiers[key]; ok {
						vi.Cid = value.(string)
					}
				}
			}
			podsCids := v[kubernetes.PodResource]
			if podsCids != nil {
				pCids := podsCids.(map[string]interface{})
				if !ok {
					collector.resetIdentifierCache()
					logrus.Warningf("kubernetes %s response is not map[string]", collector.ReportHandler)
					return
				}
				for key, value := range pCids {
					// add cid to cache
					if vi, ok := collector.secondIdentifiers[key]; ok {
						vi.Cid = value.(string)
					}
				}
			}
		} else {
			for key, value := range v {
				// add cid to cache
				if vi, ok := collector.identifiers[key]; ok {
					vi.Cid = value.(string)
				}
			}
		}
		logrus.Infof("Report kubernetes resources success, %s, ns: %s, size: %d", collector.ReportHandler, namespace, size)
	} else {
		logrus.Infof("Report old kubernetes resources success, %s, ns: %s, size: %d", collector.ReportHandler, namespace, size)
	}
}

func (collector *K8sBaseCollector) resetIdentifierCache() {
	collector.IdentifierLock.Lock()
	defer collector.IdentifierLock.Unlock()
	collector.identifiers = make(map[string]*ResourceIdentifier, 0)
	if collector.secondIdentifiers != nil {
		collector.secondIdentifiers = make(map[string]*ResourceIdentifier, 0)
	}
}

type ResourceIdentifier struct {
	Uid string
	Cid string
	Md5 string
	// 当前是否存在；
	Curr bool
	// 重要：此字段目前只做缓存使用，不发往后端
	name string
}

type CommonInfo struct {
	Uid         string            `json:"uid"`
	Name        string            `json:"name"`
	CreatedTime string            `json:"createdTime"`
	Labels      map[string]string `json:"labels,omitempty"`
	Exist       bool              `json:"exist"`
	Cid         string            `json:"cid,omitempty"`
}

type podNode interface {
	namespace() string
	labels() map[string]string
	addLink(string, string)
}

// multiListerWatcher
type multiListerWatcher []cache.ListerWatcher

// List implements the ListerWatcher interface.
// It combines the output of the List method of every ListerWatcher into
// a single result.
func (mlw multiListerWatcher) List(options metav1.ListOptions) (runtime.Object, error) {
	l := metav1.List{}
	var resourceVersions []string
	for _, lw := range mlw {
		list, err := lw.List(options)
		if err != nil {
			return nil, err
		}
		items, err := meta.ExtractList(list)
		if err != nil {
			return nil, err
		}
		metaObj, err := meta.ListAccessor(list)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			l.Items = append(l.Items, runtime.RawExtension{Object: item.DeepCopyObject()})
		}
		resourceVersions = append(resourceVersions, metaObj.GetResourceVersion())
	}
	// Combine the resource versions so that the composite Watch method can
	// distribute appropriate versions to each underlying Watch func.
	l.ListMeta.ResourceVersion = strings.Join(resourceVersions, "/")
	return &l, nil
}

// Watch implements the ListerWatcher interface.
// It returns a watch.Interface that combines the output from the
// watch.Interface of every cache.ListerWatcher into a single result chan.
func (mlw multiListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	resourceVersions := make([]string, len(mlw))
	// Allow resource versions to be "".
	if options.ResourceVersion != "" {
		rvs := strings.Split(options.ResourceVersion, "/")
		if len(rvs) != len(mlw) {
			return nil, fmt.Errorf("expected resource version to have %d parts to match the number of ListerWatchers", len(mlw))
		}
		resourceVersions = rvs
	}
	return newMultiWatch(mlw, resourceVersions, options)
}

// multiWatch abstracts multiple watch.Interface's, allowing them
// to be treated as a single watch.Interface.
type multiWatch struct {
	result   chan watch.Event
	stopped  chan struct{}
	stoppers []func()
}

// newMultiWatch returns a new multiWatch or an error if one of the underlying
// Watch funcs errored. The length of []cache.ListerWatcher and []string must
// match.
func newMultiWatch(lws []cache.ListerWatcher, resourceVersions []string, options metav1.ListOptions) (*multiWatch, error) {
	var (
		result   = make(chan watch.Event)
		stopped  = make(chan struct{})
		stoppers []func()
		wg       sync.WaitGroup
	)

	wg.Add(len(lws))

	for i, lw := range lws {
		o := options.DeepCopy()
		o.ResourceVersion = resourceVersions[i]
		w, err := lw.Watch(*o)
		if err != nil {
			return nil, err
		}

		go func() {
			defer wg.Done()

			for {
				event, ok := <-w.ResultChan()
				if !ok {
					return
				}

				select {
				case result <- event:
				case <-stopped:
					return
				}
			}
		}()
		stoppers = append(stoppers, w.Stop)
	}

	// result chan must be closed,
	// once all event sender goroutines exited.
	go func() {
		wg.Wait()
		close(result)
	}()

	return &multiWatch{
		result:   result,
		stoppers: stoppers,
		stopped:  stopped,
	}, nil
}

// ResultChan implements the watch.Interface interface.
func (mw *multiWatch) ResultChan() <-chan watch.Event {
	return mw.result
}

// Stop implements the watch.Interface interface.
// It stops all of the underlying watch.Interfaces and closes the backing chan.
// Can safely be called more than once.
func (mw *multiWatch) Stop() {
	select {
	case <-mw.stopped:
		// nothing to do, we are already stopped
	default:
		for _, stop := range mw.stoppers {
			stop()
		}
		close(mw.stopped)
	}
	return
}

// common function
func getPodRestartCount(pod *v1.Pod) int32 {
	count := int32(0)
	for _, c := range pod.Status.ContainerStatuses {
		count += c.RestartCount
	}
	return count
}

func reflectorPreNamespace(ns []string, clientset *k8s.Clientset, ctx context.Context, expectedType interface{}, options metav1.ListOptions, listWatchFunc listWatchFunc) cache.Indexer {
	lwf := func(ns string) cache.ListerWatcher {
		return listWatchFunc(clientset, ns, options)
	}

	lw := MultiNamespaceListerWatcher(ns, lwf)
	indexer, reflector := cache.NewNamespaceKeyedIndexerAndReflector(lw, expectedType, 0)
	go reflector.Run(ctx.Done())
	return indexer
}

func MultiNamespaceListerWatcher(allowedNamespaces []string, f func(string) cache.ListerWatcher) cache.ListerWatcher {
	// If there is only one namespace then there is no need to create a
	// multi lister watcher proxy.
	if IsAllNamespaces(allowedNamespaces) || len(allowedNamespaces) == 1 {
		return f(allowedNamespaces[0])
	}

	var lws []cache.ListerWatcher
	for _, n := range allowedNamespaces {
		lws = append(lws, f(n))
	}
	return multiListerWatcher(lws)
}

// IsAllNamespaces checks if the given slice of namespaces
// contains only v1.NamespaceAll.
func IsAllNamespaces(namespaces []string) bool {
	return len(namespaces) == 1 && namespaces[0] == v1.NamespaceAll
}

// getServicePort returns service port
func getServicePort(servicePort intstr.IntOrString) string {
	switch servicePort.Type {
	case intstr.String:
		return servicePort.StrVal
	case intstr.Int:
		return strconv.Itoa(int(servicePort.IntVal))
	}
	return ""
}

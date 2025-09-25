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

package litmuschaos

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/chaosblade-io/chaos-agent/pkg/options"
)

type ChaosEngine struct {
	Kind       string `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	APIVersion string `json:"apiVersion,omitempty" protobuf:"bytes,2,opt,name=apiVersion"`
	ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec       ChaosEngineSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type ObjectMeta struct {
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	GenerateName string `json:"generateName,omitempty" protobuf:"bytes,2,opt,name=generateName"`

	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
}

type ChaosEngineSpec struct {
	Appinfo             map[string]string `json:"appinfo,omitempty" protobuf:"bytes,3,opt,name=appinfo"`
	AnnotationCheck     string            `json:"annotationCheck" protobuf:"bytes,3,opt,name=annotationCheck"`
	EngineState         string            `json:"engineState" protobuf:"bytes,3,opt,name=engineState"`
	AuxiliaryAppInfo    string            `json:"auxiliaryAppInfo,omitempty"  protobuf:"bytes,3,opt,name=auxiliaryAppInfo"`
	ChaosServiceAccount string            `json:"chaosServiceAccount,omitempty"`
	JobCleanUpPolicy    string            `json:"jobCleanUpPolicy,omitempty"`
	Experiments         []SpecExperiment  `json:"experiments,omitempty"`
}

type SpecExperiment struct {
	Name string         `json:"name,omitempty"`
	Spec ExperimentSpec `json:"spec,omitempty"`
}
type ExperimentSpec struct {
	Components Components `json:"components,omitempty"`
}
type Components struct {
	Env []ComponentsEnv `json:"env,omitempty"`
}

// ChaosExperiment
type ChaosExperiment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChaosExperimentSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec" yaml:"spec"`
	Status ChaosExperimentStatus `json:"status,omitempty"`
}

// ChaosExperimentStatus defines the observed state of ChaosExperiment
// +k8s:openapi-gen=true
type ChaosExperimentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

type ChaosExperimentSpec struct {
	Definition Definition `json:"definition" protobuf:"bytes,2,opt,name=definition" yaml:"definition"`
}

type Definition struct {
	// Default labels of the runner pod
	// +optional
	Labels map[string]string `json:"labels"`
	// Image of the chaos experiment
	Image string `json:"image"`
	// ImagePullPolicy of the chaos experiment container
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Scope specifies the service account scope (& thereby blast radius) of the experiment
	Scope string `json:"scope"`
	// List of Permission needed for a service account to execute experiment
	Permissions []rbacV1.PolicyRule `json:"permissions"`
	// List of ENV vars passed to executor pod
	ENVList []corev1.EnvVar `json:"env"`
	// Defines command to invoke experiment
	Command []string `json:"command"`
	// Defines arguments to runner's entrypoint command
	Args []string `json:"args"`
	// ConfigMaps contains a list of ConfigMaps
	ConfigMaps []ConfigMap `json:"configMaps,omitempty"`
	// Secrets contains a list of Secrets
	Secrets []Secret `json:"secrets,omitempty"`
	// HostFileVolume defines the host directory/file to be mounted
	HostFileVolumes []HostFile `json:"hostFileVolumes,omitempty"`
	// Annotations that needs to be provided in the pod for pod that is getting created
	ExperimentAnnotations map[string]string `json:"experimentAnnotations,omitempty"`
	// SecurityContext holds security configuration that will be applied to a container
	SecurityContext SecurityContext `json:"securityContext,omitempty"`
	// HostPID is need to be provided in the chaospod
	HostPID bool `json:"hostPID,omitempty"`
}

// ConfigMap is an simpler implementation of corev1.ConfigMaps, needed for experiments
type ConfigMap struct {
	Data      map[string]string `json:"data,omitempty"`
	Name      string            `json:"name"`
	MountPath string            `json:"mountPath"`
}

// Secret is an simpler implementation of corev1.Secret
type Secret struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
}

// HostFile is an simpler implementation of corev1.HostPath, needed for experiments
type HostFile struct {
	Name      string              `json:"name"`
	MountPath string              `json:"mountPath"`
	NodePath  string              `json:"nodePath"`
	Type      corev1.HostPathType `json:"type,omitempty"`
}

// SecurityContext defines the security contexts of the pod and container.
type SecurityContext struct {
	// PodSecurityContext holds security configuration that will be applied to a pod
	PodSecurityContext corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// ContainerSecurityContext holds security configuration that will be applied to a container
	ContainerSecurityContext corev1.SecurityContext `json:"containerSecurityContext,omitempty"`
}

type ComponentsEnv struct {
	Name  string `json:"name,omitempty" protobuf:"bytes,11,rep,name=name" yaml:"name"`
	Value string `json:"value,omitempty" protobuf:"bytes,11,rep,name=value" yaml:"value"`
}

const (
	serviceName = "litmuschaos"

	LitmusHelmNamespace = "chaos"
	LitmusHelmName      = "litmuschaos"
)

// resource type
const (
	LITMUS_EXPERIMENT = "experiment"
	LITMUS_RBAC       = "rbac"
	LITMUS_ENGINE     = "engine"

	ENGINE_KIND = "ChaosEngine"

	K8sKindClusterRole = "ClusterRole"
	K8sKindRole        = "Role"

	LITMUS_CRD_VERSION = "litmuschaos.io/v1alpha1"
)

func getLitmusUrlByVersionAndEnv(version string) string {
	return fmt.Sprintf("%s:%s", options.Opts.LitmusChartUrl, version)
}

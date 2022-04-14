module github.com/chaosblade-io/chaos-agent

go 1.16

require (
	github.com/c9s/goprocinfo v0.0.0-20210130143923-c95fcf8c64a8
	github.com/containerd/containerd v1.4.4
	github.com/deislabs/oras v0.11.1
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/go-units v0.4.0
	github.com/gosuri/uitable v0.0.4
	github.com/litmuschaos/chaos-operator v0.0.0-20210601045805-bab1c1c4b082
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/openebs/maya v1.12.1
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	helm.sh/helm/v3 v3.4.1 //v3.6.1-0.20210831203412-3d1bc72827e4
	k8s.io/api v0.19.9
	k8s.io/apimachinery v0.19.9
	k8s.io/client-go v12.0.0+incompatible
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/docker/docker => github.com/docker/docker v0.7.3-0.20190826074503-38ab9da00309

// pinned for github.com/litmuschaos/chaos-operator
replace (
	k8s.io/api => k8s.io/api v0.19.9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.10-rc.0
	k8s.io/apiserver => k8s.io/apiserver v0.19.9
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.9
	k8s.io/client-go => k8s.io/client-go v0.19.9
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.9
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.9
	k8s.io/code-generator => k8s.io/code-generator v0.19.10-rc.0
	k8s.io/component-base => k8s.io/component-base v0.19.9
	k8s.io/cri-api => k8s.io/cri-api v0.19.10-rc.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.9
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.9
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.9
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.9
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.9
	k8s.io/kubectl => k8s.io/kubectl v0.19.9
	k8s.io/kubelet => k8s.io/kubelet v0.19.9
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.9
	k8s.io/metrics => k8s.io/metrics v0.19.9
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.9
)

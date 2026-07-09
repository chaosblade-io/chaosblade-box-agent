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

package litmuschaos

import (
	//"fmt"

	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openebs/maya/pkg/util/retry"
	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	k8symaml "sigs.k8s.io/yaml"

	"github.com/litmuschaos/chaos-operator/api/litmuschaos/v1alpha1"
	chaosClient "github.com/litmuschaos/chaos-operator/pkg/client/clientset/versioned/typed/litmuschaos/v1alpha1"

	"github.com/chaosblade-io/chaos-agent/conn/asyncreport"
	"github.com/chaosblade-io/chaos-agent/pkg/kubernetes"
	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"

	//"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type LitmusChaosHandler struct {
	k8sChannel      *kubernetes.Channel
	LitmusClientSet *chaosClient.LitmuschaosV1alpha1Client

	transportClient *transport.TransportClient
}

func NewLitmusChaosHandler(transportClient *transport.TransportClient, k8sInstance *kubernetes.Channel) *LitmusChaosHandler {
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		logrus.Warnf("[litmus chaos] get cluster config failed, err: %s", err.Error())
		return nil
	}

	litmusClientSet, err := chaosClient.NewForConfig(clusterConfig)
	if err != nil {
		logrus.Warnf("[litmus chaos] get clientset failed, err: %s", err.Error())
		return nil
	}
	if k8sInstance == nil {
		logrus.Warnf("[litmus chaos] get k8s instance failed, the instance is nil")
		return nil
	}

	return &LitmusChaosHandler{
		k8sChannel:      k8sInstance,
		LitmusClientSet: litmusClientSet,
		transportClient: transportClient,
	}
}

func (lh *LitmusChaosHandler) Handle(request *transport.Request) *transport.Response {
	chaosAction := request.Params["chaosAction"]
	ctx := context.TODO()
	if _, ok := options.CreateOperation[chaosAction]; ok {
		return lh.createParamerAndExec(ctx, request)
	} else if _, ok := options.DestroyOperation[chaosAction]; ok {
		return lh.destroyParamerAndExec(ctx, request)
	}

	return transport.ReturnFail(transport.ServerError, fmt.Sprintf("litmus exec failed, no such action: %s", chaosAction))
}

// destroy
func (lh *LitmusChaosHandler) destroyParamerAndExec(ctx context.Context, request *transport.Request) *transport.Response {
	name, ok := request.Params["name"]
	if !ok || name == "" {
		logrus.Warnf("[litmus destroy] less parameter: `name`")
		return transport.ReturnFail(transport.ParameterEmpty, "name")
	}

	namespace, ok := request.Params["namespace"]
	if !ok || namespace == "" {
		namespace = coreV1.NamespaceDefault
	}

	return lh.destroyExec(ctx, name, namespace)
}

func (lh *LitmusChaosHandler) destroyExec(ctx context.Context, name, namespace string) *transport.Response {
	err := lh.LitmusClientSet.ChaosEngines(namespace).Delete(ctx, name, metaV1.DeleteOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return transport.ReturnFail(transport.ServerError, fmt.Sprintf("litmus destroy engine failed, err: %s", err.Error()))
		}
	}

	return transport.ReturnSuccess()
}

// create
func (lh *LitmusChaosHandler) createParamerAndExec(ctx context.Context, request *transport.Request) *transport.Response {
	name, _ := tools.GenerateUid()

	namespace, ok := request.Params["namespace"]
	if !ok || namespace == "" {
		namespace = coreV1.NamespaceDefault
	}
	experimentType, ok := request.Params["experimentType"]
	if !ok || experimentType == "" {
		return transport.ReturnFail(transport.ParameterEmpty, "experimentType")
	}

	experimentName, ok := request.Params["experimentName"]
	if !ok || experimentType == "" {
		return transport.ReturnFail(transport.ParameterEmpty, "experimentName")
	}

	appInfoStr, ok := request.Params["appInfo"]
	if !ok || experimentType == "" {
		return transport.ReturnFail(transport.ParameterEmpty, "appInfo")
	}
	var appInfo map[string]string
	if err := json.Unmarshal([]byte(appInfoStr), &appInfo); err != nil {
		return transport.ReturnFail(transport.ParameterTypeError, "appInfo")
	}

	componentsStr, ok := request.Params["components"]
	if !ok || experimentType == "" {
		return transport.ReturnFail(transport.ParameterEmpty, "components")
	}
	var components map[string]string
	if err := json.Unmarshal([]byte(componentsStr), &components); err != nil {
		return transport.ReturnFail(transport.ParameterTypeError, "components")
	}

	return lh.createExec(ctx, experimentType, experimentName, namespace, name, appInfo, components)
}

func (lh *LitmusChaosHandler) createExec(ctx context.Context, experimentType, experimentName, namespace, name string, appInfo, components map[string]string) *transport.Response {
	if options.Opts.LitmusChaosVerison == "" {
		return transport.ReturnFail(transport.ServerError, "litmus operator not installed, please install first")
	}

	if err := lh.prepareLitmusExperiment(ctx, experimentType, experimentName, namespace); err != nil {
		return transport.ReturnFail(transport.ServerError, fmt.Sprintf("litmus prepare experiment failed, err: %s", err.Error()))
	}

	if err := lh.prepareLitmusRbac(experimentType, experimentName, namespace); err != nil {
		return transport.ReturnFail(transport.ServerError, fmt.Sprintf("litmus prepare rbac failed, err: %s", err.Error()))
	}

	if err := lh.createEnginer(ctx, name, namespace, experimentName, appInfo, components); err != nil {
		return transport.ReturnFail(transport.ServerError, fmt.Sprintf("litmus create enginer failed, err: %s", err.Error()))
	}

	// async report inject fault status
	go lh.AsyncHandlerResultStatus(ctx, name, namespace, experimentName)

	return transport.ReturnSuccess()
}

func (lh *LitmusChaosHandler) AsyncHandlerResultStatus(ctx context.Context, name, namespace, experimentName string) {
	status := "Unknown"
	var errorStr string
	time.Sleep(15 * time.Second)
	err := retry.
		Times(90).
		Wait(1 * time.Second).
		Try(func(attempt uint) error {
			chaosResult, err := lh.handlerResult(ctx, name, namespace, experimentName)
			if err != nil {
				return err
			}
			failedRuns := chaosResult.Status.History.FailedRuns
			if chaosResult.Spec.EngineName != "" && failedRuns == 0 {
				return nil
			}
			return fmt.Errorf("failed in chaos injection phase")

			// todo: 拿到的chaosresult里面的status.ExperimentStatus，一直是空，所以根据这个来判断不好用

			// https://github.com/litmuschaos/chaos-operator/issues/368 所以暂时通过先sleep再感觉faile个数来判断是否OK
			//phase := chaosResult.Status.ExperimentStatus.Phase
			//verdict := chaosResult.Status.ExperimentStatus.Verdict
			//if phase == "" || verdict == "" {
			//	return fmt.Errorf("chaosengine not ready")
			//}
			//if phase == "Running" && verdict == "Awaited" {
			//	return nil
			//}
			//if phase == "Completed" && (verdict == "Pass" || verdict == "Stopped") {
			//	return nil
			//}
			//return fmt.Errorf(chaosResult.Status.ExperimentStatus.FailStep)
		})
	if err != nil {
		errorStr = fmt.Sprintf("inject fault failed, err: %s", err.Error())
		status = "Error"
	} else {
		status = "Success"
	}

	// async report result to server
	uri := transport.TransportUriMap[transport.API_CHAOSBLADE_ASYNC]
	ar := asyncreport.NewClientCloseHandler(lh.transportClient)
	ar.ReportStatus(name, status, errorStr, LitmusHelmName, uri)
}

// prepareLitmus before inject fault, need create experiment
func (lh *LitmusChaosHandler) prepareLitmusExperiment(ctx context.Context, experimentType, experimentName, namespace string) error {
	// var crdsDefinition  *apiextensionv1beta1.CustomResourceDefinition
	var expriment *v1alpha1.ChaosExperiment
	litmusExperiment, err := DownloadLitmus(experimentType, experimentName, LITMUS_EXPERIMENT)
	if err != nil {
		return err
	}
	if err = k8symaml.Unmarshal(litmusExperiment, &expriment); err != nil {
		return fmt.Errorf("experiment Unmarshal err: %v", err.Error())
	}

	_, err = lh.LitmusClientSet.ChaosExperiments(namespace).Create(ctx, expriment, metaV1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// 由于不同版本的role的内容还是有很大不同的不同的，所以每次都需要先判断，如果已存在，则需要先删掉，然后再重新创建
// 但是对于sa和rolebinding，就没有必要重建，对于已经有了，则直接pass
func (lh *LitmusChaosHandler) prepareLitmusRbac(experimentType, experimentName, namespace string) error {
	// get rbac yaml
	result, err := DownloadLitmus(experimentType, experimentName, LITMUS_RBAC)
	if err != nil {
		return err
	}
	restr := string(result)
	resArr := strings.Split(restr, "---")
	if len(resArr) < 3 {
		return fmt.Errorf("get rbac.yaml failed")
	}

	// service account
	var sa *coreV1.ServiceAccount
	saYamlStr := []byte(strings.Trim(resArr[1], "\n"))
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(saYamlStr), len(saYamlStr)).Decode(&sa)
	if err != nil {
		logrus.Error(err.Error())
	}
	_, err = lh.k8sChannel.ClientSet.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), sa, metaV1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
		} else {
			return err
		}
	}

	// role and rolebinding
	var role *rbacV1.Role
	roleYamlStr := []byte(strings.Trim(resArr[2], "\n"))
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(roleYamlStr), len(roleYamlStr)).Decode(&role)
	if err != nil {
		return err
	}
	rolebindingYamlStr := []byte(strings.Trim(resArr[3], "\n"))
	if role.Kind == K8sKindClusterRole {
		return lh.createClusterRoleAndRoleBinding(roleYamlStr, rolebindingYamlStr, namespace)
	}
	return lh.createRoleAndRoleBinding(roleYamlStr, rolebindingYamlStr, namespace)
}

func (lh *LitmusChaosHandler) createClusterRoleAndRoleBinding(roleStr, roleBindingStr []byte, namespace string) error {
	// clusterRole
	var role *rbacV1.ClusterRole
	err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(roleStr), len(roleStr)).Decode(&role)
	if err != nil {
		return err
	}
	_, err = lh.k8sChannel.ClientSet.RbacV1().ClusterRoles().Get(context.TODO(), role.Name, metaV1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	if err == nil {
		logrus.Info("[prepare litmus] ClusterRole created need delete!")
		err = lh.k8sChannel.ClientSet.RbacV1().ClusterRoles().Delete(context.TODO(), role.Name, metaV1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	_, err = lh.k8sChannel.ClientSet.RbacV1().ClusterRoles().Create(context.TODO(), role, metaV1.CreateOptions{})
	if err != nil {
		return err
	}

	// role binding
	var roleBinding *rbacV1.ClusterRoleBinding
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(roleBindingStr), len(roleBindingStr)).Decode(&roleBinding)
	if err != nil {
		return err
	}
	_, err = lh.k8sChannel.ClientSet.RbacV1().ClusterRoleBindings().Create(context.TODO(), roleBinding, metaV1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			logrus.Infof("[prepare litmus] create ClusterRoleBindings created. err %s", err.Error())
		} else {
			return err
		}
	}
	return nil
}

func (lh *LitmusChaosHandler) createRoleAndRoleBinding(roleStr, roleBindingStr []byte, namespace string) error {
	// role
	var role *rbacV1.Role
	err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(roleStr), len(roleStr)).Decode(&role)
	if err != nil {
		return err
	}
	_, err = lh.k8sChannel.ClientSet.RbacV1().Roles(namespace).Get(context.TODO(), role.Name, metaV1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	if err == nil {
		logrus.Info("[prepare litmus] Role created need delete!")
		err = lh.k8sChannel.ClientSet.RbacV1().Roles(namespace).Delete(context.TODO(), role.Name, metaV1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	_, err = lh.k8sChannel.ClientSet.RbacV1().Roles(namespace).Create(context.TODO(), role, metaV1.CreateOptions{})
	if err != nil {
		return err
	}

	// role binding
	var roleBinding *rbacV1.RoleBinding
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(roleBindingStr), len(roleBindingStr)).Decode(&roleBinding)
	if err != nil {
		return err
	}
	_, err = lh.k8sChannel.ClientSet.RbacV1().RoleBindings(namespace).Create(context.TODO(), roleBinding, metaV1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			logrus.Infof("[prepare litmus] create RoleBindings created. err %s", err.Error())
		} else {
			return err
		}
	}
	logrus.Errorf("end create RoleBinding!!!")
	return nil
}

func (lh *LitmusChaosHandler) createEnginer(ctx context.Context, name, namespace, experimentName string, appInfo, components map[string]string) error {
	var componentEnv []coreV1.EnvVar
	for key, value := range components {
		if value == "" {
			continue
		}
		component := coreV1.EnvVar{
			Name:  key,
			Value: value,
		}
		componentEnv = append(componentEnv, component)
	}

	var appInfoParam v1alpha1.ApplicationParams
	for key, value := range appInfo {
		if value == "" {
			continue
		}
		switch key {
		case "appkind":
			appInfoParam.AppKind = value
		case "appns":
			appInfoParam.Appns = value
		case "applabel":
			appInfoParam.Applabel = value
		default:
			continue
		}
	}

	chaosEnginer := &v1alpha1.ChaosEngine{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ChaosEngineSpec{
			Appinfo:             appInfoParam,
			ChaosServiceAccount: experimentName + "-sa",
			JobCleanUpPolicy:    "delete",
			EngineState:         "active",
			// AnnotationCheck:     "false",
			Experiments: []v1alpha1.ExperimentList{
				{
					Name: experimentName,
					Spec: v1alpha1.ExperimentAttributes{
						Components: v1alpha1.ExperimentComponents{
							ENV: componentEnv,
						},
					},
				},
			},
		},
	}
	chaosEnginer.Kind = ENGINE_KIND
	chaosEnginer.APIVersion = LITMUS_CRD_VERSION

	_, err := lh.LitmusClientSet.ChaosEngines(namespace).Create(ctx, chaosEnginer, metaV1.CreateOptions{})
	return err
}

func (lh *LitmusChaosHandler) handlerResult(ctx context.Context, name, namespace, experimentName string) (*v1alpha1.ChaosResult, error) {
	chaosResultName := fmt.Sprintf("%s-%s", name, experimentName)

	return lh.LitmusClientSet.ChaosResults(namespace).Get(ctx, chaosResultName, metaV1.GetOptions{})
}

// Download from oss
func DownloadLitmus(experimentType, experimentName, objectType string) ([]byte, error) {
	version := options.Opts.LitmusChaosVerison
	localFilePath := fmt.Sprintf("%s/%s/%s/%s.yaml", version, experimentType, experimentName, objectType)
	if tools.IsExist(localFilePath) {
		return ioutil.ReadFile(localFilePath)
	}

	// download url
	// eg: https://hub.litmuschaos.io/api/chaos/1.13.5?file=charts/generic/pod-delete/experiment.yaml
	experimentUrl := fmt.Sprintf("https://hub.litmuschaos.io/api/chaos/%s?file=charts/%s/%s/%s.yaml", version, experimentType, experimentName, objectType)
	resp, err := http.Get(experimentUrl)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download `%s` failed! response code: %d", experimentUrl, resp.StatusCode)
	}

	// create file
	tools.IsExist(localFilePath)
	file, err := os.Create(localFilePath)
	os.Chmod(localFilePath, 0o744)
	defer file.Close()

	// copy url response to file
	buf := make([]byte, 0)
	for {
		n, _ := resp.Body.Read(buf)
		if 0 == n {
			break
		}
		file.WriteString(string(buf[:n]))
	}
	return ioutil.ReadAll(resp.Body)
}

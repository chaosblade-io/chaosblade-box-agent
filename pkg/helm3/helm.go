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

package helm3

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/strvals"

	"github.com/chaosblade-io/chaos-agent/pkg/helm3/registry"
)

// https://github.com/helm/helm/issues/8255

type Helm struct {
	helmName       string
	helmNamespace  string
	setting        *cli.EnvSettings
	actionConfig   *action.Configuration
	RegistryClient *registry.Client
}

func GetHelmInstance(helmName, helmNamespace string, out io.Writer) *Helm {
	setting := cli.New()
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(setting.RESTClientGetter(), helmNamespace, os.Getenv("HELM_DRIVER"), logrus.Infof); err != nil {
		logrus.Warnf("[helm] config init failed, err: %s", err.Error())
		return nil
	}

	registryClient, err := registry.NewClient(registry.ClientOptDebug(setting.Debug),
		registry.ClientOptWriter(out),
		registry.ClientOptCredentialsFile(setting.RegistryConfig))
	if err != nil {
		logrus.Warnf("[helm] registry client failed, err: %s", err.Error())
		return nil
	}

	return &Helm{
		helmName:      helmName,
		helmNamespace: helmNamespace,

		setting:        setting,
		actionConfig:   actionConfig,
		RegistryClient: registryClient,
	}
}

func (h *Helm) PullChart(chartUrl string) error {
	r, err := registry.ParseReference(chartUrl)
	if err != nil {
		return err
	}

	return h.RegistryClient.PullChartToCache(r)
}

func (h *Helm) LoadChart(chartUrl string) (*chart.Chart, error) {
	r, err := registry.ParseReference(chartUrl)
	if err != nil {
		return nil, err
	}

	return h.RegistryClient.LoadChart(r)
}

func (h *Helm) Install(charts *chart.Chart, args map[string]string) error {
	client := action.NewInstall(h.actionConfig)
	client.Namespace = h.helmNamespace
	client.ReleaseName = h.helmName

	// vals
	p := getter.All(h.setting)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return err
	}
	//add vals
	if err := strvals.ParseInto(args["set"], vals); err != nil {
		return err
	}

	validInstallableChart, err := isChartInstallable(charts)
	if !validInstallableChart {
		return err
	}

	_, err = client.Run(charts, vals)
	return err
}

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func (helm *Helm) Uninstall() error {
	client := action.NewUninstall(helm.actionConfig)
	_, err := client.Run(helm.helmName)
	return err
}

func (helm *Helm) List() ([]*release.Release, error) {
	client := action.NewList(helm.actionConfig)
	return client.Run()

}

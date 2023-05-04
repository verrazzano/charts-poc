// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"fmt"
	"os"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	helmrepo "helm.sh/helm/v3/pkg/repo"
)

type HelmConfig struct {
	settings     *cli.EnvSettings
	helmRepoFile *helmrepo.File
	vzHelper     helpers.VZHelper
}

type ChartProvenance struct {
	UpstreamVersion        string                 `yaml:"upstreamVersion"`
	UpstreamChartLocalPath string                 `yaml:"upstreamChartLocalPath"`
	UpstreamIndexEntry     *helmrepo.ChartVersion `yaml:"upstreamIndexEntry"`
}

func NewHelmConfig(vzHelper helpers.VZHelper) (*HelmConfig, error) {
	helmConfig := &HelmConfig{vzHelper: vzHelper}
	helmConfig.settings = cli.New()
	if helmConfig.settings.RepositoryConfig == "" {
		helmConfig.settings.RepositoryConfig = "/tmp/vz_helm_repo.yaml"
	}

	var err error
	if helmConfig.settings.RepositoryCache == "" {
		helmConfig.settings.RepositoryCache = "/tmp/vz_helm_repo_cache"
		err = os.MkdirAll(helmConfig.settings.RepositoryCache, 0755)
		if err != nil {
			return nil, err
		}
	}

	if _, err = os.Stat(helmConfig.settings.RepositoryConfig); err != nil {
		err = helmrepo.NewFile().WriteFile(helmConfig.settings.RepositoryConfig, 0o644)
		if err != nil {
			return nil, err
		}
	}

	helmConfig.helmRepoFile, err = helmrepo.LoadFile(helmConfig.settings.RepositoryConfig)
	if err != nil {
		return nil, err
	}
	return helmConfig, nil
}

func getHelmRepoName(chart string) string {
	return fmt.Sprintf("%s-provider", chart)
}

func (h *HelmConfig) AddAndUpdateChartRepo(chart string, repoUrl string) (string, error) {
	repoEntry, err := h.getRepoEntry(repoUrl)
	if err != nil {
		return "", err
	}

	if repoEntry == nil {
		repoEntry = &helmrepo.Entry{
			Name: getHelmRepoName(chart),
			URL:  repoUrl,
		}
		fmt.Fprintf(h.vzHelper.GetOutputStream(), "Adding helm repo %s.\n", repoEntry.Name)
	} else {
		fmt.Fprintf(h.vzHelper.GetOutputStream(), "Using helm repo %s\n", repoEntry.Name)
	}

	chartRepo, err := helmrepo.NewChartRepository(repoEntry, getter.All(h.settings))
	if err != nil {
		return "", err
	}

	_, err = chartRepo.DownloadIndexFile()
	if err != nil {
		return "", err
	}

	h.helmRepoFile.Update(repoEntry)
	return repoEntry.Name, h.helmRepoFile.WriteFile(h.settings.RepositoryConfig, 0o644)
}

func (h *HelmConfig) DownloadChart(chart string, repo string, version string, targetVersion string, chartDir string) error {
	registryClient, err := registry.NewClient(
		registry.ClientOptDebug(h.settings.Debug),
		registry.ClientOptCredentialsFile(h.settings.RegistryConfig),
	)
	if err != nil {
		return err
	}

	config := &action.Configuration{
		RegistryClient: registryClient,
	}

	pull := action.NewPullWithOpts(action.WithConfig(config))
	pull.Untar = true
	pull.UntarDir = fmt.Sprintf("%s/%s/%s", chartDir, chart, targetVersion)
	pull.Settings = cli.New()
	pull.Version = version
	err = os.RemoveAll(pull.UntarDir)
	if err != nil {
		return err
	}

	out, err := pull.Run(fmt.Sprintf("%s/%s", repo, chart))
	if out != "" {
		fmt.Fprintln(h.vzHelper.GetOutputStream(), out)
	}

	return err
}

func (h *HelmConfig) GetChartProvenance(chart string, repo string, version string) (*ChartProvenance, error) {
	repoEntry, err := h.getRepoEntry(repo)
	if err != nil {
		return nil, err
	}

	indexPath := fmt.Sprintf("%s/%s-index.yaml", h.settings.RepositoryCache, repoEntry.Name)
	if _, err = os.Stat(indexPath); err != nil {
		return nil, err
	}

	indexFile, err := helmrepo.LoadIndexFile(indexPath)
	if err != nil {
		return nil, err
	}

	chartVersion, err := indexFile.Get(chart, version)
	if err != nil {
		return nil, err
	}

	return &ChartProvenance{
		UpstreamVersion:        version,
		UpstreamChartLocalPath: fmt.Sprintf("upstreams/%s", version),
		UpstreamIndexEntry:     chartVersion,
	}, nil
}

func (h *HelmConfig) getRepoEntry(repoUrl string) (*helmrepo.Entry, error) {
	for _, repoEntry := range h.helmRepoFile.Repositories {
		if repoEntry.URL == repoUrl {
			return repoEntry, nil
		}
	}

	return nil, fmt.Errorf("could not find repo entry")
}

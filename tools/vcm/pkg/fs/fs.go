// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/verrazzano/charts-poc/tools/vcm/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"gopkg.in/yaml.v3"
)

func RearrangeChartDirectory(chart string, chartDir string, targetVersion string) error {
	pulledChartDir := fmt.Sprintf("%s/%s/%s/%s", chartDir, chart, targetVersion, chart)
	cmd := exec.Command("cp", "-R", fmt.Sprintf("%s/", pulledChartDir), fmt.Sprintf("%s/%s/%s", chartDir, chart, targetVersion))
	err := cmd.Run()
	if err != nil {
		return err
	}

	err = os.RemoveAll(pulledChartDir)
	if err != nil {
		return err
	}
	return nil
}

func SaveUpstreamChart(chart string, version string, targetVersion string, chartDir string) error {
	provenanceDir := fmt.Sprintf("%s/../provenance/%s/upstreams/%s", chartDir, chart, version)
	err := os.RemoveAll(provenanceDir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(provenanceDir, 0755)
	if err != nil {
		return err
	}

	cmd := exec.Command("cp", "-R", fmt.Sprintf("%s/%s/%s/", chartDir, chart, targetVersion), provenanceDir)
	return cmd.Run()
}

func SaveChartProvenance(chartProvenance *helm.ChartProvenance, chart string, targetVersion string, chartDir string) error {
	provenanceFile := fmt.Sprintf("%s/../provenance/%s/%s.yaml", chartDir, chart, targetVersion)
	out, err := yaml.Marshal(chartProvenance)
	if err != nil {
		return err
	}

	return os.WriteFile(provenanceFile, out, 0755)
}

func GeneratePatchFile(chart string, version string, chartsDir string) (string, error) {
	provenanceFile := fmt.Sprintf("%s/../provenance/%s/%s.yaml", chartsDir, chart, version)
	if _, err := os.Stat(provenanceFile); err != nil {
		return "", fmt.Errorf("provenance file %s not found, error %v", provenanceFile, err)
	}

	in, err := os.ReadFile(provenanceFile)
	if err != nil {
		return "", fmt.Errorf("unable to read provenance file %s, error %v", provenanceFile, err)
	}

	chartProvenance := helm.ChartProvenance{}
	err = yaml.Unmarshal(in, &chartProvenance)
	if err != nil {
		return "", fmt.Errorf("unable to parse provenance file %s, error %v", provenanceFile, err)
	}

	chartDir := fmt.Sprintf("%s/%s/%s", chartsDir, chart, version)
	if _, err := os.Stat(chartDir); err != nil {
		return "", fmt.Errorf("chart directory %s not found, error %v", chartDir, err)
	}

	upstreamChartDir, err := filepath.Abs(fmt.Sprintf("%s/../provenance/%s/%s", chartsDir, chart, chartProvenance.UpstreamChartLocalPath))
	if err != nil {
		return "", fmt.Errorf("unable to find absolute path to upstream chart directory at %s, error %v", upstreamChartDir, err)
	}

	if _, err := os.Stat(upstreamChartDir); err != nil {
		return "", fmt.Errorf("upstream chart directory %s not found, error %v", upstreamChartDir, err)
	}

	patchFile, err := os.Create(fmt.Sprintf("%s/../vz_charts_patch_%s_%s.patch", chartsDir, chart, version))
	if err != nil {
		return "", fmt.Errorf("unable to create empty patch file")
	}

	cmd := exec.Command("diff", "-Naurw", upstreamChartDir, chartDir)
	cmd.Stdout = patchFile
	err = cmd.Run()
	if err != nil {
		// diff returning exit status 1 even when file diff is completed and no underlying error.
		// error out onlt when message is different
		if err.Error() != "exit status 1" {
			return "", fmt.Errorf("error running command %s, error %v", cmd.String(), err)
		}
	}

	patchFileStats, err := os.Stat(patchFile.Name())
	if err != nil {
		return "", fmt.Errorf("unable to stat patch file at %v, error %v", patchFile.Name(), err)
	}

	if patchFileStats.Size() == 0 {
		err := os.Remove(patchFile.Name())
		if err != nil {
			return "", fmt.Errorf("unable to remove empty patch file at %v, error %v", patchFile.Name(), err)
		}

		return "", nil
	}

	return patchFile.Name(), nil
}

func FindChartVersionToPatch(chartsDir string, chart string, version string) (string, error) {
	chartDirParent := fmt.Sprintf("%s/%s", chartsDir, chart)
	entries, err := os.ReadDir(chartDirParent)
	if err != nil {
		return "", fmt.Errorf("unable to read chart dierctory %s, error %v", chartDirParent, err)
	}

	currentChartVersion, err := semver.NewSemVersion(version)
	if err != nil {
		return "", fmt.Errorf("invalid chart version %s, error %v", version, err)
	}

	var versions []*semver.SemVersion
	for _, entry := range entries {
		if entry.IsDir() {
			chartVersion, err := semver.NewSemVersion(entry.Name())
			if err != nil {
				return "", fmt.Errorf("invalid chart version %s, error %v", chartVersion.ToString(), err)
			}

			if chartVersion.IsLessThan(currentChartVersion) {
				versions = append(versions, chartVersion)
			}
		}
	}

	if len(versions) == 0 {
		return "", nil
	}

	highestVersion := versions[0]
	for _, version := range versions {
		if version.IsGreatherThan(highestVersion) {
			highestVersion = version
		}
	}
	return highestVersion.ToString(), nil
}

func ApplyPatchFile(chart string, version string, chartsDir string, patchFile string) ([]byte, string, error) {
	chartDir := fmt.Sprintf("%s/%s/%s/", chartsDir, chart, version)
	if _, err := os.Stat(chartDir); err != nil {
		return nil, "", fmt.Errorf("chart directory %s not found, error %v", chartDir, err)
	}

	if _, err := os.Stat(patchFile); err != nil {
		return nil, "", fmt.Errorf("patch file %s not found, error %v", patchFile, err)
	}

	rejectsFilePathAbsolute, err := filepath.Abs(fmt.Sprintf("%s/../vz_charts_patch_%s_%s_rejects.rejects", chartsDir, chart, version))
	if err != nil {
		return nil, "", fmt.Errorf("unable to find absolute path for rejects file")
	}

	_, err = os.Create(rejectsFilePathAbsolute)
	if err != nil {
		return nil, "", fmt.Errorf("unable to create empty rejects file")
	}

	cmd := exec.Command("patch", "--no-backup-if-mismatch", "-p"+fmt.Sprint(strings.Count(chartDir, string(os.PathSeparator))), "-r", rejectsFilePathAbsolute, "--directory", chartDir, "<"+patchFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("error running command %s, error %v", cmd.String(), err)
	}

	rejectsFileStats, err := os.Stat(rejectsFilePathAbsolute)
	if err != nil {
		return nil, "", fmt.Errorf("unable to stat reject file at %v, error %v", rejectsFilePathAbsolute, err)
	}

	if rejectsFileStats.Size() == 0 {
		err := os.Remove(rejectsFilePathAbsolute)
		if err != nil {
			return nil, "", fmt.Errorf("unable to remove empty rejects file at %v, error %v", rejectsFilePathAbsolute, err)
		}

		return out, "", nil
	}

	return out, rejectsFilePathAbsolute, nil
}

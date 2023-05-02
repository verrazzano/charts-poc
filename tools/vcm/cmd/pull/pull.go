// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pull

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	vcmhelpers "github.com/verrazzano/charts-poc/tools/vcm/cmd/helpers"
	"github.com/verrazzano/charts-poc/tools/vcm/pkg/constants"
	"github.com/verrazzano/charts-poc/tools/vcm/pkg/fs"
	"github.com/verrazzano/charts-poc/tools/vcm/pkg/helm"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "pull"
	helpShort   = "Pulls an upstream chart/version"
	helpLong    = `The command 'pull' pulls an upstream chart`
)

func buildExample() string {
	examples := []string{fmt.Sprintf(constants.CommandWithFlagExampleFormat+" "+
		constants.FlagExampleFormat+" "+
		constants.FlagExampleFormat+" "+
		constants.FlagExampleFormat,
		CommandName, constants.FlagChartName, constants.FlagChartShorthand, constants.FlagChartExampleKeycloak,
		constants.FlagVersionName, constants.FlagPatchVersionShorthand, constants.FlagVersionExample210,
		constants.FlagRepoName, constants.FlagRepoShorthand, constants.FlagRepoExampleCodecentric,
		constants.FlagDirName, constants.FlagDirShorthand, constants.FlagDirExampleLocal)}

	examples = append(examples, fmt.Sprintf(constants.CommandWithFlagExampleFormat, examples[len(examples)-1],
		constants.FlagTargetVersionName, constants.FlagChartShorthand, constants.FlagTargetVersionExample002))

	examples = append(examples, fmt.Sprintf(constants.CommandWithFlagExampleFormat, examples[len(examples)-1],
		constants.FlagUpstreamProvenanceName, constants.FlagUpstreamProvenanceShorthand, constants.FlagUpstreamProvenanceDefault))

	examples = append(examples, fmt.Sprintf(constants.CommandWithFlagExampleFormat, examples[len(examples)-1],
		constants.FlagPatchName, constants.FlagPatchShorthand, constants.FlagPatchDefault))

	examples = append(examples, fmt.Sprintf(constants.CommandWithFlagExampleFormat, examples[len(examples)-1],
		constants.FlagPatchVersionName, constants.FlagPatchVersionShorthand, constants.FlagPatchVersionExample001))

	return fmt.Sprintln(examples)
}

func NewCmdPull(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdPull(cmd, vzHelper)
	}
	cmd.Example = buildExample()
	cmd.PersistentFlags().StringP(constants.FlagChartName, constants.FlagChartShorthand, "", constants.FlagChartUsage)
	cmd.PersistentFlags().StringP(constants.FlagVersionName, constants.FlagVersionShorthand, "", constants.FlagVersionExample210)
	cmd.PersistentFlags().StringP(constants.FlagRepoName, constants.FlagRepoShorthand, "", constants.FlagRepoUsage)
	cmd.PersistentFlags().StringP(constants.FlagDirName, constants.FlagDirShorthand, "", constants.FlagDirUsage)
	cmd.PersistentFlags().StringP(constants.FlagTargetVersionName, constants.FlagTargetVersionShorthand, "", constants.FlagTargetVersionExample002)
	cmd.PersistentFlags().BoolP(constants.FlagUpstreamProvenanceName, constants.FlagUpstreamProvenanceShorthand, true, constants.FlagUpstreamProvenanceUsage)
	cmd.PersistentFlags().BoolP(constants.FlagPatchName, constants.FlagPatchShorthand, true, constants.FlagPatchUsage)
	cmd.PersistentFlags().StringP(constants.FlagPatchVersionName, constants.FlagPatchVersionShorthand, "", constants.FlagPatchVersionUsage)

	return cmd
}

// runCmdStatus - run the "vz status" command
func runCmdPull(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	chart, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagChartName, constants.FlagChartShorthand)
	if err != nil {
		return err
	}

	version, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagVersionName, constants.FlagVersionShorthand)
	if err != nil {
		return err
	}

	repo, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagRepoName, constants.FlagRepoShorthand)
	if err != nil {
		return err
	}

	chartsDir, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagDirName, constants.FlagDirShorthand)
	if err != nil {
		return err
	}

	targetVersion, err := cmd.PersistentFlags().GetString(constants.FlagTargetVersionName)
	if err != nil {
		return err
	}

	if targetVersion == "" {
		targetVersion = version
	}

	if len(strings.TrimSpace(targetVersion)) == 0 {
		return fmt.Errorf("%s can not be empty", constants.FlagTargetVersionName)
	}

	saveUpstream, err := cmd.PersistentFlags().GetBool(constants.FlagUpstreamProvenanceName)
	if err != nil {
		return err
	}

	patchDiffs, err := cmd.PersistentFlags().GetBool(constants.FlagPatchName)
	if err != nil {
		return err
	}

	var patchVersion string
	if patchDiffs {
		patchVersion, err = cmd.PersistentFlags().GetString(constants.FlagPatchVersionName)
		if err != nil {
			return err
		}
	}

	helmConfig, err := helm.NewHelmConfig(vzHelper)
	if err != nil {
		return fmt.Errorf("unable to init helm config, error %v", err)
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), "\nAdding/Updtaing %s chart repo with url %s\n", chart, repo)
	repoName, err := helmConfig.AddAndUpdateChartRepo(chart, repo)
	if err != nil {
		return err
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), "Pulling %s chart version %s to target version %s..\n", chart, version, targetVersion)
	err = helmConfig.DownloadChart(chart, repoName, version, targetVersion, chartsDir)
	if err != nil {
		return err
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), "Rearrange chart directory\n")
	err = fs.RearrangeChartDirectory(chart, chartsDir, targetVersion)
	if err != nil {
		return err
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), "\nPulled chart %s version %s to target version %s from %s to %s/%s/%s", chart, version, targetVersion, repo, chartsDir, chart, targetVersion)
	if saveUpstream {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Save upstream chart\n")
		err = fs.SaveUpstreamChart(chart, version, targetVersion, chartsDir)
		if err != nil {
			return err
		}

		fmt.Fprintf(vzHelper.GetOutputStream(), "Save chart provenance file\n")
		chartProvenance, err := helmConfig.GetChartProvenance(chart, repo, version)
		if err != nil {
			return err
		}

		err = fs.SaveChartProvenance(chartProvenance, chart, targetVersion, chartsDir)
		if err != nil {
			return err
		}

		fmt.Fprintf(vzHelper.GetOutputStream(), "\nUpstream chart saved to %s/../provenance/%s/upstreams/%s", chartsDir, chart, version)
		fmt.Fprintf(vzHelper.GetOutputStream(), "\nUpstream provenance manifest created in %s/../provenance/%s/%s.yaml", chartsDir, chart, targetVersion)

	}

	if patchDiffs {
		if patchVersion == "" {
			patchVersion, err = fs.FindChartVersionToPatch(chartsDir, chart, targetVersion)
			if err != nil {
				return fmt.Errorf("unable to find version to patch, error %v", err)
			}
		}

		if patchVersion != "" {
			patchFile, err := fs.GeneratePatchFile(chart, patchVersion, chartsDir)
			if err != nil {
				return fmt.Errorf("unable to generate patch file, error %v", err)
			}

			fileStat, err := os.Stat(patchFile)
			if err != nil {
				return fmt.Errorf("unable to read patch file at %s, error %v", patchFile, err)
			}

			if fileStat.Size() == 0 {
				fmt.Fprintf(vzHelper.GetOutputStream(), "Nothing to patch from previous version\n")
			}

			outFile, rejectsFile, err := fs.ApplyPatchFile(chart, targetVersion, chartsDir, patchFile)
			if err != nil {
				return fmt.Errorf("unable to apply patch file %s, error %v", patchFile, err)
			}

			fileStat, err = os.Stat(outFile)
			if err != nil {
				return fmt.Errorf("unable to read patching output file at %s, error %v", outFile, err)
			}

			if fileStat.Size() != 0 {
				output, err := os.ReadFile(outFile)
				if err != nil {
					return fmt.Errorf("unable to read patching output file at %s, error %v", outFile, err)
				}

				fmt.Fprintf(vzHelper.GetOutputStream(), "Patching output:\n%s", string(output))
			}

			fileStat, err = os.Stat(rejectsFile)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintf(vzHelper.GetOutputStream(), "No rejects from patching\n")
				} else {
					return fmt.Errorf("unable to read rejects file at %s, error %v", rejectsFile, err)
				}
			} else {
				if fileStat.Size() != 0 {
					rejects, err := os.ReadFile(rejectsFile)
					if err != nil {
						return fmt.Errorf("unable to read rejects file at %s, error %v", rejectsFile, err)
					}

					fmt.Fprintf(vzHelper.GetOutputStream(), "Patching results for rejects:\n%s", string(rejects))
				}
			}
			fmt.Fprintf(vzHelper.GetOutputStream(), "\nAny diffs from version %s has been applied\n", patchVersion)
		}
	}
	return nil
}

// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/charts-poc/tools/vcm/cmd/pull"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

var kubeconfigFlagValPointer string
var contextFlagValPointer string

const (
	CommandName = "vcm"
	helpShort   = "The vz tool is a command-line utility that allows Verrazzano operators to query and manage a Verrazzano environment"
	helpLong    = "The vz tool is a command-line utility that allows Verrazzano operators to query and manage a Verrazzano environment"
)

// NewRootCmd - create the root cobra command
func NewRootCmd(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)

	// Add global flags
	cmd.PersistentFlags().StringVar(&kubeconfigFlagValPointer, constants.GlobalFlagKubeConfig, "", constants.GlobalFlagKubeConfigHelp)
	cmd.PersistentFlags().StringVar(&contextFlagValPointer, constants.GlobalFlagContext, "", constants.GlobalFlagContextHelp)

	// Add commands
	cmd.AddCommand(pull.NewCmdPull(vzHelper))
	return cmd
}

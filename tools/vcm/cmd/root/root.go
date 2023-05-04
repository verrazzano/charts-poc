// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/charts-poc/tools/vcm/cmd/diff"
	"github.com/verrazzano/charts-poc/tools/vcm/cmd/patch"
	"github.com/verrazzano/charts-poc/tools/vcm/cmd/pull"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "vcm"
	helpShort   = "The vcm tool is a command-line utility that enables developers to pull and customize helm charts."
)

// NewRootCmd - create the root cobra command
func NewRootCmd(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpShort)
	// Add commands
	cmd.AddCommand(pull.NewCmdPull(vzHelper))
	cmd.AddCommand(diff.NewCmdDiff(vzHelper))
	cmd.AddCommand(patch.NewCmdPatch(vzHelper))
	return cmd
}

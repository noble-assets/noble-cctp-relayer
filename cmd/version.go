package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	// Version defines the application version (defined at compile time)
	Version = ""
	Commit  = ""
	Dirty   = ""
)

type versionInfo struct {
	Version string `json:"version" yaml:"version"`
	Commit  string `json:"commit" yaml:"commit"`
	Go      string `json:"go" yaml:"go"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the relayer version info",
	RunE:  getVersionCmd,
}

func getVersionCmd(cmd *cobra.Command, args []string) error {
	// jsn, err := cmd.Flags().GetBool(flagJSON)
	// if err != nil {
	// 	return err
	// }

	commit := Commit
	if Dirty != "0" {
		commit += " (dirty)"
	}

	verInfo := versionInfo{
		Version: Version,
		Commit:  commit,
		Go:      fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	}

	bz, err := yaml.Marshal(&verInfo)

	fmt.Fprintln(cmd.OutOrStdout(), string(bz))
	return err
}

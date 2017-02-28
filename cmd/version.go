package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints version of booster-web",
	Long:  `Prints version of booster-web`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("booster-web " + Version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

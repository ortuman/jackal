package command

import (
	"fmt"

	"github.com/ortuman/jackal/version"
	"github.com/spf13/cobra"
)

// NewVersionCommand prints out the version of jackal.
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Prints the version of etcdctl",
		Run:   versionCommandFunc,
	}
}

func versionCommandFunc(_ *cobra.Command, _ []string) {
	fmt.Println("jackalctl version:", version.Version)
	fmt.Println("API version:", version.APIVersion)
}

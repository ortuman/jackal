// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// copied from https://github.com/etcd-io/etcd/blob/master/etcdctl/ctlv3/ctl.go

package ctlv1

import (
	"time"

	"github.com/ortuman/jackal/cmd/jackalctl/ctlv1/command"
	"github.com/spf13/cobra"
)

const (
	cliName        = "jackalctl"
	cliDescription = "A simple command line client for jackal."

	defaultDialTimeout    = 2 * time.Second
	defaultCommandTimeOut = 5 * time.Second
)

var (
	globalFlags = command.GlobalFlags{}
)

var (
	rootCmd = &cobra.Command{
		Use:        cliName,
		Short:      cliDescription,
		SuggestFor: []string{"jackalctl"},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&globalFlags.Host, "host", "127.0.0.1:15280", "admin server host")

	rootCmd.PersistentFlags().DurationVar(&globalFlags.DialTimeout, "dial-timeout", defaultDialTimeout, "dial timeout for client connections")
	rootCmd.PersistentFlags().DurationVar(&globalFlags.CommandTimeOut, "command-timeout", defaultCommandTimeOut, "timeout for running command")

	rootCmd.AddCommand(
		command.NewUserCommand(),
		command.NewVersionCommand(),
	)
}

// Start status ctl command.
func Start() error {
	rootCmd.SetUsageFunc(usageFunc)
	// Make help just show the usage
	rootCmd.SetHelpTemplate(`{{.UsageString}}`)
	return rootCmd.Execute()
}

// MustStart is like Start but exiting in case an error occurs.
func MustStart() {
	if err := Start(); err != nil {
		command.ExitWithError(command.ExitError, err)
	}
}

func init() {
	cobra.EnablePrefixMatching = true
}

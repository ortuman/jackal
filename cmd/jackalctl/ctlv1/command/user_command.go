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

// copied from https://github.com/etcd-io/etcd/blob/master/etcdctl/ctlv3/command/user_command.go

package command

import (
	"fmt"
	"strings"

	"github.com/bgentry/speakeasy"
	adminpb "github.com/ortuman/jackal/admin/pb"
	"github.com/spf13/cobra"
)

var (
	passwordFromFlag    string
	passwordInteractive bool
)

// NewUserCommand returns the cobra command for "user".
func NewUserCommand() *cobra.Command {
	ac := &cobra.Command{
		Use:   "user <subcommand>",
		Short: "User related commands",
	}

	ac.AddCommand(newUserAddCommand())
	ac.AddCommand(newUserChangePasswordCommand())
	ac.AddCommand(newUserDeleteCommand())

	return ac
}

func newUserAddCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "add <user name or user:password> [options]",
		Short: "Adds a new user",
		Run:   userAddCommandFunc,
	}

	cmd.Flags().BoolVar(&passwordInteractive, "interactive", true, "Read password from stdin instead of interactive terminal")
	cmd.Flags().StringVar(&passwordFromFlag, "new-user-password", "", "Supply password from the command line flag")

	return &cmd
}

func newUserChangePasswordCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "passwd <user name> [options]",
		Short: "Changes password of user",
		Run:   userChangePasswordCommandFunc,
	}

	cmd.Flags().BoolVar(&passwordInteractive, "interactive", true, "If true, read password from stdin instead of interactive terminal")

	return &cmd
}

func newUserDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <user name>",
		Short: "Deletes a user",
		Run:   userDeleteCommandFunc,
	}
}

// userAddCommandFunc executes the "user add" command.
func userAddCommandFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		ExitWithError(ExitBadArgs, fmt.Errorf("user add command requires user name as its argument"))
	}

	var password string
	var username string

	if passwordFromFlag != "" {
		username = args[0]
		password = passwordFromFlag
	} else {
		splitted := strings.SplitN(args[0], ":", 2)
		if len(splitted) < 2 {
			username = args[0]
			if !passwordInteractive {
				_, _ = fmt.Scanf("%s", &password)
			} else {
				password = readPasswordInteractive(args[0])
			}
		} else {
			username = splitted[0]
			password = splitted[1]
			if len(username) == 0 {
				ExitWithError(ExitBadArgs, fmt.Errorf("empty user name is not allowed"))
			}
		}
	}
	cc, ctx, cancel := mustUsersClientFromCmd(cmd)
	defer cancel()

	resp, err := cc.CreateUser(ctx, &adminpb.CreateUserRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		ExitWithError(ExitError, err)
	}
	display.CreateUser(username, resp)
}

// userChangePasswordCommandFunc executes the "user passwd" command.
func userChangePasswordCommandFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		ExitWithError(ExitBadArgs, fmt.Errorf("user passwd command requires user name as its argument"))
	}
	username := args[0]

	var password string

	if !passwordInteractive {
		_, _ = fmt.Scanf("%s", &password)
	} else {
		password = readPasswordInteractive(args[0])
	}
	cc, ctx, cancel := mustUsersClientFromCmd(cmd)
	defer cancel()

	resp, err := cc.ChangeUserPassword(ctx, &adminpb.ChangeUserPasswordRequest{
		Username:    username,
		NewPassword: password,
	})
	if err != nil {
		ExitWithError(ExitError, err)
	}
	display.ChangeUserPassword(resp)
}

// userDeleteCommandFunc executes the "user delete" command.
func userDeleteCommandFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		ExitWithError(ExitBadArgs, fmt.Errorf("user delete command requires user name as its argument"))
	}
	username := args[0]

	cc, ctx, cancel := mustUsersClientFromCmd(cmd)
	defer cancel()

	resp, err := cc.DeleteUser(ctx, &adminpb.DeleteUserRequest{Username: username})
	if err != nil {
		ExitWithError(ExitError, err)
	}
	display.DeleteUser(username, resp)
}

func readPasswordInteractive(name string) string {
	prompt1 := fmt.Sprintf("Password of %s: ", name)
	password1, err1 := speakeasy.Ask(prompt1)
	if err1 != nil {
		ExitWithError(ExitBadArgs, fmt.Errorf("failed to ask password: %s", err1))
	}

	if len(password1) == 0 {
		ExitWithError(ExitBadArgs, fmt.Errorf("empty password"))
	}

	prompt2 := fmt.Sprintf("Type password of %s again for confirmation: ", name)
	password2, err2 := speakeasy.Ask(prompt2)
	if err2 != nil {
		ExitWithError(ExitBadArgs, fmt.Errorf("failed to ask password: %s", err2))
	}

	if password1 != password2 {
		ExitWithError(ExitBadArgs, fmt.Errorf("given passwords are different"))
	}

	return password1
}

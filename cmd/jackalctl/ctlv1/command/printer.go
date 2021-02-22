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

package command

import (
	"fmt"

	adminpb "github.com/ortuman/jackal/admin/pb"
)

type printer interface {
	CreateUser(name string, _ *adminpb.CreateUserResponse)
	ChangeUserPassword(*adminpb.ChangeUserPasswordResponse)
	DeleteUser(string, *adminpb.DeleteUserResponse)
}

type simplePrinter struct{}

func (p *simplePrinter) CreateUser(name string, _ *adminpb.CreateUserResponse) {
	fmt.Printf("User %s created\n", name)
}

func (p *simplePrinter) ChangeUserPassword(*adminpb.ChangeUserPasswordResponse) {
	fmt.Println("Password updated")
}

func (p *simplePrinter) DeleteUser(user string, _ *adminpb.DeleteUserResponse) {
	fmt.Printf("User %s deleted\n", user)
}

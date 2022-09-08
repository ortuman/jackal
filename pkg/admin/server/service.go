// Copyright 2022 The jackal Authors
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

package adminserver

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"

	kitlog "github.com/go-kit/log"

	"github.com/go-kit/log/level"

	userspb "github.com/ortuman/jackal/pkg/admin/pb"
	"github.com/ortuman/jackal/pkg/auth/pepper"
	"github.com/ortuman/jackal/pkg/hook"
	usermodel "github.com/ortuman/jackal/pkg/model/user"
	"github.com/ortuman/jackal/pkg/storage/repository"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const iterationCount = 15_000

type usersService struct {
	userspb.UnimplementedUsersServer
	rep     repository.Repository
	peppers *pepper.Keys
	hk      *hook.Hooks
	logger  kitlog.Logger
}

func newUsersService(rep repository.Repository, peppers *pepper.Keys, hk *hook.Hooks, logger kitlog.Logger) userspb.UsersServer {
	return &usersService{
		rep:     rep,
		peppers: peppers,
		hk:      hk,
		logger:  logger,
	}
}

func (s *usersService) CreateUser(ctx context.Context, req *userspb.CreateUserRequest) (*userspb.CreateUserResponse, error) {
	username := req.GetUsername()
	if err := s.ensureUserNotFound(ctx, username); err != nil {
		return nil, err
	}
	if err := s.upsertUser(ctx, username, req.GetPassword()); err != nil {
		return nil, err
	}
	// run user created hook
	_, err := s.hk.Run(hook.UserCreated, &hook.ExecutionContext{
		Info: &hook.UserInfo{
			Username: username,
		},
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}
	level.Info(s.logger).Log("msg", "user created", "username", username)
	return &userspb.CreateUserResponse{}, nil
}

func (s *usersService) ChangeUserPassword(ctx context.Context, req *userspb.ChangeUserPasswordRequest) (*userspb.ChangeUserPasswordResponse, error) {
	username := req.GetUsername()
	if err := s.ensureUserAlreadyExists(ctx, username); err != nil {
		return nil, err
	}
	if err := s.upsertUser(ctx, username, req.GetNewPassword()); err != nil {
		return nil, err
	}
	level.Info(s.logger).Log("msg", "password updated", "username", username)

	return &userspb.ChangeUserPasswordResponse{}, nil
}

func (s *usersService) DeleteUser(ctx context.Context, req *userspb.DeleteUserRequest) (*userspb.DeleteUserResponse, error) {
	username := req.GetUsername()
	if err := s.ensureUserAlreadyExists(ctx, username); err != nil {
		return nil, err
	}
	if err := s.rep.DeleteUser(ctx, username); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	// run user deleted hook
	_, err := s.hk.Run(hook.UserDeleted, &hook.ExecutionContext{
		Info: &hook.UserInfo{
			Username: username,
		},
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}
	level.Info(s.logger).Log("msg", "user deleted", "username", username)

	return &userspb.DeleteUserResponse{}, nil
}

func (s *usersService) ensureUserNotFound(ctx context.Context, username string) error {
	exists, err := s.rep.UserExists(ctx, username)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	if exists {
		return status.Errorf(codes.AlreadyExists, fmt.Sprintf("user %s already exists", username))
	}
	return nil
}

func (s *usersService) ensureUserAlreadyExists(ctx context.Context, username string) error {
	exists, err := s.rep.UserExists(ctx, username)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return status.Errorf(codes.NotFound, fmt.Sprintf("user %s not found", username))
	}
	return nil
}

func (s *usersService) upsertUser(ctx context.Context, username, password string) error {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	buf := bytes.NewBuffer(salt)
	pp := s.peppers.GetActiveKey()
	buf.WriteString(pp)
	pepperedSalt := buf.Bytes()

	// generate password hashes
	hSHA1 := hashPassword([]byte(password), pepperedSalt, iterationCount, sha1.Size, sha1.New)
	hSHA256 := hashPassword([]byte(password), pepperedSalt, iterationCount, sha256.Size, sha256.New)
	hSHA512 := hashPassword([]byte(password), pepperedSalt, iterationCount, sha512.Size, sha512.New)
	hSHA3512 := hashPassword([]byte(password), pepperedSalt, iterationCount, sha512.Size, sha3.New512)

	usr := usermodel.User{
		Username: username,
		Scram:    &usermodel.Scram{},
	}
	usr.Scram.Sha1 = base64.RawURLEncoding.EncodeToString(hSHA1)
	usr.Scram.Sha256 = base64.RawURLEncoding.EncodeToString(hSHA256)
	usr.Scram.Sha512 = base64.RawURLEncoding.EncodeToString(hSHA512)
	usr.Scram.Sha3512 = base64.RawURLEncoding.EncodeToString(hSHA3512)
	usr.Scram.Salt = base64.RawURLEncoding.EncodeToString(salt)
	usr.Scram.IterationCount = iterationCount
	usr.Scram.PepperId = s.peppers.GetActiveID()

	if err := s.rep.UpsertUser(ctx, &usr); err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	return nil
}

func hashPassword(password, salt []byte, iterations int, hKeyLen int, h func() hash.Hash) []byte {
	return pbkdf2.Key(password, salt, iterations, hKeyLen, h)
}

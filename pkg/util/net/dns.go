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

package net

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// SRVResolver defines a SRV dns resolver.
type SRVResolver struct {
	r *net.Resolver
}

// NewSRVResolver creates and returns an initialized SRVResolver instance.
func NewSRVResolver() *SRVResolver {
	r := &net.Resolver{
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: time.Second * 5}
			return d.DialContext(ctx, "tcp", address)
		},
	}
	return &SRVResolver{r: r}
}

// Resolve performs SRV resolution over dns.
func (r *SRVResolver) Resolve(ctx context.Context, service, proto, remoteAddr string) ([]string, error) {
	var retVal []string

	_, addrs, err := r.r.LookupSRV(ctx, service, proto, remoteAddr)
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if addr.Target == "." {
			continue
		}
		host := strings.TrimSuffix(addr.Target, ".")
		port := strconv.Itoa(int(addr.Port))

		retVal = append(retVal, fmt.Sprintf("%s:%s", host, port))
	}
	return retVal, nil
}

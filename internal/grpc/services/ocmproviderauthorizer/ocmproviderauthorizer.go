// Copyright 2018-2020 CERN
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
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package ocmproviderauthorizer

import (
	"context"
	"fmt"

	providerpb "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	"github.com/cs3org/reva/pkg/ocm/provider"
	"github.com/cs3org/reva/pkg/ocm/provider/authorizer/registry"
	"github.com/cs3org/reva/pkg/rgrpc"
	"github.com/cs3org/reva/pkg/rgrpc/status"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func init() {
	rgrpc.Register("ocmproviderauthorizer", New)
}

type config struct {
	Driver  string                            `mapstructure:"driver"`
	Drivers map[string]map[string]interface{} `mapstructure:"drivers"`
}

type service struct {
	conf *config
	pa   provider.Authorizer
}

func getProviderAuthorizer(c *config) (provider.Authorizer, error) {
	if f, ok := registry.NewFuncs[c.Driver]; ok {
		return f(c.Drivers[c.Driver])
	}
	return nil, fmt.Errorf("driver not found: %s", c.Driver)
}

func (s *service) Close() error {
	return nil
}

func (s *service) UnprotectedEndpoints() []string {
	return []string{"/cs3.ocm.provider.v1beta1.ProviderAPI/IsProviderAllowed"}
}

func (s *service) Register(ss *grpc.Server) {
	providerpb.RegisterProviderAPIServer(ss, s)
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

// New creates a new OCM provider authorizer svc
func New(m map[string]interface{}, ss *grpc.Server) (rgrpc.Service, error) {

	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	// if driver is empty we default to json
	if c.Driver == "" {
		c.Driver = "json"
	}

	pa, err := getProviderAuthorizer(c)
	if err != nil {
		return nil, err
	}

	service := &service{
		conf: c,
		pa:   pa,
	}
	return service, nil
}

func (s *service) GetInfoByDomain(ctx context.Context, req *providerpb.GetInfoByDomainRequest) (*providerpb.GetInfoByDomainResponse, error) {
	domainInfo, err := s.pa.GetInfoByDomain(ctx, req.Domain)
	if err != nil {
		return &providerpb.GetInfoByDomainResponse{
			Status: status.NewInternal(ctx, err, "error getting provider info"),
		}, nil
	}

	return &providerpb.GetInfoByDomainResponse{
		Status:       status.NewOK(ctx),
		ProviderInfo: domainInfo,
	}, nil
}

func (s *service) IsProviderAllowed(ctx context.Context, req *providerpb.IsProviderAllowedRequest) (*providerpb.IsProviderAllowedResponse, error) {
	err := s.pa.IsProviderAllowed(ctx, req.User)
	if err != nil {
		return &providerpb.IsProviderAllowedResponse{
			Status: status.NewInternal(ctx, err, "error verifying mesh provider"),
		}, nil
	}

	return &providerpb.IsProviderAllowedResponse{
		Status: status.NewOK(ctx),
	}, nil
}

func (s *service) ListAllProviders(ctx context.Context, req *providerpb.ListAllProvidersRequest) (*providerpb.ListAllProvidersResponse, error) {
	providers, err := s.pa.ListAllProviders(ctx)
	if err != nil {
		return &providerpb.ListAllProvidersResponse{
			Status: status.NewInternal(ctx, err, "error retrieving mesh providers"),
		}, nil
	}

	return &providerpb.ListAllProvidersResponse{
		Status:    status.NewOK(ctx),
		Providers: providers,
	}, nil
}

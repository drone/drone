// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"context"

	"github.com/drone/drone/core"
	"github.com/drone/drone/logger"
	"github.com/drone/drone/plugin/registry/auths"
)

// Static returns a new static credentials controller.
func Static(secretService core.SecretService) core.RegistryService {
	return &staticController{secretService: secretService}
}

type staticController struct {
	secretService core.SecretService
}

func (c *staticController) List(ctx context.Context, in *core.RegistryArgs) ([]*core.Registry, error) {
	var results []*core.Registry
	for _, name := range in.Pipeline.PullSecrets {
		logger := logger.FromContext(ctx).WithField("name", name)
		logger.Trace("registry: database: find secret");
		
		var args = core.SecretArgs{
			Name: name,
			Repo:  in.Repo,
			Build: in.Build,
			Conf: in.Conf,
		};

		secret, err := c.secretService.Find(ctx, &args);
		
		if err != nil {
			logger.WithError(err).Error("registry: database: finding secret error")
			return nil, err
		}

		if secret == nil {
			logger.Trace("registry: database: cannot find secret")
			continue
		}
		
		logger.Trace("registry: database: secret found")
		parsed, err := auths.ParseString(secret.Data)
		if err != nil {
			logger.WithError(err).Error("registry: database: parsing error")
			return nil, err
		}

		results = append(results, parsed...)
	}
	return results, nil
}
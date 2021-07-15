// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package converter

import (
	"context"
	"github.com/drone/go-scm/scm"
	"strings"

	"github.com/drone/drone/core"
	"github.com/drone/drone/plugin/converter/jsonnet"
)

// TODO(bradrydzewski) handle jsonnet imports
// TODO(bradrydzewski) handle jsonnet object vs array output

// Jsonnet returns a conversion service that converts the
// jsonnet file to a yaml file.
func Jsonnet(enabled bool, client *scm.Client) core.ConvertService {
	return &jsonnetPlugin{
		enabled: enabled,
		client:  client,
	}
}

type jsonnetPlugin struct {
	enabled bool
	client  *scm.Client
}

func (p *jsonnetPlugin) Convert(ctx context.Context, req *core.ConvertArgs) (*core.Config, error) {
	if p.enabled == false {
		return nil, nil
	}

	// if the file extension is not jsonnet we can
	// skip this plugin by returning zero values.
	if strings.HasSuffix(req.Repo.Config, ".jsonnet") == false {
		return nil, nil
	}

	file, err := jsonnet.Parse(req, p.client, nil, nil)

	if err != nil {
		return nil, err
	}
	return &core.Config{
		Data: file,
	}, nil
}

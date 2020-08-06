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

package core

import (
	"context"
	"errors"
	"time"
)

// License types.
const (
	LicenseFoss     = "foss"
	LicensePersonal = "personal"
	LicenseStandard = "standard"
	LicenseTrial    = "trial"
)

// ErrUserLimit is returned when attempting to create a new
// user but the maximum number of allowed user accounts
// is exceeded.
var ErrUserLimit = errors.New("User limit exceeded")

// ErrRepoLimit is returned when attempting to create a new
// repository but the maximum number of allowed repositories
// is exceeded.
var ErrRepoLimit = errors.New("Repository limit exceeded")

// ErrBuildLimit is returned when attempting to create a new
// build but the maximum number of allowed builds is exceeded.
var ErrBuildLimit = errors.New("Build limit exceeded")

type (
	// License defines software license parameters.
	License struct {
		Licensor     string    `json:"-"`
		Subscription string    `json:"-"`
		Expires      time.Time `json:"expires_at,omitempty"`
		Kind         string    `json:"kind,omitempty"`
		Repos        int64     `json:"repos,omitempty"`
		Users        int64     `json:"users,omitempty"`
		Builds       int64     `json:"builds,omitempty"`
		Nodes        int64     `json:"nodes,omitempty"`
	}

	// LicenseService provides access to the license
	// service and can be used to check for violations
	// and expirations.
	LicenseService interface {
		// Exceeded returns true if the system has exceeded
		// its limits as defined in the license.
		Exceeded(context.Context) (bool, error)

		// Expired returns true if the license is expired.
		Expired(context.Context) bool
	}
)

// Expired returns true if the license is expired.
func (l *License) Expired() bool {
	return l.Expires.IsZero() == false && time.Now().After(l.Expires)
}

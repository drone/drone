// Copyright 2018 Drone.IO Inc.
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

package version

import (
	"fmt"
	"os"
	"runtime"

	"github.com/coreos/go-semver/semver"
	"github.com/rs/zerolog/log"
)

var (
	// VersionMajor is for an API incompatible changes.
	VersionMajor int64
	// VersionMinor is for functionality in a backwards-compatible manner.
	VersionMinor int64 = 8
	// VersionPatch is for backwards-compatible bug fixes.
	VersionPatch int64 = 9
	// VersionPre indicates prerelease.
	VersionPre string
	// VersionDev indicates development branch. Releases will be empty string.
	VersionDev string
	// BuildDate is the ISO 8601 day drone was built.
	BuildDate string
)

// Version is the specification version that the package types support.
var Version = semver.Version{
	Major:      VersionMajor,
	Minor:      VersionMinor,
	Patch:      VersionPatch,
	PreRelease: semver.PreRelease(VersionPre),
	Metadata:   VersionDev,
}

func PrintVersion(logOutput bool) {
	output := fmt.Sprintf("Running %v version %s, built on %s, %s", os.Args[0], Version, BuildDate, runtime.Version())
	if !logOutput {
		fmt.Fprintf(os.Stderr, output)
	} else {
		log.Info().Msg(output)
	}
}

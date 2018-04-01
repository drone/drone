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

package model

import (
	"encoding/xml"
	"strconv"
	"time"
)

type CCProjects struct {
	XMLName  xml.Name     `xml:"Projects"`
	Projects []*CCProject `xml:"Project"`
}

type CCProject struct {
	XMLName         xml.Name `xml:"Project"`
	Name            string   `xml:"name,attr"`
	Activity        string   `xml:"activity,attr"`
	LastBuildStatus string   `xml:"lastBuildStatus,attr"`
	LastBuildLabel  string   `xml:"lastBuildLabel,attr"`
	LastBuildTime   string   `xml:"lastBuildTime,attr"`
	WebURL          string   `xml:"webUrl,attr"`
}

func NewCC(r *Repo, bs []*Build, link string) *CCProjects {
	projs := &CCProjects{Projects: []*CCProject{}}

	pz := NewCCProject(r, bs[0], r.FullName, link)
	projs.Projects = append(projs.Projects, pz)

	branches := []string{}
BuildLoop:
	for _, b := range bs {
		for _, br := range branches {
			if br == b.Branch {
				continue BuildLoop
			}
		}

		p := NewCCProject(r, b, r.FullName+" "+b.Branch, link)

		projs.Projects = append(projs.Projects, p)
		branches = append(branches, b.Branch)
	}
	return projs
}

func NewCCProject(r *Repo, b *Build, name, link string) *CCProject {
	proj := &CCProject{
		Name:            name,
		WebURL:          link,
		Activity:        "Building",
		LastBuildStatus: "Unknown",
		LastBuildLabel:  "Unknown",
	}
	// if the build is not currently running then
	// we can return the latest build status.
	if b.Status != StatusPending &&
		b.Status != StatusRunning {
		proj.Activity = "Sleeping"
		proj.LastBuildTime = time.Unix(b.Started, 0).Format(time.RFC3339)
		proj.LastBuildLabel = strconv.Itoa(b.Number)
	}
	// ensure the last build Status accepts a valid
	// ccmenu enumeration
	switch b.Status {
	case StatusError, StatusKilled:
		proj.LastBuildStatus = "Exception"
	case StatusSuccess:
		proj.LastBuildStatus = "Success"
	case StatusFailure:
		proj.LastBuildStatus = "Failure"
	}
	return proj
}

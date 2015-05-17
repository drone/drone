package ccmenu

import (
	"encoding/xml"
	"strconv"
	"time"

	"github.com/drone/drone/pkg/types"
)

type CCProjects struct {
	XMLName xml.Name   `xml:"Projects"`
	Project *CCProject `xml:"Project"`
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

func NewCC(r *types.Repo, c *types.Commit) *CCProjects {
	proj := &CCProject{
		Name:            r.Owner + "/" + r.Name,
		WebURL:          r.Self,
		Activity:        "Building",
		LastBuildStatus: "Unknown",
		LastBuildLabel:  "Unknown",
	}

	// if the build is not currently running then
	// we can return the latest build status.
	if c.State != types.StatePending &&
		c.State != types.StateRunning {
		proj.Activity = "Sleeping"
		proj.LastBuildTime = time.Unix(c.Started, 0).Format(time.RFC3339)
		proj.LastBuildLabel = strconv.Itoa(c.Sequence)
	}

	// ensure the last build state accepts a valid
	// ccmenu enumeration
	switch c.State {
	case types.StateError, types.StateKilled:
		proj.LastBuildStatus = "Exception"
	case types.StateSuccess:
		proj.LastBuildStatus = "Success"
	case types.StateFailure:
		proj.LastBuildStatus = "Failure"
	default:
		proj.LastBuildStatus = "Unknown"
	}

	return &CCProjects{Project: proj}
}

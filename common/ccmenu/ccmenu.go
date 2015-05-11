package ccmenu

import (
	"encoding/xml"
	"strconv"
	"time"

	"github.com/drone/drone/common"
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

func NewCC(r *common.Repo, c *common.Commit, url string) *CCProjects {
	proj := &CCProject{
		Name:            r.Owner + "/" + r.Name,
		WebURL:          url,
		Activity:        "Building",
		LastBuildStatus: "Unknown",
		LastBuildLabel:  "Unknown",
	}

	// if the build is not currently running then
	// we can return the latest build status.
	if c.State != common.StatePending &&
		c.State != common.StateRunning {
		proj.Activity = "Sleeping"
		proj.LastBuildTime = time.Unix(c.Started, 0).Format(time.RFC3339)
		proj.LastBuildLabel = strconv.Itoa(c.Sequence)
	}

	// ensure the last build state accepts a valid
	// ccmenu enumeration
	switch c.State {
	case common.StateError, common.StateKilled:
		proj.LastBuildStatus = "Exception"
	case common.StateSuccess:
		proj.LastBuildStatus = "Success"
	case common.StateFailure:
		proj.LastBuildStatus = "Failure"
	default:
		proj.LastBuildStatus = "Unknown"
	}

	return &CCProjects{Project: proj}
}

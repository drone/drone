package parser

import (
	"github.com/drone/drone/common"
	"github.com/drone/drone/parser/inject"
	"github.com/drone/drone/parser/matrix"

	"gopkg.in/yaml.v2"
)

// Opts specifies parser options that will permit
// or deny certain Yaml settings.
type Opts struct {
	Volumes    bool
	Network    bool
	Privileged bool
}

var defaultOpts = &Opts{
	Volumes:    false,
	Network:    false,
	Privileged: false,
}

// Parse parses a build matrix and returns
// a list of build configurations for each axis
// using the default parsing options.
func Parse(raw string) ([]*common.Config, error) {
	return ParseOpts(raw, defaultOpts)
}

// ParseOpts parses a build matrix and returns
// a list of build configurations for each axis
// using the provided parsing options.
func ParseOpts(raw string, opts *Opts) ([]*common.Config, error) {
	confs, err := parse(raw)
	if err != nil {
		return nil, err
	}
	for _, conf := range confs {
		err := Lint(conf)
		if err != nil {
			return nil, err
		}
		transformSetup(conf)
		transformClone(conf)
		transformBuild(conf)
		transformImages(conf)
		transformDockerPlugin(conf)
		if !opts.Network {
			rmNetwork(conf)
		}
		if !opts.Volumes {
			rmVolumes(conf)
		}
		if !opts.Privileged {
			rmPrivileged(conf)
		}
	}
	return confs, nil
}

// helper function to parse a matrix configuraiton file.
func parse(raw string) ([]*common.Config, error) {
	axis, err := matrix.Parse(raw)
	if err != nil {
		return nil, err
	}
	confs := []*common.Config{}

	// when no matrix values exist we should return
	// a single config value with an empty label.
	if len(axis) == 0 {
		conf, err := parseYaml(raw)
		if err != nil {
			return nil, err
		}
		confs = append(confs, conf)
	}

	for _, ax := range axis {
		// inject the matrix values into the raw script
		injected := inject.Inject(raw, ax)
		conf, err := parseYaml(injected)
		if err != nil {
			return nil, err
		}
		conf.Axis = common.Axis(ax)
		confs = append(confs, conf)
	}
	return confs, nil
}

// helper funtion to parse a yaml configuration file.
func parseYaml(raw string) (*common.Config, error) {
	conf := &common.Config{}
	err := yaml.Unmarshal([]byte(raw), conf)
	return conf, err
}

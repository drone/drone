// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package nomad

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/scheduler/internal"

	"github.com/dchest/uniuri"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/sirupsen/logrus"
)

var _ core.Scheduler = (*nomadScheduler)(nil)

// Docker host.
const (
	dockerHostPosix   = "/var/run/docker.sock"
	dockerHostWindows = "////./pipe/docker_engine"
)

type nomadScheduler struct {
	client *api.Client
	config Config
}

// FromConfig returns a new Nomad scheduler.
func FromConfig(conf Config) (core.Scheduler, error) {
	config := api.DefaultConfig()
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &nomadScheduler{client: client, config: conf}, nil
}

// Schedule schedules the stage for execution.
func (s *nomadScheduler) Schedule(ctx context.Context, stage *core.Stage) error {
	env := map[string]string{
		"DRONE_RUNNER_PRIVILEGED_IMAGES": strings.Join(s.config.DockerImagePriv, ","),
		"DRONE_LIMIT_MEM":                fmt.Sprint(s.config.LimitMemory),
		"DRONE_LIMIT_CPU":                fmt.Sprint(s.config.LimitCompute),
		"DRONE_STAGE_ID":                 fmt.Sprint(stage.ID),
		"DRONE_LOGS_DEBUG":               fmt.Sprint(s.config.LogDebug),
		"DRONE_LOGS_TRACE":               fmt.Sprint(s.config.LogTrace),
		"DRONE_LOGS_PRETTY":              fmt.Sprint(s.config.LogPretty),
		"DRONE_LOGS_TEXT":                fmt.Sprint(s.config.LogText),
		"DRONE_RPC_PROTO":                s.config.CallbackProto,
		"DRONE_RPC_HOST":                 s.config.CallbackHost,
		"DRONE_RPC_SECRET":               s.config.CallbackSecret,
		"DRONE_RPC_DEBUG":                fmt.Sprint(s.config.LogTrace),
		"DRONE_REGISTRY_ENDPOINT":        s.config.RegistryEndpoint,
		"DRONE_REGISTRY_SECRET":          s.config.RegistryToken,
		"DRONE_REGISTRY_SKIP_VERIFY":     fmt.Sprint(s.config.RegistryInsecure),
		"DRONE_SECRET_ENDPOINT":          s.config.SecretEndpoint,
		"DRONE_SECRET_SECRET":            s.config.SecretToken,
		"DRONE_SECRET_SKIP_VERIFY":       fmt.Sprint(s.config.SecretInsecure),
	}

	volume := "/var/run/docker.sock:/var/run/docker.sock"
	if stage.OS == "windows" {
		volume = "////./pipe/docker_engine:////./pipe/docker_engine"
	}

	task := &api.Task{
		Name:      "stage",
		Driver:    "docker",
		Env:       env,
		Resources: &api.Resources{},
		Config: map[string]interface{}{
			"image":      internal.DefaultImage(s.config.DockerImage),
			"force_pull": s.config.DockerImagePull,
			"volumes":    []string{volume},
		},
	}

	if i := s.config.RequestCompute; i != 0 {
		task.Resources.CPU = intToPtr(i)
	}
	if i := s.config.RequestMemory; i != 0 {
		task.Resources.MemoryMB = intToPtr(i)
	}

	rand := uniuri.NewLen(12)
	name := fmt.Sprintf("drone-job-%d-%s", stage.BuildID, rand)

	job := &api.Job{
		ID:          stringToPtr(name),
		Name:        stringToPtr(name),
		Type:        stringToPtr("batch"),
		Datacenters: s.config.Datacenter,
		TaskGroups: []*api.TaskGroup{
			{
				Name:  stringToPtr("pipeline"),
				Tasks: []*api.Task{task},
				RestartPolicy: &api.RestartPolicy{
					Mode: stringToPtr("fail"),
				},
			},
		},
		Meta: map[string]string{
			"io.drone":                 "true",
			"io.drone.stage.created":   time.Unix(stage.Created, 0).String(),
			"io.drone.stage.scheduled": time.Now().String(),
			"io.drone.stage.id":        fmt.Sprint(stage.ID),
			"io.drone.stage.number":    fmt.Sprint(stage.Number),
			"io.drone.stage.os":        fmt.Sprint(stage.OS),
			"io.drone.stage.arch":      fmt.Sprint(stage.Arch),
			"io.drone.build.id":        fmt.Sprint(stage.BuildID),
			"io.drone.repo.id":         fmt.Sprint(stage.RepoID),
		},
	}

	if s := s.config.Namespace; s != "" {
		job.Namespace = stringToPtr(s)
	}
	if s := s.config.Region; s != "" {
		job.Region = stringToPtr(s)
	}

	// if we are running on darwin we disable os and arch
	// constraints, since it is possible nomad is running
	// on the host machine and reports a darwin os, instead
	// of a linux os.
	if runtime.GOOS != "darwin" {
		job.Constraints = []*api.Constraint{
			{
				LTarget: "${attr.kernel.name}",
				RTarget: stage.OS,
				Operand: "=",
			},
			{
				LTarget: "${attr.cpu.arch}",
				RTarget: stage.Arch,
				Operand: "=",
			},
		}
	}

	for k, v := range stage.Labels {
		job.Constraints = append(job.Constraints, &api.Constraint{
			LTarget: fmt.Sprintf("${meta.%s}", k),
			RTarget: v,
			Operand: "=",
		})
	}

	for k, v := range s.config.Labels {
        job.Constraints = append(job.Constraints, &api.Constraint{
            LTarget: fmt.Sprintf("${meta.%s}", k),
            RTarget: v,
            Operand: "=",
        })
    }

	log := logrus.WithFields(logrus.Fields{
		"stage-id":     stage.ID,
		"stage-number": stage.Number,
		"stage-name":   stage.Name,
		"repo-id":      stage.RepoID,
		"build-id":     stage.BuildID,
	})

	log.Debugf("nomad: creating job")
	_, _, err := s.client.Jobs().RegisterOpts(job, &api.RegisterOptions{}, nil)
	if err != nil {
		log.WithError(err).Errorln("nomad: cannot create job")
	} else {
		log.WithField("job-id", job.ID).Debugf("nomad: successfully created job")
	}

	return err
}

func (s *nomadScheduler) Request(context.Context, core.Filter) (*core.Stage, error) {
	return nil, errors.New("not implemented")
}

// Cancel cancels a scheduled or running stage.
func (s *nomadScheduler) Cancel(ctx context.Context, id int64) error {
	prefix := fmt.Sprintf("drone-job-%d-", id)
	jobs, _, err := s.client.Jobs().PrefixList(prefix)
	if err != nil {
		return err
	}

	var result error
	for _, job := range jobs {
		_, _, err := s.client.Jobs().Deregister(job.ID, false, nil)
		if err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}

func (s *nomadScheduler) Cancelled(context.Context, int64) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *nomadScheduler) Stats(context.Context) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (s *nomadScheduler) Pause(context.Context) error {
	return errors.New("not implemented")
}

func (s *nomadScheduler) Resume(context.Context) error {
	return errors.New("not implemented")
}

// stringToPtr returns the pointer to a string
func stringToPtr(str string) *string {
	return &str
}

// intToPtr returns the pointer to a int
func intToPtr(i int) *int {
	return &i
}

// durationToPtr returns the pointer to a duration
func durationToPtr(dur time.Duration) *time.Duration {
	return &dur
}

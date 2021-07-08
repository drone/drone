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

package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/drone/drone-runtime/engine"
	"github.com/drone/drone-runtime/runtime"
	"github.com/drone/drone-yaml/yaml"
	"github.com/drone/drone-yaml/yaml/compiler"
	"github.com/drone/drone-yaml/yaml/compiler/transform"
	"github.com/drone/drone-yaml/yaml/converter"
	"github.com/drone/drone-yaml/yaml/linter"
	"github.com/drone/drone/core"
	"github.com/drone/drone/operator/manager"
	"github.com/drone/drone/plugin/registry"
	"github.com/drone/drone/plugin/secret"
	"github.com/drone/drone/store/shared/db"
	"github.com/drone/envsubst"
	"golang.org/x/sync/errgroup"

	"github.com/sirupsen/logrus"
)

// Limits defines runtime container limits.
type Limits struct {
	MemSwapLimit int64
	MemLimit     int64
	ShmSize      int64
	CPUQuota     int64
	CPUShares    int64
	CPUSet       string
}

// Runner is responsible for retrieving and executing builds, and
// reporting back their status to the central server.
type Runner struct {
	sync.Mutex

	Engine     engine.Engine
	Manager    manager.BuildManager
	Registry   core.RegistryService
	Secrets    core.SecretService
	Limits     Limits
	Volumes    []string
	Networks   []string
	Devices    []string
	Privileged []string
	Environ    map[string]string
	Machine    string
	Labels     map[string]string

	Kind     string
	Type     string
	Platform string
	OS       string
	Arch     string
	Kernel   string
	Variant  string
}

func (r *Runner) handleError(ctx context.Context, stage *core.Stage, err error) error {
	switch stage.Status {
	case core.StatusPending,
		core.StatusRunning:
	default:
	}
	for _, step := range stage.Steps {
		if step.Status == core.StatusPending {
			step.Status = core.StatusSkipped
		}
		if step.Status == core.StatusRunning {
			step.Status = core.StatusPassing
			step.Stopped = time.Now().Unix()
		}
	}
	stage.Status = core.StatusError
	stage.Error = err.Error()
	stage.Stopped = time.Now().Unix()
	switch v := err.(type) {
	case *runtime.ExitError:
		stage.Error = ""
		stage.Status = core.StatusFailing
		stage.ExitCode = v.Code
	case *runtime.OomError:
		stage.Error = "OOM kill signaled by host operating system"
	}
	return r.Manager.AfterAll(ctx, stage)
}

//
// this is a quick copy-paste duplicate of above that
// removes some code. this is for testing purposes only.
//

func (r *Runner) Run(ctx context.Context, id int64) error {
	logger := logrus.WithFields(
		logrus.Fields{
			"machine":  r.Machine,
			"os":       r.OS,
			"arch":     r.Arch,
			"stage-id": id,
		},
	)

	logger.Debug("runner: get stage details from server")

	defer func() {
		// taking the paranoid approach to recover from
		// a panic that should absolutely never happen.
		if r := recover(); r != nil {
			logger.Errorf("runner: unexpected panic: %s", r)
			debug.PrintStack()
		}
	}()

	m, err := r.Manager.Details(ctx, id)
	if err != nil {
		logger.WithError(err).Warnln("runner: cannot get stage details")
		return err
	}

	logger = logger.WithFields(
		logrus.Fields{
			"repo":  m.Repo.Slug,
			"build": m.Build.Number,
			"stage": m.Stage.Number,
		},
	)

	netrc, err := r.Manager.Netrc(ctx, m.Repo.ID)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("runner: cannot get netrc file")
		return r.handleError(ctx, m.Stage, err)
	}
	if netrc == nil {
		netrc = new(core.Netrc)
	}

	if m.Build.Status == core.StatusKilled || m.Build.Status == core.StatusSkipped {
		logger = logger.WithError(err)
		logger.Infoln("runner: cannot run a canceled build")
		return nil
	}

	environ := combineEnviron(
		agentEnviron(r),
		buildEnviron(m.Build),
		repoEnviron(m.Repo),
		stageEnviron(m.Stage),
		systemEnviron(m.System),
		linkEnviron(m.Repo, m.Build, m.System),
		m.Build.Params,
	)

	//
	// parse configuration file
	//

	//
	// TODO extract the yaml document by index
	// TODO mutate the yaml
	//

	// this code is temporarily in place to detect and convert
	// the legacy yaml configuration file to the new format.
	y, err := converter.ConvertString(string(m.Config.Data), converter.Metadata{
		Filename: m.Repo.Config,
		URL:      m.Repo.Link,
		Ref:      m.Build.Ref,
	})

	if err != nil {
		return err
	}

	y, err = envsubst.Eval(y, func(name string) string {
		env := environ[name]
		if strings.Contains(env, "\n") {
			env = fmt.Sprintf("%q", env)
		}
		return env
	})
	if err != nil {
		return r.handleError(ctx, m.Stage, err)
	}

	manifest, err := yaml.ParseString(y)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("runner: cannot parse yaml")
		return r.handleError(ctx, m.Stage, err)
	}

	var pipeline *yaml.Pipeline
	for _, resource := range manifest.Resources {
		v, ok := resource.(*yaml.Pipeline)
		if !ok {
			continue
		}
		if v.Name == m.Stage.Name {
			pipeline = v
			break
		}
	}
	if pipeline == nil {
		logger = logger.WithError(err)
		logger.Errorln("runner: cannot find named pipeline")
		return r.handleError(ctx, m.Stage,
			errors.New("cannot find named pipeline"),
		)
	}

	logger = logger.WithField("pipeline", pipeline.Name)

	err = linter.Lint(pipeline, m.Repo.Trusted)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("runner: yaml lint errors")
		return r.handleError(ctx, m.Stage, err)
	}

	secretService := secret.Combine(
		secret.Encrypted(),
		secret.Static(m.Secrets),
		r.Secrets,
	)
	registryService := registry.Combine(
		registry.Static(secretService),
		r.Registry,
	)

	comp := new(compiler.Compiler)
	comp.PrivilegedFunc = compiler.DindFunc(
		append(
			r.Privileged,
			"plugins/docker",
			"plugins/acr",
			"plugins/ecr",
			"plugins/gcr",
			"plugins/heroku",
		),
	)
	comp.SkipFunc = compiler.SkipFunc(
		compiler.SkipData{
			Action:   m.Build.Action,
			Branch:   m.Build.Target,
			Cron:     m.Build.Cron,
			Event:    m.Build.Event,
			Instance: m.System.Host,
			Ref:      m.Build.Ref,
			Repo:     m.Repo.Slug,
			Target:   m.Build.Deploy,
		},
	)
	comp.TransformFunc = transform.Combine(
		// transform.Include(),
		// transform.Exclude(),
		// transform.ResumeAt(),
		transform.WithAuthsFunc(
			func() []*engine.DockerAuth {
				in := &core.RegistryArgs{
					Build:    m.Build,
					Repo:     m.Repo,
					Conf:     manifest,
					Pipeline: pipeline,
				}
				out, err := registryService.List(ctx, in)
				if err != nil {
					return nil
				}
				return convertRegistry(out)
			},
		),
		transform.WithEnviron(environ),
		transform.WithEnviron(r.Environ),
		transform.WithLables(
			map[string]string{
				"io.drone":                "true",
				"io.drone.build.number":   fmt.Sprint(m.Build.Number),
				"io.drone.repo.namespace": m.Repo.Namespace,
				"io.drone.repo.name":      m.Repo.Name,
				"io.drone.stage.name":     m.Stage.Name,
				"io.drone.stage.number":   fmt.Sprint(m.Stage.Number),
				"io.drone.ttl":            fmt.Sprint(time.Duration(m.Repo.Timeout) * time.Minute),
				"io.drone.expires":        fmt.Sprint(time.Now().Add(time.Duration(m.Repo.Timeout)*time.Minute + time.Hour).Unix()),
				"io.drone.created":        fmt.Sprint(time.Now().Unix()),
				"io.drone.protected":      "false",
			},
		), // TODO append labels here
		transform.WithLimits(
			r.Limits.MemLimit,
			0, // no clue how to apply the docker cpu limit
		),
		transform.WithNetrc(
			netrc.Machine,
			netrc.Login,
			netrc.Password,
		),
		transform.WithNetworks(r.Networks),
		transform.WithProxy(),
		transform.WithSecretFunc(
			func(name string) *engine.Secret {
				in := &core.SecretArgs{
					Name:  name,
					Build: m.Build,
					Repo:  m.Repo,
					Conf:  manifest,
				}
				out, err := secretService.Find(ctx, in)
				if err != nil {
					return nil
				}
				if out == nil {
					return nil
				}
				return &engine.Secret{
					Metadata: engine.Metadata{Name: name},
					Data:     out.Data,
				}
			},
		),
		transform.WithVolumes(
			convertVolumes(r.Volumes),
		),
	)
	ir := comp.Compile(pipeline)

	steps := map[string]*core.Step{}
	i := 0
	for _, s := range ir.Steps {
		if s.RunPolicy == engine.RunNever {
			continue
		}
		i++
		dst := &core.Step{
			Number:    i,
			Name:      s.Metadata.Name,
			StageID:   m.Stage.ID,
			Status:    core.StatusPending,
			ErrIgnore: s.IgnoreErr,
		}
		steps[dst.Name] = dst
		m.Stage.Steps = append(m.Stage.Steps, dst)
	}

	hooks := &runtime.Hook{
		BeforeEach: func(s *runtime.State) error {
			r.Lock()
			s.Step.Envs["DRONE_MACHINE"] = r.Machine
			s.Step.Envs["CI_BUILD_STATUS"] = "success"
			s.Step.Envs["CI_BUILD_STARTED"] = strconv.FormatInt(s.Runtime.Time, 10)
			s.Step.Envs["CI_BUILD_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)
			s.Step.Envs["DRONE_BUILD_STATUS"] = "success"
			s.Step.Envs["DRONE_BUILD_STARTED"] = strconv.FormatInt(s.Runtime.Time, 10)
			s.Step.Envs["DRONE_BUILD_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)

			s.Step.Envs["CI_JOB_STATUS"] = "success"
			s.Step.Envs["CI_JOB_STARTED"] = strconv.FormatInt(s.Runtime.Time, 10)
			s.Step.Envs["CI_JOB_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)
			s.Step.Envs["DRONE_JOB_STATUS"] = "success"
			s.Step.Envs["DRONE_JOB_STARTED"] = strconv.FormatInt(s.Runtime.Time, 10)
			s.Step.Envs["DRONE_JOB_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)
			s.Step.Envs["DRONE_STAGE_STATUS"] = "success"
			s.Step.Envs["DRONE_STAGE_STARTED"] = strconv.FormatInt(s.Runtime.Time, 10)
			s.Step.Envs["DRONE_STAGE_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)

			if s.Runtime.Error != nil {
				s.Step.Envs["CI_BUILD_STATUS"] = "failure"
				s.Step.Envs["CI_JOB_STATUS"] = "failure"
				s.Step.Envs["DRONE_BUILD_STATUS"] = "failure"
				s.Step.Envs["DRONE_STAGE_STATUS"] = "failure"
				s.Step.Envs["DRONE_JOB_STATUS"] = "failure"
			}
			for _, stage := range m.Build.Stages {
				if stage.IsFailed() {
					s.Step.Envs["DRONE_BUILD_STATUS"] = "failure"
					break
				}
			}

			step, ok := steps[s.Step.Metadata.Name]
			if ok {
				step.Status = core.StatusRunning
				step.Started = time.Now().Unix()

				s.Step.Envs["DRONE_STEP_NAME"] = step.Name
				s.Step.Envs["DRONE_STEP_NUMBER"] = fmt.Sprint(step.Number)
			}

			stepClone := new(core.Step)
			*stepClone = *step
			r.Unlock()

			err := r.Manager.Before(ctx, stepClone)
			if err != nil {
				return err
			}

			r.Lock()
			step.ID = stepClone.ID
			step.Version = stepClone.Version
			r.Unlock()
			return nil
		},

		AfterEach: func(s *runtime.State) error {
			r.Lock()
			step, ok := steps[s.Step.Metadata.Name]
			if ok {
				step.Status = core.StatusPassing
				step.Stopped = time.Now().Unix()
				step.ExitCode = s.State.ExitCode
				if s.State.ExitCode != 0 && s.State.ExitCode != 78 {
					step.Status = core.StatusFailing
				}
			}
			stepClone := new(core.Step)
			*stepClone = *step
			r.Unlock()

			err := r.Manager.After(ctx, stepClone)
			if err != nil {
				return err
			}

			r.Lock()
			step.Version = stepClone.Version
			r.Unlock()

			return nil
		},

		GotLine: func(s *runtime.State, line *runtime.Line) error {
			r.Lock()
			step, ok := steps[s.Step.Metadata.Name]
			r.Unlock()
			if !ok {
				// TODO log error
				return nil
			}
			return r.Manager.Write(ctx, step.ID, convertLine(line))
		},

		GotLogs: func(s *runtime.State, lines []*runtime.Line) error {
			r.Lock()
			step, ok := steps[s.Step.Metadata.Name]
			r.Unlock()
			if !ok {
				// TODO log error
				return nil
			}
			raw, _ := json.Marshal(
				convertLines(lines),
			)
			return r.Manager.UploadBytes(ctx, step.ID, raw)
		},
	}

	runner := runtime.New(
		runtime.WithEngine(r.Engine),
		runtime.WithConfig(ir),
		runtime.WithHooks(hooks),
	)

	m.Stage.Status = core.StatusRunning
	m.Stage.Started = time.Now().Unix()
	m.Stage.Machine = r.Machine
	err = r.Manager.BeforeAll(ctx, m.Stage)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("runner: cannot initialize pipeline")
		return r.handleError(ctx, m.Stage, err)
	}

	timeout, cancel := context.WithTimeout(ctx, time.Duration(m.Repo.Timeout)*time.Minute)
	defer cancel()

	logger.Infoln("runner: start execution")

	err = runner.Run(timeout)
	if err != nil && err != runtime.ErrInterrupt {
		logger = logger.WithError(err)
		logger.Infoln("runner: execution failed")
		return r.handleError(ctx, m.Stage, err)
	}
	logger = logger.WithError(err)
	logger.Infoln("runner: execution complete")

	m.Stage.Status = core.StatusPassing
	m.Stage.Stopped = time.Now().Unix()
	for _, step := range m.Stage.Steps {
		if step.Status == core.StatusPending {
			step.Status = core.StatusSkipped
		}
		if step.Status == core.StatusRunning {
			step.Status = core.StatusPassing
			step.Stopped = time.Now().Unix()
		}
	}

	return r.Manager.AfterAll(ctx, m.Stage)
}

// Start starts N build runner processes. Each process polls
// the server for pending builds to execute.
func (r *Runner) Start(ctx context.Context, n int) error {
	var g errgroup.Group
	for i := 0; i < n; i++ {
		g.Go(func() error {
			return r.start(ctx)
		})
	}
	return g.Wait()
}

func (r *Runner) start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// This error is ignored on purpose. The system
			// should not exit the runner on error. The run
			// function logs all errors, which should be enough
			// to surface potential issues to an administrator.
			r.poll(ctx)
		}
	}
}

func (r *Runner) poll(ctx context.Context) error {
	logger := logrus.WithFields(
		logrus.Fields{
			"machine": r.Machine,
			"os":      r.OS,
			"arch":    r.Arch,
		},
	)

	logger.Debugln("runner: polling queue")
	p, err := r.Manager.Request(ctx, &manager.Request{
		Kind:    "pipeline",
		Type:    "docker",
		OS:      r.OS,
		Arch:    r.Arch,
		Kernel:  r.Kernel,
		Variant: r.Variant,
		Labels:  r.Labels,
	})
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("runner: cannot get queue item")
		return err
	}
	if p == nil || p.ID == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err = r.Manager.Accept(ctx, p.ID, r.Machine)
	if err == db.ErrOptimisticLock {
		return nil
	} else if err != nil {
		logger.WithError(err).
			WithFields(
				logrus.Fields{
					"stage-id": p.ID,
					"build-id": p.BuildID,
					"repo-id":  p.RepoID,
				}).Warnln("runner: cannot ack stage")
		return err
	}

	go func() {
		logger.Debugln("runner: watch for cancel signal")
		done, _ := r.Manager.Watch(ctx, p.BuildID)
		if done {
			cancel()
			logger.Debugln("runner: received cancel signal")
		} else {
			logger.Debugln("runner: done listening for cancel signals")
		}
	}()

	return r.Run(ctx, p.ID)
}

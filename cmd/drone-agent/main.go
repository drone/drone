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

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/drone/drone/version"

	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "drone-agent"
	app.Version = version.Version.String()
	app.Usage = "drone agent"
	app.Action = loop
	app.Commands = []cli.Command{
		{
			Name:   "ping",
			Usage:  "ping the agent",
			Action: pinger,
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			EnvVar: "DRONE_SERVER",
			Name:   "server",
			Usage:  "drone server address",
			Value:  "localhost:9000",
		},
		cli.StringFlag{
			EnvVar: "DRONE_USERNAME",
			Name:   "username",
			Usage:  "drone auth username",
			Value:  "x-oauth-basic",
		},
		cli.StringFlag{
			EnvVar: "DRONE_PASSWORD,DRONE_SECRET",
			Name:   "password",
			Usage:  "server-agent shared password",
		},
		cli.BoolTFlag{
			EnvVar: "DRONE_DEBUG",
			Name:   "debug",
			Usage:  "enable agent debug mode",
		},
		cli.BoolFlag{
			EnvVar: "DRONE_DEBUG_PRETTY",
			Name:   "pretty",
			Usage:  "enable pretty-printed debug output",
		},
		cli.BoolTFlag{
			EnvVar: "DRONE_DEBUG_NOCOLOR",
			Name:   "nocolor",
			Usage:  "disable colored debug output",
		},
		cli.StringFlag{
			EnvVar: "DRONE_HOSTNAME,HOSTNAME",
			Name:   "hostname",
			Usage:  "agent hostname",
		},
		cli.StringFlag{
			EnvVar: "DRONE_PLATFORM",
			Name:   "platform",
			Usage:  "restrict builds by platform conditions",
			Value:  "linux/amd64",
		},
		cli.StringFlag{
			EnvVar: "DRONE_FILTER",
			Name:   "filter",
			Usage:  "filter expression to restrict builds by label",
		},
		cli.IntFlag{
			EnvVar: "DRONE_MAX_PROCS",
			Name:   "max-procs",
			Usage:  "agent parallel builds",
			Value:  1,
		},
		cli.BoolTFlag{
			EnvVar: "DRONE_HEALTHCHECK",
			Name:   "healthcheck",
			Usage:  "enable healthcheck endpoint",
		},
		cli.DurationFlag{
			EnvVar: "DRONE_KEEPALIVE_TIME",
			Name:   "keepalive-time",
			Usage:  "after a duration of this time of no activity, the agent pings the server to check if the transport is still alive",
		},
		cli.DurationFlag{
			EnvVar: "DRONE_KEEPALIVE_TIMEOUT",
			Name:   "keepalive-timeout",
			Usage:  "after pinging for a keepalive check, the agent waits for a duration of this time before closing the connection if no activity",
			Value:  time.Second * 20,
		},
	}

	version.PrintVersion(true)

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

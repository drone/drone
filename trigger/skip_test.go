// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package trigger

import (
	"testing"

	"github.com/drone/drone-yaml/yaml"
	"github.com/drone/drone/core"
)

func Test_skipBranch(t *testing.T) {
	tests := []struct {
		config string
		branch string
		want   bool
	}{
		{
			config: "kind: pipeline\ntrigger: { }",
			branch: "master",
			want:   false,
		},
		{
			config: "kind: pipeline\ntrigger: { branch: [ master ] }",
			branch: "master",
			want:   false,
		},
		{
			config: "kind: pipeline\ntrigger: { branch: [ master ] }",
			branch: "develop",
			want:   true,
		},
	}
	for i, test := range tests {
		manifest, err := yaml.ParseString(test.config)
		if err != nil {
			t.Error(err)
		}
		pipeline := manifest.Resources[0].(*yaml.Pipeline)
		got, want := skipBranch(pipeline, test.branch), test.want
		if got != want {
			t.Errorf("Want test %d to return %v", i, want)
		}
	}
}

func Test_skipEvent(t *testing.T) {
	tests := []struct {
		config string
		event  string
		want   bool
	}{
		{
			config: "kind: pipeline\ntrigger: { }",
			event:  "push",
			want:   false,
		},
		{
			config: "kind: pipeline\ntrigger: { event: [ push ] }",
			event:  "push",
			want:   false,
		},
		{
			config: "kind: pipeline\ntrigger: { event: [ push ] }",
			event:  "pull_request",
			want:   true,
		},
	}
	for i, test := range tests {
		manifest, err := yaml.ParseString(test.config)
		if err != nil {
			t.Error(err)
		}
		pipeline := manifest.Resources[0].(*yaml.Pipeline)
		got, want := skipEvent(pipeline, test.event), test.want
		if got != want {
			t.Errorf("Want test %d to return %v", i, want)
		}
	}
}

// This verifies the skipEventAction handler behaviour.
func Test_skipEventAction(t *testing.T) {
	tests := []struct {
		config string
		event  string
		action string
		want   bool
	}{
		// Push should not be handled [sanity check]
		{
			config: "kind: pipeline\ntrigger: { }",
			event:  "push",
			action: "created",
			want:   false,
		},
		// Tag should not be handled [sanity check]
		{
			config: "kind: pipeline\ntrigger: { }",
			event:  "tag",
			action: "created",
			want:   false,
		},
		// Pull request open with opt in for close event [sanity check]
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ], action: [ closed, opened, synchronized ] }",
			event:  "pull_request",
			action: "opened",
			want:   false,
		},
		// Pull request synchronized with opt in for close event [sanity check]
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ], action: [ closed, opened, synchronized ] }",
			event:  "pull_request",
			action: "synchronized",
			want:   false,
		},
		// Pull request close with opt in for close event
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ], action: [ closed, opened, synchronized ] }",
			event:  "pull_request",
			action: "synchronized",
			want:   false,
		},
		// Pull request close without opt in for close event [normal behaviour]
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ], action: [ opened, synchronized ] }",
			event:  "pull_request",
			action: "closed",
			want:   true,
		},
		// Pull request close without explicit skip for close. Should still skip
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ], action: { exclude: [ opened, synchronized ] }}",
			event:  "pull_request",
			action: "closed",
			want:   true,
		},
		// Pull request opened with exclude for opened
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ], action: { exclude: [ opened, synchronized ] }}",
			event:  "pull_request",
			action: "opened",
			want:   true,
		},
		// Pull request with exclude only for opened
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ], action: { exclude: [ opened ] }}",
			event:  "pull_request",
			action: "synchronized",
			want:   false,
		},
		// Pull request close with opt in for close.
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ], action: { include: [ closed ] }}",
			event:  "pull_request",
			action: "closed",
			want:   false,
		},
		// Pull request open without specifying action
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ]}",
			event:  "pull_request",
			action: "opened",
			want:   false,
		},
		// Pull request synchronized specifying action
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ]}",
			event:  "pull_request",
			action: "synchronized",
			want:   false,
		},
		// Pull closed synchronized specifying action
		{
			config: "kind: pipeline\ntrigger: { event: [ pull_request ]}",
			event:  "pull_request",
			action: "closed",
			want:   true,
		},
	}

	for i, test := range tests {
		manifest, err := yaml.ParseString(test.config)
		if err != nil {
			t.Error(err)
		}

		pipeline := manifest.Resources[0].(*yaml.Pipeline)
		got, want := skipEventAction(pipeline, test.event, test.action), test.want

		if got != want {
			t.Errorf("Want test %d to return %v", i, want)
		}
	}
}

func Test_skipPullRequestEval(t *testing.T) {
	tests := []struct {
		condition *yaml.Condition
		action    string
		want      bool
	}{
		// Normal include condition
		{
			condition: &yaml.Condition{
				Include: []string{
					"closed",
					"opened",
					"synchronized",
				},
			},
			action: "opened",
			want:   false,
		},
		// Normal include condition with new property
		{
			condition: &yaml.Condition{
				Include: []string{
					"closed",
					"opened",
					"synchronized",
				},
			},
			action: "closed",
			want:   false,
		},
		// Normal skipped include condition
		{
			condition: &yaml.Condition{
				Include: []string{
					"closed",
				},
			},
			action: "opened",
			want:   true,
		},
		//  Default to ignoring the closed condition
		{
			condition: &yaml.Condition{
				Include: []string{},
				Exclude: []string{},
			},
			action: "closed",
			want:   true,
		},
		// When some excludes are supplied, still exclude closed.
		{
			condition: &yaml.Condition{
				Include: []string{},
				Exclude: []string{
					"synchronized",
				},
			},
			action: "closed",
			want:   true,
		},
	}

	for _, test := range tests {
		got := skipPullRequestEval(test.condition, test.action)
		if got != test.want {
			t.Errorf("Want { condition: %+v, action: %q } to return %v",
				test.condition, test.action, test.want)
		}
	}
}

// func Test_skipPath(t *testing.T) {
// 	tests := []struct {
// 		config string
// 		paths  []string
// 		want   bool
// 	}{
// 		{
// 			config: "trigger: { }",
// 			paths:  []string{},
// 			want:   false,
// 		},
// 		{
// 			config: "trigger: { }",
// 			paths:  []string{"README.md"},
// 			want:   false,
// 		},
// 		{
// 			config: "trigger: { paths: foo/* }",
// 			paths:  []string{"foo/README"},
// 			want:   false,
// 		},
// 		{
// 			config: "trigger: { paths: foo/* }",
// 			paths:  []string{"bar/README"},
// 			want:   true,
// 		},
// 		// if empty changeset, never skip the pipeline
// 		{
// 			config: "trigger: { paths: foo/* }",
// 			paths:  []string{},
// 			want:   false,
// 		},
// 		// if max changeset, never skip the pipeline
// 		{
// 			config: "trigger: { paths: foo/* }",
// 			paths:  make([]string, 400),
// 			want:   false,
// 		},
// 	}
// 	for i, test := range tests {
// 		document, err := config.ParseString(test.config)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		got, want := skipPaths(document, test.paths), test.want
// 		if got != want {
// 			t.Errorf("Want test %d to return %v", i, want)
// 		}
// 	}
// }

func Test_skipMessage(t *testing.T) {
	tests := []struct {
		event   string
		message string
		title   string
		want    bool
	}{
		{
			event:   "push",
			message: "update readme",
			want:    false,
		},
		// skip when message contains [CI SKIP]
		{
			event:   "push",
			message: "update readme [CI SKIP]",
			want:    true,
		},
		{
			event:   "pull_request",
			message: "update readme  [CI SKIP]",
			want:    true,
		},
		// skip when title contains [CI SKIP]

		{
			event: "push",
			title: "update readme [CI SKIP]",
			want:  true,
		},
		{
			event: "pull_request",
			title: "update readme  [CI SKIP]",
			want:  true,
		},
		// ignore [CI SKIP] when event is tag
		{
			event:   "tag",
			message: "update readme [CI SKIP]",
			want:    false,
		},
		{
			event: "tag",
			title: "update readme [CI SKIP]",
			want:  false,
		},
		{
			event: "cron",
			title: "update readme [CI SKIP]",
			want:  false,
		},
		{
			event: "cron",
			title: "update readme [CI SKIP]",
			want:  false,
		},
		{
			event: "custom",
			title: "update readme [CI SKIP]",
			want:  false,
		},
		{
			event: "custom",
			title: "update readme [CI SKIP]",
			want:  false,
		},
		{
			event: "promote",
			title: "update readme [CI SKIP]",
			want:  false,
		},
		{
			event: "promote",
			title: "update readme [CI SKIP]",
			want:  false,
		},
		{
			event: "rollback",
			title: "update readme [CI SKIP]",
			want:  false,
		},
		{
			event: "rollback",
			title: "update readme [CI SKIP]",
			want:  false,
		},
	}
	for _, test := range tests {
		hook := &core.Hook{
			Message: test.message,
			Title:   test.title,
			Event:   test.event,
		}
		got, want := skipMessage(hook), test.want
		if got != want {
			t.Errorf("Want { event: %q, message: %q, title: %q } to return %v",
				test.event, test.message, test.title, want)
		}
	}
}

func Test_skipMessageEval(t *testing.T) {
	tests := []struct {
		eval string
		want bool
	}{
		{"update readme", false},
		// test [CI SKIP]
		{"foo [ci skip] bar", true},
		{"foo [CI SKIP] bar", true},
		{"foo [CI Skip] bar", true},
		{"foo [CI SKIP]", true},
		// test [SKIP CI]
		{"foo [skip ci] bar", true},
		{"foo [SKIP CI] bar", true},
		{"foo [Skip CI] bar", true},
		{"foo [SKIP CI]", true},
		// test ***NO_CI***
		{"foo ***NO_CI*** bar", true},
		{"foo ***NO_CI*** bar", true},
		{"foo ***NO_CI*** bar", true},
		{"foo ***NO_CI***", true},
	}
	for _, test := range tests {
		got, want := skipMessageEval(test.eval), test.want
		if got != want {
			t.Errorf("Want %q to return %v, got %v", test.eval, want, got)
		}
	}
}

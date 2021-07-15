// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package converter

import (
	"github.com/drone/drone/mock/mockscm"
	"github.com/drone/go-scm/scm"
	"github.com/golang/mock/gomock"
	"testing"

	"github.com/drone/drone/core"
)

const jsonnetFile = `{"foo": "bar"}`
const jsonnetFileAfter = `---
{
   "foo": "bar"
}
`

const jsonnetStream = `[{"foo": "bar"}]`
const jsonnetStreamAfter = `---
{
   "foo": "bar"
}
`

const jsonnetFileImport = `local step = import '.step.libsonnet';
{"foo": ["bar"], "steps": [step]}`
const jsonnetFileImportLib = `{"image": "app"}`
const jsonnetFileImportAfter = `---
{
   "foo": [
      "bar"
   ],
   "steps": [
      {
         "image": "app"
      }
   ]
}
`

func TestJsonnet_Stream(t *testing.T) {
	args := &core.ConvertArgs{
		Repo:   &core.Repository{Config: ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetStream},
	}
	service := Jsonnet(true, nil)
	res, err := service.Convert(noContext, args)
	if err != nil {
		t.Error(err)
		return
	}
	if res == nil {
		t.Errorf("Expected a converted file, got nil")
		return
	}
	if got, want := res.Data, jsonnetStreamAfter; got != want {
		t.Errorf("Want converted file %q, got %q", want, got)
	}
}

func TestJsonnet_Snippet(t *testing.T) {
	args := &core.ConvertArgs{
		Repo:   &core.Repository{Config: ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetFile},
	}
	service := Jsonnet(true, nil)
	res, err := service.Convert(noContext, args)
	if err != nil {
		t.Error(err)
		return
	}
	if res == nil {
		t.Errorf("Expected a converted file, got nil")
		return
	}
	if got, want := res.Data, jsonnetFileAfter; got != want {
		t.Errorf("Want converted file %q, got %q", want, got)
	}
}

func TestJsonnet_Error(t *testing.T) {
	args := &core.ConvertArgs{
		Repo:   &core.Repository{Config: ".drone.jsonnet"},
		Config: &core.Config{Data: "\\"}, // invalid jsonnet
	}
	service := Jsonnet(true, nil)
	_, err := service.Convert(noContext, args)
	if err == nil {
		t.Errorf("Expect jsonnet parsing error, got nil")
	}
}

func TestJsonnet_Disabled(t *testing.T) {
	service := Jsonnet(false, nil)
	res, err := service.Convert(noContext, nil)
	if err != nil {
		t.Error(err)
	}
	if res != nil {
		t.Errorf("Expect nil response when disabled")
	}
}

func TestJsonnet_NotJsonnet(t *testing.T) {
	args := &core.ConvertArgs{
		Repo: &core.Repository{Config: ".drone.yml"},
	}
	service := Jsonnet(true, nil)
	res, err := service.Convert(noContext, args)
	if err != nil {
		t.Error(err)
	}
	if res != nil {
		t.Errorf("Expect nil response when not jsonnet")
	}
}

func TestJsonnet_Import(t *testing.T) {

	args := &core.ConvertArgs{
		Build: &core.Build{
			Ref: "a6586b3db244fb6b1198f2b25c213ded5b44f9fa",
		},
		Repo: &core.Repository{
			Namespace: "octocat",
			Name:      "hello-world",
			Config:    ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetFileImport},
		User: &core.User{
			Token: "foobar",
		},
	}
	importedContent := &scm.Content{
		Data: []byte(jsonnetFileImportLib),
	}
	cli := new(scm.Client)
	controller := gomock.NewController(t)
	mockContents := mockscm.NewMockContentService(controller)
	mockContents.EXPECT().Find(gomock.Any(), "octocat/hello-world", ".step.libsonnet", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa").Return(importedContent, nil, nil).Times(2)
	cli.Contents = mockContents
	service := Jsonnet(true, cli)
	res, err := service.Convert(noContext, args)
	if err != nil {
		t.Error(err)
	}
	if got, want := res.Data, jsonnetFileImportAfter; got != want {
		t.Errorf("Want converted file:\n%q\ngot\n%q", want, got)
	}
}

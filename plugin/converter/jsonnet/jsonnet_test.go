package jsonnet

import (
	"io/ioutil"
	"testing"

	"github.com/drone/drone/core"
)

func TestParse(t *testing.T) {
	before, err := ioutil.ReadFile("../testdata/input.jsonnet")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("../testdata/input.jsonnet.golden")
	if err != nil {
		t.Error(err)
		return
	}

	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.yml",
		},
		Config: &core.Config{},
	}

	template := &core.Template{
		Name: "my_template.jsonnet",
		Data: string(before),
	}

	templateData := map[string]interface{}{
		"stepName": "my_step",
		"image":    "my_image",
		"commands": "my_command",
	}

	req.Config.Data = string(before)

	parsedFile, err := Parse(req, nil, template, templateData)
	if err != nil {
		t.Error(err)
		return
	}

	if want, got := parsedFile, string(after); want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}

func TestParseJsonnetNotTemplateFile(t *testing.T) {
	before, err := ioutil.ReadFile("../testdata/single.jsonnet")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("../testdata/input.jsonnet.golden")
	if err != nil {
		t.Error(err)
		return
	}

	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.jsonnet",
		},
		Config: &core.Config{},
	}

	req.Repo.Config = "plugin.jsonnet"
	req.Config.Data = string(before)

	parsedFile, err := Parse(req, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}

	if want, got := parsedFile, string(after); want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}

package parser

import (
	"fmt"
	"strings"

	"github.com/drone/drone/common"
)

// lintRule defines a function that runs lint
// checks against a Yaml Config file. If the rule
// fails it should return an error message.
type lintRule func(*common.Config) error

var lintRules = [...]lintRule{
	expectBuild,
	expectImage,
	expectCommand,
	expectTrustedSetup,
	expectTrustedClone,
	expectTrustedPublish,
	expectTrustedDeploy,
	expectTrustedNotify,
}

// Lint runs all lint rules against the Yaml Config.
func Lint(c *common.Config) error {
	for _, rule := range lintRules {
		err := rule(c)
		if err != nil {
			return err
		}
	}
	return nil
}

// lint rule that fails when no build is defined
func expectBuild(c *common.Config) error {
	if c.Build == nil {
		return fmt.Errorf("Yaml must define a build section")
	}
	return nil
}

// lint rule that fails when no build image is defined
func expectImage(c *common.Config) error {
	if len(c.Build.Image) == 0 {
		return fmt.Errorf("Yaml must define a build image")
	}
	return nil
}

// lint rule that fails when no build commands are defined
func expectCommand(c *common.Config) error {
	if c.Build.Config == nil || c.Build.Config["commands"] == nil {
		return fmt.Errorf("Yaml must define build commands")
	}
	return nil
}

// lint rule that fails when a non-trusted clone plugin is used.
func expectTrustedClone(c *common.Config) error {
	if c.Clone != nil && strings.Contains(c.Clone.Image, "/") {
		return fmt.Errorf("Yaml must use trusted clone plugins")
	}
	return nil
}

// lint rule that fails when a non-trusted setup plugin is used.
func expectTrustedSetup(c *common.Config) error {
	if c.Setup != nil && strings.Contains(c.Setup.Image, "/") {
		return fmt.Errorf("Yaml must use trusted setup plugins")
	}
	return nil
}

// lint rule that fails when a non-trusted publish plugin is used.
func expectTrustedPublish(c *common.Config) error {
	for _, step := range c.Publish {
		if strings.Contains(step.Image, "/") {
			return fmt.Errorf("Yaml must use trusted publish plugins")
		}
	}
	return nil
}

// lint rule that fails when a non-trusted deploy plugin is used.
func expectTrustedDeploy(c *common.Config) error {
	for _, step := range c.Deploy {
		if strings.Contains(step.Image, "/") {
			return fmt.Errorf("Yaml must use trusted deploy plugins")
		}
	}
	return nil
}

// lint rule that fails when a non-trusted notify plugin is used.
func expectTrustedNotify(c *common.Config) error {
	for _, step := range c.Notify {
		if strings.Contains(step.Image, "/") {
			return fmt.Errorf("Yaml must use trusted notify plugins")
		}
	}
	return nil
}

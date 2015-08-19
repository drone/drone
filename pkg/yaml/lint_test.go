package parser

import (
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
	common "github.com/drone/drone/pkg/types"
)

func Test_Linter(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Linter", func() {

		g.It("Should fail when nil build", func() {
			c := &common.Config{}
			g.Assert(expectBuild(c) != nil).IsTrue()
		})

		g.It("Should fail when no image", func() {
			c := &common.Config{
				Build: &common.Step{},
			}
			g.Assert(expectImage(c) != nil).IsTrue()
		})

		g.It("Should fail when no commands", func() {
			c := &common.Config{
				Setup: &common.Step{},
			}
			g.Assert(expectCommand(c) != nil).IsTrue()
		})

		g.It("Should pass when proper Build provided", func() {
			c := &common.Config{
				Build: &common.Step{
					Config: map[string]interface{}{
						"commands": []string{"echo hi"},
					},
				},
			}
			g.Assert(expectImage(c) != nil).IsTrue()
		})

		g.It("Should pass linter when build properly setup", func() {
			c := &common.Config{}
			c.Build = &common.Step{}
			c.Build.Image = "golang"
			c.Setup = &common.Step{}
			c.Setup.Config = map[string]interface{}{}
			c.Setup.Config["commands"] = []string{"go build", "go test"}
			c.Clone = &common.Step{}
			c.Clone.Config = map[string]interface{}{}
			c.Clone.Config["path"] = "/drone/src/foo/bar"
			c.Publish = map[string]*common.Step{}
			c.Publish["docker"] = &common.Step{Image: "docker"}
			c.Deploy = map[string]*common.Step{}
			c.Deploy["kubernetes"] = &common.Step{Image: "kubernetes"}
			c.Notify = map[string]*common.Step{}
			c.Notify["email"] = &common.Step{Image: "email"}
			g.Assert(Lint(c) == nil).IsTrue()
		})

		g.It("Should pass with clone path inside workspace", func() {
			c := &common.Config{
				Clone: &common.Step{
					Config: map[string]interface{}{
						"path": "/drone/src/foo/bar",
					},
				},
			}
			g.Assert(expectCloneInWorkspace(c) == nil).IsTrue()
		})

		g.It("Should fail with clone path outside workspace", func() {
			c := &common.Config{
				Clone: &common.Step{
					Config: map[string]interface{}{
						"path": "/foo/bar",
					},
				},
			}
			g.Assert(expectCloneInWorkspace(c) != nil).IsTrue()
		})

		g.It("Should pass with cache path inside workspace", func() {
			c := &common.Config{
				Build: &common.Step{
					Cache: []string{".git", "/.git", "/.git/../.git/../.git"},
				},
			}
			g.Assert(expectCacheInWorkspace(c) == nil).IsTrue()
		})

		g.It("Should fail with cache path outside workspace", func() {
			c := &common.Config{
				Build: &common.Step{
					Cache: []string{".git", "/.git", "../../.git"},
				},
			}
			g.Assert(expectCacheInWorkspace(c) != nil).IsTrue()
		})

		g.It("Should fail when caching workspace directory", func() {
			c := &common.Config{
				Build: &common.Step{
					Cache: []string{".git", ".git/../"},
				},
			}
			g.Assert(expectCacheInWorkspace(c) != nil).IsTrue()
		})

		g.It("Should fail when : is in the cache path", func() {
			c := &common.Config{
				Build: &common.Step{
					Cache: []string{".git", ".git:/../"},
				},
			}
			g.Assert(expectCacheInWorkspace(c) != nil).IsTrue()
		})
	})
}

func Test_LintPlugins(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Plugin Linter", func() {

		g.It("Should fail un-trusted plugin", func() {
			c := &common.Config{
				Setup:   &common.Step{Image: "foo/baz"},
				Clone:   &common.Step{Image: "foo/bar"},
				Notify:  map[string]*common.Step{},
				Deploy:  map[string]*common.Step{},
				Publish: map[string]*common.Step{},
			}
			o := &Opts{Whitelist: []string{"plugins/*"}}
			g.Assert(LintPlugins(c, o) != nil).IsTrue()
		})

		g.It("Should pass when empty whitelist", func() {
			c := &common.Config{
				Setup:   &common.Step{Image: "foo/baz"},
				Clone:   &common.Step{Image: "foo/bar"},
				Notify:  map[string]*common.Step{},
				Deploy:  map[string]*common.Step{},
				Publish: map[string]*common.Step{},
			}
			o := &Opts{Whitelist: []string{}}
			g.Assert(LintPlugins(c, o) == nil).IsTrue()
		})

		g.It("Should pass wildcard", func() {
			c := &common.Config{
				Setup:   &common.Step{Image: "plugins/drone-setup"},
				Clone:   &common.Step{Image: "plugins/drone-build"},
				Notify:  map[string]*common.Step{},
				Deploy:  map[string]*common.Step{},
				Publish: map[string]*common.Step{},
			}
			o := &Opts{Whitelist: []string{"plugins/*"}}
			g.Assert(LintPlugins(c, o) == nil).IsTrue()
		})

		g.It("Should pass itemized", func() {
			c := &common.Config{
				Setup:   &common.Step{Image: "plugins/drone-setup"},
				Clone:   &common.Step{Image: "plugins/drone-build"},
				Notify:  map[string]*common.Step{},
				Deploy:  map[string]*common.Step{},
				Publish: map[string]*common.Step{},
			}
			o := &Opts{Whitelist: []string{"plugins/drone-setup", "plugins/drone-build"}}
			g.Assert(LintPlugins(c, o) == nil).IsTrue()
		})
	})
}

package model

import (
	"testing"

	"github.com/drone/drone/shared/database"
	"github.com/franela/goblin"
)

func TestBuild(t *testing.T) {
	db := database.Open("sqlite3", ":memory:")
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Builds", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM builds")
			db.Exec("DELETE FROM jobs")
		})

		g.It("Should Post a Build", func() {
			build := Build{
				RepoID: 1,
				Status: StatusSuccess,
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			err := CreateBuild(db, &build, []*Job{}...)
			g.Assert(err == nil).IsTrue()
			g.Assert(build.ID != 0).IsTrue()
			g.Assert(build.Number).Equal(1)
			g.Assert(build.Commit).Equal("85f8c029b902ed9400bc600bac301a0aadb144ac")
		})

		g.It("Should Put a Build", func() {
			build := Build{
				RepoID: 1,
				Number: 5,
				Status: StatusSuccess,
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			CreateBuild(db, &build, []*Job{}...)
			build.Status = StatusRunning
			err1 := UpdateBuild(db, &build)
			getbuild, err2 := GetBuild(db, build.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(build.ID).Equal(getbuild.ID)
			g.Assert(build.RepoID).Equal(getbuild.RepoID)
			g.Assert(build.Status).Equal(getbuild.Status)
			g.Assert(build.Number).Equal(getbuild.Number)
		})

		g.It("Should Get a Build", func() {
			build := Build{
				RepoID: 1,
				Status: StatusSuccess,
			}
			CreateBuild(db, &build, []*Job{}...)
			getbuild, err := GetBuild(db, build.ID)
			g.Assert(err == nil).IsTrue()
			g.Assert(build.ID).Equal(getbuild.ID)
			g.Assert(build.RepoID).Equal(getbuild.RepoID)
			g.Assert(build.Status).Equal(getbuild.Status)
		})

		g.It("Should Get a Build by Number", func() {
			build1 := &Build{
				RepoID: 1,
				Status: StatusPending,
			}
			build2 := &Build{
				RepoID: 1,
				Status: StatusPending,
			}
			err1 := CreateBuild(db, build1, []*Job{}...)
			err2 := CreateBuild(db, build2, []*Job{}...)
			getbuild, err3 := GetBuildNumber(db, &Repo{ID: 1}, build2.Number)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
		})

		g.It("Should Get a Build by Ref", func() {
			build1 := &Build{
				RepoID: 1,
				Status: StatusPending,
				Ref:    "refs/pull/5",
			}
			build2 := &Build{
				RepoID: 1,
				Status: StatusPending,
				Ref:    "refs/pull/6",
			}
			err1 := CreateBuild(db, build1, []*Job{}...)
			err2 := CreateBuild(db, build2, []*Job{}...)
			getbuild, err3 := GetBuildRef(db, &Repo{ID: 1}, "refs/pull/6")
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
			g.Assert(build2.Ref).Equal(getbuild.Ref)
		})

		g.It("Should Get a Build by Ref", func() {
			build1 := &Build{
				RepoID: 1,
				Status: StatusPending,
				Ref:    "refs/pull/5",
			}
			build2 := &Build{
				RepoID: 1,
				Status: StatusPending,
				Ref:    "refs/pull/6",
			}
			err1 := CreateBuild(db, build1, []*Job{}...)
			err2 := CreateBuild(db, build2, []*Job{}...)
			getbuild, err3 := GetBuildRef(db, &Repo{ID: 1}, "refs/pull/6")
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
			g.Assert(build2.Ref).Equal(getbuild.Ref)
		})

		g.It("Should Get a Build by Commit", func() {
			build1 := &Build{
				RepoID: 1,
				Status: StatusPending,
				Branch: "master",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			build2 := &Build{
				RepoID: 1,
				Status: StatusPending,
				Branch: "dev",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144aa",
			}
			err1 := CreateBuild(db, build1, []*Job{}...)
			err2 := CreateBuild(db, build2, []*Job{}...)
			getbuild, err3 := GetBuildCommit(db, &Repo{ID: 1}, build2.Commit, build2.Branch)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
			g.Assert(build2.Commit).Equal(getbuild.Commit)
			g.Assert(build2.Branch).Equal(getbuild.Branch)
		})

		g.It("Should Get a Build by Commit", func() {
			build1 := &Build{
				RepoID: 1,
				Status: StatusFailure,
				Branch: "master",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144ac",
			}
			build2 := &Build{
				RepoID: 1,
				Status: StatusSuccess,
				Branch: "master",
				Commit: "85f8c029b902ed9400bc600bac301a0aadb144aa",
			}
			err1 := CreateBuild(db, build1, []*Job{}...)
			err2 := CreateBuild(db, build2, []*Job{}...)
			getbuild, err3 := GetBuildLast(db, &Repo{ID: 1}, build2.Branch)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsTrue()
			g.Assert(build2.ID).Equal(getbuild.ID)
			g.Assert(build2.RepoID).Equal(getbuild.RepoID)
			g.Assert(build2.Number).Equal(getbuild.Number)
			g.Assert(build2.Status).Equal(getbuild.Status)
			g.Assert(build2.Branch).Equal(getbuild.Branch)
			g.Assert(build2.Commit).Equal(getbuild.Commit)
		})

		g.It("Should get recent Builds", func() {
			build1 := &Build{
				RepoID: 1,
				Status: StatusFailure,
			}
			build2 := &Build{
				RepoID: 1,
				Status: StatusSuccess,
			}
			CreateBuild(db, build1, []*Job{}...)
			CreateBuild(db, build2, []*Job{}...)
			builds, err := GetBuildList(db, &Repo{ID: 1})
			g.Assert(err == nil).IsTrue()
			g.Assert(len(builds)).Equal(2)
			g.Assert(builds[0].ID).Equal(build2.ID)
			g.Assert(builds[0].RepoID).Equal(build2.RepoID)
			g.Assert(builds[0].Status).Equal(build2.Status)
		})
	})
}

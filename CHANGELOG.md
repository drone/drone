# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased
### Added
- Allow jsonnet imports in pipeline configuration

## [2.0.4]
### Fixed
- DRON-97 remove use of request animation frame to prevent high CPU on tab refocus events.

## [2.0.3]
### Fixed
- DONE-91 handle extra slashes in url. [#3009](https://github.com/drone/drone/pull/3099).

## [2.0.2]
### Added
- Merge remote-tracking branch 'origin/master'
- prevent repository list short circuit in UI
- remove deprecated steps from building file [#3097](https://github.com/drone/drone/pull/3097)
- adding depends_on, image and detached fields to step [#3072](https://github.com/drone/drone/pull/3072)
- Add ctx.build.debug boolean [#3082](https://github.com/drone/drone/pull/3082)
- Bump github.com/google/go-jsonnet to v0.17.0 [#3084](https://github.com/drone/drone/pull/3084)
- bump go-scm v1.15.1 [#3096](https://github.com/drone/drone/pull/3096)
- bitbucket server build issue [#3092](https://github.com/drone/drone/pull/3092)
- update scm version [#3091](https://github.com/drone/drone/pull/3091)  
- Limit graceful shutdown duration [#3093](https://github.com/drone/drone/pull/3093)  
- bump user interface  
- bump ui version
- ignore skip directive for promote and rollback events
- new feature: maximum open DB connections is configurable[#3089](https://github.com/drone/drone/pull/3089) 
- jsonnet additional parameters [#3087](https://github.com/drone/drone/pull/3087)
- hide login button if user already authenticated  
- new feature: configuration templates [#3081](https://github.com/drone/drone/pull/3081)

### Fixed
- various typos [#3088](https://github.com/drone/drone/pull/3088)
- handle error properly if template doesn't exist [#3095](https://github.com/drone/drone/pull/3093)
- oss build issue [#3086](https://github.com/drone/drone/pull/3086)
- graceful shutdown [#3083](https://github.com/drone/drone/pull/3083)

## [2.0.1]
### Added
- support for configuring the internal yaml cache size.

## [2.0.0]
### Added
- feature flags for mixed-mode database encryption.

### Changed
- user-interface re-design

### Breaking
- removed deprecated kubernetes integration in favor of official kubernetes runner.
- removed deprecated nomad integration in favor of official nomad runner.

## [1.10.1]
### Added
- support for repository-level concurrency limits.
- support for gitlab and github internal visibility on initial sync.

### Fixed
- create machine user with a custom API token.

## [1.10.0]
### Added
- support for starlark scripts in core.
- support for executing pipelines in debug mode.

## [1.9.2]
### Added
- update go-scm dependency to fix

## [1.9.1]
### Added
- support for increasing the http request timeout for extensions. [#2998](https://github.com/drone/drone/pull/2998).
- support for skipping a pipeline if the validation extension returns an ErrSkip.
- support for blocking a pipeline if the validation extension returns an ErrBlock.

### Fixed
- rollback endpoint should be available to users with write permission.
- retrying a build should re-use custom build parameters from parent build.

## [1.9.0] - 2020-07-12
### Added
- ui support for deployment list and summary.
- ui support for promoting and rolling back builds.
- feature flag to use static secret when signing webhooks, from @chiraggadasc.

### Fixed
- ui branch list improperly capped.

### Changed
- upgrade drone/envsubst dependency
- upgrade drone/go-scm dependency

## [1.8.1] - 2020-06-23
### Fixed
- support for gitea api pagination, repository sync hanging.

## [1.8.0] - 2020-06-10
### Added
- re-assigned repository ownership when deactivating a user.
- re-assigned repository ownership when deleting a user.
- de-activate a repository when deleting a user if re-assignment fails.
- de-activate a repository when deactivating a user if re-assignment fails.
- routine to cleanup builds stuck in a pending state.
- routine to cleanup builds stuck in a running state.
- private mode setting requires authentication to view public repositories.

### Fixed
- canceling a build emits a sql.ErrNoRows error.
- custom token is ignored when creating a user account via the API.
- machine accounts with sufficient permissions can create builds via the API.

### Changed
- upgraded Go toolchain to version 1.14.4.

## [1.7.0] - 2020-03-27
### Added
- endpoint to display the latest build by branch. [#2940](https://github.com/drone/drone/pull/2940).
- endpoint to display the latest build by pull request. [#2940](https://github.com/drone/drone/pull/2940).
- endpoint to display the latest build by environment. [#2940](https://github.com/drone/drone/pull/2940).
- endpoint to delete a branch from the index. [#2940](https://github.com/drone/drone/pull/2940).
- endpoint to delete a pull request from the index. [#2940](https://github.com/drone/drone/pull/2940).
- endpoint to delete an environment from the index. [#2940](https://github.com/drone/drone/pull/2940).
- page to view the latest build per branch.

### Fixed
- sync routine not executing asynchronously, being cancelled by http context.
- sync routine should ignore gitlab subrepositories
- convert deploy events in 0.8 yaml to promote events.
- do not execute cron job for disabled repositories. [#2931](https://github.com/drone/drone/issues/2931).
- remove trailing slash from gitea url to prevent oauth2 token refresh errors, by [@cmj0121](https://github.com/cmj0121). [#2920](https://github.com/drone/drone/issues/2920). 
- disable font ligatures in build log output. [drone/drone-ui#322](https://github.com/drone/drone-ui/pull/322).
- missing am/pm in timestamps

## [1.6.5] - 2020-01-29
### Changed
- update version of go-scm
- update alpine version in docker images
- use ticker for cron jobs for more accurate timing

## [1.6.4] - 2019-12-30
### Added
- optionally enable pprof endpoints for profiling, by [@bradrydzewski](https://github.com/bradrydzewski).

## [1.6.3] - 2019-12-10
### Fixed
- disable caching generated yaml files by commit sha, by [@bradrydzewski](https://github.com/bradrydzewski).

### Added
- support for bitbucket skipverify, by [@toni-moreno](https://github.com/toni-moreno).
- support for gitea skipverify, by [@toni-moreno](https://github.com/toni-moreno).

## [1.6.2] - 2019-11-08
### Added
- support for loading license contents from env, by [@bradrydzewski](https://github.com/bradrydzewski).

### Fixed
- regression not converting legacy pipeline when using new runners, by [@bradrydzewski](https://github.com/bradrydzewski).

## [1.6.1] - 2019-10-17
### Added
- updated autocert library in support of acme v2 protocol, by [@bradrydzewski](https://github.com/bradrydzewski).

### Fixed
- fixed nil pointer when manually adding user from api, by [@bradrydzewski](https://github.com/bradrydzewski).

## [1.6.0] - 2019-10-04
### Added
- added nsswitch to docker images
- option to auto-cancel pending builds when newer build enqueued, by [@bradrydzewski](https://github.com/bradrydzewski). [#1980](https://github.com/drone/drone/issues/1980).
- endpoint to list all repositories in the database, by [@bradrydzewski](https://github.com/bradrydzewski). [#2785](https://github.com/drone/drone/issues/2785).

### Fixed
- improve sync to handle duplicate repository names with different unique identifiers, by [@bradrydzewski](https://github.com/bradrydzewski). [#2658](https://github.com/drone/drone/issues/2658). _You can revert to the previous sync logic with DRONE_DATABASE_LEGACY_BATCH=true_.

## [1.5.1] - 2019-09-30
### Added
- allow organization admins access to organization secret endpoints, by [@bradrydzewski](https://github.com/bradrydzewski). [#2838](https://github.com/drone/drone/issues/2838).

### Fixed
- fix invalid deep links in UI for github enterprise, by [@bradrydzewski](https://github.com/bradrydzewski).
- ensure correct casing when manually adding user, by [@bradrydzewski](https://github.com/bradrydzewski). [#2766](https://github.com/drone/drone/issues/2766).

## [1.5.0] - 2019-09-28
### Added
- endpoint to execute a cron pipeline on-demand, by [@bradrydzewski](https://github.com/bradrydzewski). [#2781](https://github.com/drone/drone/issues/2781).
- endpoint to list builds by branch, by [@bradrydzewski](https://github.com/bradrydzewski). [#1495](https://github.com/drone/drone/issues/1495).
- ignore skip comments when cron event, by [@bradrydzewski](https://github.com/bradrydzewski). [#2835](https://github.com/drone/drone/issues/2835).
- support for admission extensions, by [@bradrydzewski](https://github.com/bradrydzewski). [#2043](https://github.com/drone/drone/issues/2043).
- endpoint to provide link to git resources, by [@bradrydzewski](https://github.com/bradrydzewski). [#2843](https://github.com/drone/drone/issues/2843).
- improve bitbucket status display text on new pull request screen, by [@bradrydzewski](https://github.com/bradrydzewski).

### Fixed
- missing cron job name in user interface, by [@bradrydzewski](https://github.com/bradrydzewski).
- log lines not properly wrapping in user interface, by [@bradrydzewski](https://github.com/bradrydzewski).
[#309](https://github.com/drone/drone-ui/issues/309).

### Breaking
- the server now runs in multi-machine mode by default. In order to run the server in single-machine mode (agents disabled) you must set DRONE_AGENTS_DISABLED=true.

## [1.4.0] - 2019-09-12
### Added
- upgrade to Go 1.13 to resolve arm segfault, by [@KN4CK3R](https://github.com/KN4CK3R). [#2823](https://github.com/drone/drone/issues/2823).
- configure default visibility, by [@JordanSussman](https://github.com/JordanSussman). [#2824](https://github.com/drone/drone/issues/2824).
- configure default trusted flag, by [@vyckou](https://github.com/vyckou).
- support for validation plugins, by [@bradrydzewski](https://github.com/bradrydzewski). [#2266](https://github.com/drone/drone/issues/2266).
- support for conversion plugins, by [@bradrydzewski](https://github.com/bradrydzewski).
- support for cron event type, by [@bradrydzewski](https://github.com/bradrydzewski). [#2705](https://github.com/drone/drone/issues/2705).
- support for rollback event, by [@bradrydzewski](https://github.com/bradrydzewski). [#2695](https://github.com/drone/drone/issues/2695).
- support for lets encrypt email, by [@bradrydzewski](https://github.com/bradrydzewski). [#2505](https://github.com/drone/drone/issues/2505).

### Removed
- Support for basic auth as an option for Gitea, by [@techknowlogick](https://giteahub.com/techknowlogick). [#2721](https://github.com/drone/drone/issues/2721)

### Fixed
- copy cron job name when restarting a cron job, by [@bradrydzewski](https://github.com/bradrydzewski). [#2760](https://github.com/drone/drone/issues/2760).

## [1.3.1] - 2019-08-26
### Added
- support for the GitHub deployment status API, by [@bradrydzewski](https://github.com/bradrydzewski).

## [1.3.0] - 2019-08-20
### Added
- support for storing logs in Azure Cloud Storage, by [@Lucretius](https://github.com/Lucretius). [#2788](https://github.com/drone/drone/pull/2788)
- support for windows server 1903, by [@bradrydzewski](https://github.com/bradrydzewski).
- button to view the full log file, by [@dramich](https://github.com/dramich). [drone/drone-ui#287](https://github.com/drone/drone-ui/pull/287).

### Fixed
- read gogs sha from webhook, by [@marcotuna](https://github.com/marcotuna).
- create bind volume on host if not exists, by [@bradrydzewski](https://github.com/bradrydzewski). [#2725](https://github.com/drone/drone/issues/2725).
- preserve whitespace in build logs, by [@geek1011](https://github.com/geek1011). [drone/drone-ui#294](https://github.com/drone/drone-ui/pull/294).
- enable log file download on firefox, by [@bobmanary](https://github.com/bobmanary). [drone/drone-ui#303](https://github.com/drone/drone-ui/pull/303)

### Security
- upgraded to Go 1.12.9 due to CVE-2019-9512 and CVE-2019-9514

## [1.2.3] - 2019-07-30
### Added

- disable github status for cron jobs
- support for action in conditionals, by [@bradrydzewski](https://github.com/bradrydzewski). [#2685](https://github.com/drone/drone/issues/2685).

### Fixed

- improve cancel logic for dangling stages, by [@bradrydzewski](https://github.com/bradrydzewski).
- improve error when kubernetes malforms the port configuration, by [@bradrydzewski](https://github.com/bradrydzewski). [#2742](https://github.com/drone/drone/issues/2742).
- copy parameters from parent build when promoting, by [@bradrydzewski](https://github.com/bradrydzewski). [#2748](https://github.com/drone/drone/issues/2748).

## [1.2.2] - 2019-07-29
### Added

- support for legacy environment variables
- support for legacy workspace based on repository name
- support for github deployment hooks
- provide base sha for github pull requests
- option to filter webhooks by event and type
- upgrade drone-yaml to v1.2.2
- upgrade drone-runtime to v1.0.7

### Fixed

- error when manually creating an empty user, by [@bradrydzewski](https://github.com/bradrydzewski). [#2738](https://github.com/drone/drone/issues/2738).

## [1.2.1] - 2019-06-11
### Added

- support for legacy tokens to ease upgrade path, by [@bradrydzewski](https://github.com/bradrydzewski). [#2713](https://github.com/drone/drone/issues/2713).
- include repository name and id in batch update error message, by [@bradrydzewski](https://github.com/bradrydzewski).

### Fixed

- fix inconsistent base64 encoding and decoding of encrypted secrets, by [@bradrydzewski](https://github.com/bradrydzewski).
- update drone-yaml to version 1.1.2 for improved 0.8 to 1.0 yaml marshal escaping.
- update drone-yaml to version 1.1.3 for improved 0.8 to 1.0 workspace conversion.

## [1.2.0] - 2019-05-30
### Added

- endpoint to trigger new build for default branch, by [@bradrydzewski](https://github.com/bradrydzewski). [#2679](https://github.com/drone/drone/issues/2679).
- endpoint to trigger new build for branch, by [@bradrydzewski](https://github.com/bradrydzewski). [#2679](https://github.com/drone/drone/issues/2679).
- endpoint to trigger new build for branch and sha, by [@bradrydzewski](https://github.com/bradrydzewski). [#2679](https://github.com/drone/drone/issues/2679).
- enable optional prometheus metrics guest access, by [@janberktold](https://github.com/janberktold)
- fallback to database when logs not found in s3, by [@bradrydzewski](https://github.com/bradrydzewski). [#2689](https://github.com/drone/drone/issues/2689).
- support for custom stage definitions and runners, by [@bradrydzewski](https://github.com/bradrydzewski). [#2680](https://github.com/drone/drone/issues/2680).
- update drone-yaml to version 1.1.0

### Fixed

- retrieve latest build by branch, by [@tboerger](https://github.com/tboerger).
- copy the fork value when restarting a build, by [@bradrydzewski](https://github.com/bradrydzewski). [#2708](https://github.com/drone/drone/issues/2708).
- make healthz available without redirect, by [@bradrydzewski](https://github.com/bradrydzewski). [#2706](https://github.com/drone/drone/issues/2706).

## [1.1.0] - 2019-04-23
### Added

- specify a user for the pipeline step, by [@bradrydzewski](https://github.com/bradrydzewski). [#2651](https://github.com/drone/drone/issues/2651).
- support for Gitea oauth2, by [@techknowlogick](https://github.com/techknowlogick). [#2622](https://github.com/drone/drone/pull/2622).
- ping the docker daemon before starting the agent, by [@bradrydzewski](https://github.com/bradrydzewski). [#2495](https://github.com/drone/drone/issues/2495).
- support for Cron job name in Yaml trigger block, by [@bradrydzewski](https://github.com/bradrydzewski). [#2628](https://github.com/drone/drone/issues/2628).
- support for Cron job name in Yaml when block, by [@bradrydzewski](https://github.com/bradrydzewski). [#2628](https://github.com/drone/drone/issues/2628).
- sqlite username column changed to case-insensitive, by [@bradrydzewski](https://github.com/bradrydzewski).
- endpoint to purge repository from database, by [@bradrydzewski](https://github.com/bradrydzewski).
- support for per-organization secrets, by [@bradrydzewski](https://github.com/bradrydzewski).
- include system metadata in global webhooks, by [@bradrydzewski](https://github.com/bradrydzewski).
- ability to customize cookie secure flag, by [@bradrydzewski](https://github.com/bradrydzewski).
- update drone-yaml from version 1.0.6 to 1.0.8.
- update drone-runtime from version 1.0.4 to 1.0.6.
- update go-scm from version 1.0.3 to 1.0.4.

### Fixed

- fixed error in mysql table creation syntax, from [@xuyang2](https://github.com/xuyang2). [#2677](https://github.com/drone/drone/pull/2677).
- fixed stuck builds when upstream dependency is skipped, from [@bradrydzewski](https://github.com/bradrydzewski). [#2634](https://github.com/drone/drone/issues/2634).
- fixed issue running steps with dependencies on failure, from [@bradrydzewski](https://github.com/bradrydzewski). [#2667](https://github.com/drone/drone/issues/2667).

## [1.0.1] - 2019-04-10
### Added

- pass stage environment variables to pipeline steps, by [@bradrydzewski](https://github.com/bradrydzewski).
- update go-scm to version 1.3.0, by [@bradrydzewski](https://github.com/bradrydzewski).
- update drone-runtime to version to 1.0.4, by [@bradrydzewski](https://github.com/bradrydzewski).
- ping docker daemon before agent starts to ensure connectivity, by [@bradrydzewski](https://github.com/bradrydzewski).

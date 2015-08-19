> **WARNING** the 0.4 branch is very unstable. We only recommend running this branch if you plan to dig into the codebase, troubleshoot issues and contribute (please do!). We will notify the broader community once this branch is more stable.

Drone is a Continuous Integration platform built on container technology. Every build is executed inside an ephemeral Docker container, giving developers complete control over their build environment with guaranteed isolation.

### Goals

Drone's prime directive is to help teams [ship code like GitHub](https://github.com/blog/1241-deploying-at-github#always-be-shipping). Drone is easy to install, setup and maintain and offers a powerful container-based plugin system. Drone aspires to be an industry-wide replacement for Jenkins.

### Documentation

Drone documentation is organized into several categories:

* [Setup Guide](http://readme.drone.io/docs/setup/)
* [Build Guide](http://readme.drone.io/docs/build/)
* [Plugin Guide](http://readme.drone.io/docs/plugin/)
* [API Reference](http://readme.drone.io/docs/api/)

### Community, Help

Contributions, questions, and comments are welcomed and encouraged. Drone developers hang out in the [drone/drone](https://gitter.im/drone/drone) room on gitter. We ask that you please post your questions to [gitter](https://gitter.im/drone/drone) before creating an issue.

### Cloning, Building, Running

If you are new to Go, make sure you [install](http://golang.org/doc/install) Go 1.4+ and [setup](http://golang.org/doc/code.html) your workspace (ie `$GOPATH`). Go programs use directory structure for package imports, therefore, it is very important you clone this project to the specified directory in your Go path:

```
git clone git://github.com/drone/drone.git $GOPATH/src/github.com/drone/drone
cd $GOPATH/src/github.com/drone/drone
```

Commands to build from source:

```sh
make deps    # download dependencies
make         # create binary files in ./bin
make test    # execute unit tests
```

Commands to start drone:

```sh
bin/drone
bin/drone --debug # debug mode loads static content from filesystem
```

If you are seeing slow compile times please install the following:

```sh
go install github.com/drone/drone/Godeps/_workspace/src/github.com/mattn/go-sqlite3
```

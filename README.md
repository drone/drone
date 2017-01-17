[![Build Status](http://beta.drone.io/api/badges/drone/drone/status.svg)](http://beta.drone.io/drone/drone)
![Release Status](https://img.shields.io/badge/status-beta-yellow.svg?style=flat)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/drone/drone?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Drone is a Continuous Integration platform built on container technology. Every build is executed inside an ephemeral Docker container, giving developers complete control over their build environment with guaranteed isolation.

### Goals

* Drone's prime directive is to help teams [ship code like GitHub](https://github.com/blog/1241-deploying-at-github#always-be-shipping).
* Drone is easy to install, setup and maintain and offers a powerful container-based plugin system.
* Drone aspires to be an industry-wide replacement for Jenkins.

### Documentation

Documentation is published to [readme.drone.io](http://readme.drone.io)

### Community, Help

Contributions, questions, and comments are welcomed and encouraged. Drone developers hang out in the [drone/drone](https://gitter.im/drone/drone) room on gitter. We ask that you please post your questions to [gitter](https://gitter.im/drone/drone) before creating an issue.

### Installation

Please see our [installation guide](http://readme.drone.io/admin/) to install the official Docker image.

### Building from Source

Clone the repository to your Go workspace:

```
git clone git://github.com/drone/drone.git $GOPATH/src/github.com/drone/drone
cd $GOPATH/src/github.com/drone/drone
```

Commands to build from source:

```sh
export GO15VENDOREXPERIMENT=1

make deps    # Download required dependencies
make gen     # Generate code
make build   # Build the binary
```

If you are having trouble building this project, please refer its `.drone.yml` file. Everything you need to know about building this project is defined in that file.

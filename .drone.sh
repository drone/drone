#!/bin/sh

# only execute this script as part of the pipeline.
[ -z "$CI" ] && echo "missing ci environment variable" && exit 2

# only execute the script when github token exists.
[ -z "$SSH_KEY" ] && echo "missing ssh key" && exit 3

BUILD_DATE=$(date --iso-8601)

# write the ssh key.
mkdir /root/.ssh
echo -n "$SSH_KEY" > /root/.ssh/id_rsa
chmod 600 /root/.ssh/id_rsa

# add github.com to our known hosts.
touch /root/.ssh/known_hosts
chmod 600 /root/.ssh/known_hosts
ssh-keyscan -H github.com > /etc/ssh/ssh_known_hosts 2> /dev/null

# clone the extras project.
set -e
set -x
git clone git@github.com:drone/drone-enterprise.git extras

# build a static binary with the build number and extra features.
go build -ldflags '-extldflags "-static" -X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER} -X github.com/drone/drone/version.BuildDate=${BUILD_DATE}' -o release/drone-server github.com/drone/drone/extras/cmd/drone-server
GOOS=linux GOARCH=amd64 CGO_ENABLED=0         go build -ldflags '-X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER} -X github.com/drone/drone/version.BuildDate=${BUILD_DATE}' -o release/drone-agent             github.com/drone/drone/cmd/drone-agent
GOOS=linux GOARCH=arm64 CGO_ENABLED=0         go build -ldflags '-X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER} -X github.com/drone/drone/version.BuildDate=${BUILD_DATE}' -o release/linux/arm64/drone-agent github.com/drone/drone/cmd/drone-agent
GOOS=linux GOARCH=arm   CGO_ENABLED=0 GOARM=7 go build -ldflags '-X github.com/drone/drone/version.VersionDev=build.${DRONE_BUILD_NUMBER} -X github.com/drone/drone/version.BuildDate=${BUILD_DATE}' -o release/linux/arm/drone-agent   github.com/drone/drone/cmd/drone-agent

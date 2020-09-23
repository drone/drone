# defines the target version of Go. Please do not
# update this variable unless it has been previously
# approved on the mailing list.
local golang = "golang:1.11";

# defines a temporary volume so that the Go cache can
# be shared with all pipeline steps.
local volumes = [
    {
        name: "gopath",
        temp: {},
    },
];

# defines the default Go cache location as a volume
# that is mounted into pipeline steps.
local mounts = [
    {
        name: "gopath",
        path: "/go",
    },
];

# defines a pipeline step that builds and publishes
# a docker image to a docker remote registry.
local docker(name, image, os, arch) = {
    name: "publish_" + name,
    image: "plugins/docker",
    settings: {
        repo: "drone/" + if name == "server" then "drone" else name,
        auto_tag: true,
        auto_tag_suffix: os + "-" + arch,
        username: { from_secret: "docker_username" },
        password: { from_secret: "docker_password" },
        dockerfile: "docker/Dockerfile." + name + "." + os + "." + arch,
    },
    when: {
        event: [ "push", "tag" ],
    },
};

# defines a pipeline step that creates and publishes
# a docker manifest to a docker remote registry.
local manifest(name) = {
    name: name,
    image: "plugins/manifest",
    settings: {
        auto_tag: true,
        ignore_missing: true,
        spec: "docker/manifest." + name + ".tmpl",
        username: { from_secret: "docker_username" },
        password: { from_secret: "docker_password" },
    },
    when: {
        event: [ "push", "tag" ],
    },
};

# defines a pipeline that builds, tests and publishes
# docker images for the Drone agent, server and controller.
local pipeline(name, os, arch) = {
    kind: "pipeline",
    name: name,
    volumes: volumes,
    platform: {
        os: os,
        arch: arch,
    },
    steps: [
        {
            name: "test",
            image: golang,
            volumes: mounts,
            commands: [ "go test -v ./..." ],
        },
        {
            name: "build",
            image: golang,
            volumes: mounts,
            commands: [
                "go build -ldflags \"-extldflags \\\\\"-static\\\\\"\" -o release/"+ os +"/" + arch + "/drone-server github.com/drone/drone/cmd/drone-server",
                "CGO_ENABLED=0 go build -o release/"+ os +"/" + arch + "/drone-agent github.com/drone/drone/cmd/drone-agent",
                "CGO_ENABLED=0 go build -o release/"+ os +"/" + arch + "/drone-controller github.com/drone/drone/cmd/drone-controller",
            ],
            when: {
                event: [ "push", "tag" ],
            },
        },
        docker("agent", "agent", os, arch),
        docker("controller", "controller", os, arch),
        docker("server", "drone", os, arch),
    ],
};

[
    pipeline("linux-amd64", "linux", "amd64"),
    pipeline("linux-arm", "linux", "arm"),
    pipeline("linux-arm64", "linux", "arm64"),
    pipeline("linux-ppc64le", "linux", "ppc64le"),
    {
        kind: "pipeline",
        name: "manifest",
        steps: [
            manifest("server"),
            manifest("controller"),
            manifest("agent"),
        ],
        depends_on: [
          "linux-amd64",
          "linux-arm",
          "linux-arm64",
          "linux-ppc64le",
        ],
    },
]

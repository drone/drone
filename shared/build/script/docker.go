package script

const (
	DefaultDockerNetworkMode = "bridge"
)

// Docker stores the configuration details for
// configuring docker container.
type Docker struct {
	// NetworkMode (also known as `--net` option)
	// Could be set only if Docker is running in privileged mode
	NetworkMode *string `yaml:"net,omitempty"`

	// Hostname (also known as `--hostname` option)
	// Could be set only if Docker is running in privileged mode
	Hostname *string `yaml:"hostname,omitempty"`

	// Allocate a pseudo-TTY (also known as `--tty` option)
	TTY bool `yaml:"tty,omitempty"`

	// Privileged means enabling all devices on the host in container
	// (also known as `--privileged=true` option)
	Privileged bool `yaml:"privileged,omitempty"`
}

// DockerNetworkMode returns DefaultNetworkMode
// when Docker.NetworkMode is empty.
// DockerNetworkMode returns Docker.NetworkMode
// when it is not empty.
func DockerNetworkMode(d *Docker) string {
	if d == nil || d.NetworkMode == nil {
		return DefaultDockerNetworkMode
	}
	return *d.NetworkMode
}

// DockerNetworkMode returns empty string
// when Docker.NetworkMode is empty.
// DockerNetworkMode returns Docker.NetworkMode
// when it is not empty.
func DockerHostname(d *Docker) string {
	if d == nil || d.Hostname == nil {
		return ""
	}
	return *d.Hostname
}

// DockerTty returns true if the build
// should allocate a pseudo tty
func DockerTty(d *Docker) bool {
	if d == nil {
		return false
	}
	return d.TTY
}

// DockerPrivileged returns true if the build
// should have privileged mode
func DockerPrivileged(d *Docker) bool {
	if d == nil {
		return false
	}
	return d.Privileged
}

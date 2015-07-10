# Installation

To quickly tryout Drone we have a [Docker image](https://registry.hub.docker.com/u/drone/drone/) that includes everything you need to get started. Simply run the commend below:

```
sudo docker run \
	--volume /var/lib/drone:/var/lib/drone \
	--volume /var/run/docker.sock:/var/run/docker.sock \
	--env-file /etc/defaults/drone \
	--restart=always \
	--publish=80:8000 \
	--detach=true \
	--name=drone \
	drone/drone
```

Drone is now running (in the background) on `http://localhost:80`. Note that before running you should create the `--env-file` and add your Drone configuration (GitHub, Bitbucket, GitLab credentials, etc).

## Docker options

Here are some of the Docker options, explained:

* `--restart=always` starts Drone automatically during system init process
* `--publish=80:8000` runs Drone on port `80`
* `--detach=true` starts Drone in the background
* `--volume /var/lib/drone:/var/lib/drone` mounted volume to persist sqlite database
* `--volume /var/run/docker.sock:/var/run/docker.sock` mounted volume to access Docker and spawn builds
* `--env-file /etc/defaults/drone` loads an external file with environment variables. Used to configure Drone.

## Drone settings

Drone uses environment variables for runtime settings and configuration, such as GitHub, GitLab, plugins and more. These settings can be provided to Docker using an `--env-file` as seen above.

## Starting, Stopping, Logs

Commands to start, stop and restart Drone:

```
docker start drone
docker stop drone
docker restart drone
```

And to view the Drone logs:

```
docker logs drone
```

## Upstart

Drone can be configured to work with process managers, such as **Ubuntu** Upstart, to automatically start when the operating system initializes. Here is an example upstart script that can be placed in `/etc/init/drone.conf`:

```
description "Drone container"

start on filesystem and started docker
stop on runlevel [!2345]
respawn

pre-start script
  /usr/bin/docker rm -f drone
end script

script
  /usr/bin/docker run -a drone
end script
```

Commands to start and stop Drone:

```
sudo start drone
sudo stop drone
```

## Systemd

Drone can be configured to work with Systemd to automatically start when the operating system initializes. Here is an example systemd file:

```
[Unit]
Description=Drone container
Requires=docker.service
After=docker.service

[Service]
Restart=always
ExecStart=/usr/bin/docker start -a drone
ExecStop=/usr/bin/docker stop -t 2 drone

[Install]
WantedBy=local.target
```

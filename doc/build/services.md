# Services

Drone uses the `compose` section of the `.drone.yml` to specify supporting containers (ie service containers) that should be started and linked to your build container. The `compose` section of the `.drone.yml` is modeled after `docker-compose`:

```
compose:
  [container_name:]
    image: [image_name]
    [options]
```

Example configuration that composes a Postgres and Redis container:

```yaml
compose:
  cache:
    image: redis
  database:
    image: postgres
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=mysecretpassword
```

## Service linking

Service containers share the same network as the build container (using `--net=<container>`). This means that you can access your service container using `localhost` or `127.0.0.1`.

## Service options

The service container configuration options:

* `image` - any valid Docker image name
* `pull` - if true, will always attempt to pull the latest image
* `environment` - list of environment variables, declared in `name=value` format
* `privileged` - if true, runs the container with extended privileges [1]
* `volumes` - list of bind mounted volumes on the host machine [1]
* `net` - sets the container [network mode](https://docs.docker.com/articles/networking/#container-networking) [1]

[1] Some build options are disabled for security reasons, including `volumes`, `privileged` and `net`. To enable these options, a system administrator must white-list your repository as trusted. This can be done via the repository settings screen.

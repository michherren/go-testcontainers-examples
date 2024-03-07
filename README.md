# Go Testcontainers examples

This is a minimal sample collection demonstrating the Go Testcontainers project: https://golang.testcontainers.org/

## Prerequisites

You need a supported container environment on your system. Docker-Desktop is recommended, but you can get it to work with Podman as well:

The following steps are required for running testcontainers on an ARM Mac with Podman 4.9.3:
```
podman machine stop
podman machine set --rootful
podman machine start
```

you need to add this line to `~/.testcontainers.properties`:
```
ryuk.container.privileged=true
```
and set the following ENVs to ~/.bashrc or ~/.zshrc:
```
export DOCKER_HOST=unix://$(podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}')
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
```
## Running the examples

```
go test ./...
```


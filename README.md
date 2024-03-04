# Go Testcontainers examples

This is a minimal sample collection demonstrating the Go Testcontainers project: https://golang.testcontainers.org/

**This is not a template application** It omits a couple of aspects like complete error-handling, cleanup etc.

## Prerequisites

You need a supported container environment on your system. Docker-Desktop is recommended, but you can get it to work with Podman as well:

Certain modules do require root. To run a rootful setup add the following command needs to be done on a shutdown Podman machine:
```
podman machine stop
podman machine set --rootful
podman machine start
```

you need to add this line to `~/.testcontainers.properties`:
```
ryuk.container.privileged=true
```
and set the following ENVs in your ~/.bashrc or ~/.zshrc (or what ever):
```
export DOCKER_HOST=unix://$(podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}')
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock

alias docker=podman
```
## Running the examples

```
go test ./...
```


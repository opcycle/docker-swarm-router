# Routing Service for Docker Swarm

[![Docker Stars](https://img.shields.io/docker/stars/opcycle/swarm-router.svg?style=flat-square)](https://hub.docker.com/r/opcycle/swarm-router) [![Docker Pulls](https://img.shields.io/docker/pulls/opcycle/swarm-router.svg?style=flat-square)](https://hub.docker.com/r/opcycle/swarm-router)

Swarm Router is a HTTP Routing service for Docker in Swarm mode that makes deploying microservices easy. It configures itself automatically and dynamically using services labels.

#### Docker Images

- All images based on Fedora Linux
- [GitHub actions builds](https://github.com/opcycle/docker-swarm-router/actions) 
- [Docker Hub](https://hub.docker.com/r/opcycle/swarm-router)

### Features

- No external config files needed making for easy deployments
- Automatic service discovery and load balancing handled by Docker
- Scaled and maintained by the Swarm for high resilience and performance

### Run the Service

The Router service acts as a reverse proxy in your cluster. It exposes port 80
to the public an redirects all requests to the correct service in background.
It is important that the router service can reach other services via the Swarm
network (that means they must share a network).

```bash
docker service create --name router \
  --network routing \
  -p 80:80 \
  -p 443:443 \
  --mount type=bind,source=/var/run/docker.sock,destination=/var/run/docker.sock \
  --constraint node.role==manager \
  opcycle/swarm-router
```

It is important to mount the docker socket, otherwise the service can't update
its configuration.

The Router service should be scaled to multiple nodes to prevent short outages
when the node with the router service becomes unresponsive (use `--replicas X` when starting the service).

### Register a Service for Router

A service can easily be configured using router. You must simply provide a label
`ingress.host` which determines the hostname under wich the service should be
publicly available.

## Configuration Labels

Additionally to the hostname you can also map another port and path of your service.

| Label   | Required | Default | Description |
| ------- | -------- | ------- | ----------- |
| `router.host` | `yes` | `-`      | When configured router is enabled. The hostname which should be mapped to the service. Multiple domain supported using `router.host0` .. `router.hostN` |
| `router.port` | `no`  | `80`    | The port which serves the service in the cluster. |
| `router.path` | `no`  | `/`     | A optional path which is prefixed when routing requests to the service. |
| `router.max_body_size` | `no` | `10m` | Max request body size | 
| `router.proxy_timeout` | `no` | `600` | Proxy timeout | 


### Run a Service with Enabled Router

It is important to run the service which should be used for ingress that it
shares a network. A good way to do so is to create a common network `routing`
(`docker network create --driver overlay routing`).

To start a service with router simply pass the required labels on creation.

```bash
docker service create --name my-service \
  --network routing \
  --label router.host=my-service.company.tld \
  nginx
```

It is also possible to later add a service to router using `service update`.

```bash
docker service update \
  --label-add router.host=my-service.company.tld \
  --label-add router.port=8080 \
  my-service
```

# Contributing
We'd love for you to contribute to this container. You can request new features by creating an [issue](https://github.com/opcycle/docker-swarm-router/issues), or submit a [pull request](https://github.com/opcycle/docker-swarm-router/pulls) with your contribution.

# Issues
If you encountered a problem running this container, you can file an issue. For us to provide better support, be sure to include the following information in your issue:

- Host OS and version
- Docker version
- Output of docker info
- Version of this container
- The command you used to run the container, and any relevant output you saw (masking any sensitive information)

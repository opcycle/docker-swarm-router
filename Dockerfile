FROM golang AS build

COPY . /src
RUN cd /src/router && go build

FROM fedora

LABEL maintainer="OpCycle <oss@opcycle.net>"
LABEL repository="https://github.com/opcycle/docker-swarm-router"

RUN dnf install -y nginx \
    && ln -sf /dev/stdout /var/log/nginx/access.log \
    && ln -sf /dev/stderr /var/log/nginx/error.log

ENV DOCKER_HOST "unix:///var/run/docker.sock"
ENV UPDATE_INTERVAL "1"
ENV OUTPUT_FILE "/etc/nginx/conf.d/proxy.conf"
ENV TEMPLATE_FILE "/opt/nginx/router.tpl"

COPY --from=build /src/router/router /usr/sbin/swarm-router

ADD router/router.tpl /etc/nginx
ADD nginx.conf /etc/nginx/nginx.conf

HEALTHCHECK --interval=3s --timeout=3s \
	CMD curl -f http://localhost/health || exit 1

ENTRYPOINT ["/usr/sbin/swarm-router"]

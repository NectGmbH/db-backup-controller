FROM golang:1.22.5 AS builder

COPY . /go/src/github.com/NectGmbH/db-backup-controller
WORKDIR /go/src/github.com/NectGmbH/db-backup-controller

RUN set -ex \
 && apt-get update \
 && apt-get install --no-install-recommends -y \
      git \
 && mkdir /build \
 && cd /go/src/github.com/NectGmbH/db-backup-controller/cmd/backup-controller \
 && go build \
      -ldflags "-s -w -X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly \
      -modcacherw \
      -trimpath \
      -o /build/backup-controller \
 && cd /go/src/github.com/NectGmbH/db-backup-controller/cmd/backup-runner \
 && go build \
      -ldflags "-s -w -X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly \
      -modcacherw \
      -trimpath \
      -o /build/backup-runner


FROM debian:12-slim

LABEL maintainer "Knut Ahlers <ka@nect.com>"

RUN set -ex \
 && apt-get update \
 && apt-get install --no-install-recommends -y \
      ca-certificates \
 && rm -rf /var/lib/apt/*

COPY --from=builder /build/* /usr/local/bin/

EXPOSE 3000

ENTRYPOINT ["/usr/local/bin/backup-controller"]
CMD ["--"]

# vim: set ft=Dockerfile:

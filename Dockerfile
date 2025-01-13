FROM --platform=$BUILDPLATFORM golang:1.23.4@sha256:7ea4c9dcb2b97ff8ee80a67db3d44f98c8ffa0d191399197007d8459c1453041 AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /go/src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
ARG GOOS=${TARGETOS}
ARG GOARCH=${TARGETARCH}
RUN make build

FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/ubi-micro:9.5@sha256:f6e0a71b7e0875b54ea559c2e0a6478703268a8d4b8bdcf5d911d0dae76aef51
COPY --from=build --chown=0:0 /go/src/bin/ /usr/local/bin/
COPY --from=build --chown=0:0 /go/src/inspect /usr/local/bin/
USER 1001
ENTRYPOINT ["/usr/local/bin/inspect"]

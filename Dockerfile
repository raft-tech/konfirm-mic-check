FROM --platform=$BUILDPLATFORM golang:1.23.5@sha256:51a6466e8dbf3e00e422eb0f7a97ac450b2d57b33617bbe8d2ee0bddcd9d0d37 AS build
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

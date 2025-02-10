FROM --platform=$BUILDPLATFORM golang:1.23.6@sha256:927112936d6b496ed95f55f362cc09da6e3e624ef868814c56d55bd7323e0959 AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /go/src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
ARG GOOS=${TARGETOS}
ARG GOARCH=${TARGETARCH}
RUN make build

FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/ubi-micro:9.5@sha256:4a2052ef4db4fd1a53b45263b5067eb01d5745fdd300b27986952af27887bc27
COPY --from=build --chown=0:0 /go/src/bin/ /usr/local/bin/
COPY --from=build --chown=0:0 /go/src/inspect /usr/local/bin/
USER 1001
ENTRYPOINT ["/usr/local/bin/inspect"]

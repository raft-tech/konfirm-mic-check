FROM --platform=$BUILDPLATFORM golang:1.23.0@sha256:613a108a4a4b1dfb6923305db791a19d088f77632317cfc3446825c54fb862cd AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /go/src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
ARG GOOS=${TARGETOS}
ARG GOARCH=${TARGETARCH}
RUN make build

FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/ubi-micro:9.4-13@sha256:9dbba858e5c8821fbe1a36c376ba23b83ba00f100126f2073baa32df2c8e183a
COPY --from=build --chown=0:0 /go/src/bin/ /usr/local/bin/
COPY --from=build --chown=0:0 /go/src/inspect /usr/local/bin/
USER 1001
ENTRYPOINT ["/usr/local/bin/inspect"]

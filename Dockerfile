FROM --platform=$BUILDPLATFORM golang:1.23.2@sha256:ad5c126b5cf501a8caef751a243bb717ec204ab1aa56dc41dc11be089fafcb4f AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /go/src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
ARG GOOS=${TARGETOS}
ARG GOARCH=${TARGETARCH}
RUN make build

FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi9/ubi-micro:9.5@sha256:becdf7fff4509ee81df982000d0adef858a7ae7995dfb7d774b9ded6a461ebad
COPY --from=build --chown=0:0 /go/src/bin/ /usr/local/bin/
COPY --from=build --chown=0:0 /go/src/inspect /usr/local/bin/
USER 1001
ENTRYPOINT ["/usr/local/bin/inspect"]

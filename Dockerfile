FROM golang:1.21-bullseye AS build-stage

ENV GOPRIVATE=github.com/vmindtech/*

WORKDIR /app

COPY . ./

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o vke-cluster-agent ./cmd/agent

FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=build-stage /app/vke-cluster-agent /vke-cluster-agent

ENTRYPOINT ["/vke-cluster-agent"]
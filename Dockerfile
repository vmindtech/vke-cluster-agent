FROM golang:1.23-bullseye AS build-stage

ENV GOPRIVATE=github.com/vmindtech/*

WORKDIR /app

COPY . ./

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o vke-cluster-agent-application ./cmd/api

FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=build-stage /app/vke-cluster-agent-application /vke-cluster-agent-application
COPY --from=build-stage /app/locale /locale

EXPOSE 80


ENTRYPOINT ["/vke-cluster-agent-application"]
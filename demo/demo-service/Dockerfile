FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/demo-service .

FROM scratch

ARG HEALTH_MODE=healthy
ARG APP_VERSION=dev

ENV PORT=8080 \
    ENVIRONMENT_NAME=envpilot-demo \
    HEALTH_MODE=${HEALTH_MODE} \
    APP_VERSION=${APP_VERSION}

COPY --from=build /out/demo-service /demo-service

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/demo-service"]

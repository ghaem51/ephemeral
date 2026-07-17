FROM golang:1.24-alpine AS build

RUN apk add --no-cache build-base
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/control-plane ./cmd/api

FROM alpine:3.22

RUN apk add --no-cache ca-certificates wget \
    && mkdir -p /var/lib/envpilot

COPY --from=build /out/control-plane /usr/local/bin/control-plane

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/control-plane"]

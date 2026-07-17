# EnvPilot demo service

Build the deterministic healthy and unhealthy images from the repository root:

```bash
docker build \
  --build-arg HEALTH_MODE=healthy \
  --build-arg APP_VERSION=dev \
  -t envpilot/demo-service:healthy \
  demo/demo-service

docker build \
  --build-arg HEALTH_MODE=unhealthy \
  --build-arg APP_VERSION=dev \
  -t envpilot/demo-service:unhealthy \
  demo/demo-service
```

Run one directly for local inspection:

```bash
docker run --rm -p 8080:8080 \
  -e ENVIRONMENT_NAME=local-demo \
  envpilot/demo-service:healthy
```

Then open `http://localhost:8080/`, `/info`, or `/health`.

Configuration variables are `PORT`, `ENVIRONMENT_NAME`, `APP_VERSION`, and
`HEALTH_MODE`. `HEALTH_MODE` accepts only `healthy` or `unhealthy`.

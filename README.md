# EnvPilot

EnvPilot is a self-service manager for local ephemeral demo environments. The
control plane records lifecycle workflows in SQLite and creates isolated demo
containers through the Docker Engine API. The React console shows provisioning
progress, failures, retries, and destruction.

## Local assessment

Prerequisites:

- Docker Desktop or Docker Engine with Compose v2
- `make`
- Node.js and npm only when running frontend checks outside Docker

Start the complete stack from the repository root:

```bash
make dev
```

`make dev` first builds both allowlisted demo images, then builds and starts the
control plane and web console. Open:

- Web console: <http://localhost:3000>
- Control-plane health: <http://localhost:8080/health>

The web container serves the compiled SPA and proxies same-origin `/api`
requests to `control-plane:8080`. This avoids browser CORS configuration in the
local stack. Deployed environments can instead set `VITE_API_BASE_URL` at web
build time and configure an appropriate trusted origin or reverse proxy.

## Root commands

| Command | Purpose |
| --- | --- |
| `make dev` | Build demo images and run the Compose stack |
| `make build` | Build all application and demo images |
| `make test` | Run Go tests and frontend type checking |
| `make lint` | Run Go formatting/vetting and frontend linting |
| `make demo-images` | Build the healthy and unhealthy allowlisted images |
| `make clean` | Stop containers and remove generated web build output |
| `make reset` | Stop containers and delete the SQLite named volume |

`make reset` permanently removes local EnvPilot database state. It does not
remove unrelated Docker resources.

SQLite is stored in the named volume `envpilot-data`. Compose uses the explicit
network `envpilot-network` and containers `envpilot-control-plane` and
`envpilot-web-console`.

You can also use Compose directly after building the demo images:

```bash
make demo-images
docker compose up --build
```

## Docker socket security

For this local assessment, `control-plane` mounts `/var/run/docker.sock` so it
can create and remove demo containers on the host Engine. Access to this socket
is effectively administrative access to the Docker host. A compromised control
plane could use it to control containers or access host resources.

This mount is **not a production design**. A production deployment would put
infrastructure execution behind a restricted worker with narrowly scoped
permissions. On Kubernetes, that would normally mean a dedicated worker using
a constrained service account and explicit workload policies—not mounting a
node's Docker socket into the API process.

EnvPilot additionally limits the assessment executor to two demo image names,
loopback port publishing, no privileged mode, and no host mounts. Those controls
reduce accidental scope but do not make Docker socket access safe for an
untrusted production service.

## Repository layout

```text
apps/control-plane  Go API, workflows, SQLite, and Docker executor
apps/web-console    React operator console
demo/demo-service   Healthy/unhealthy demo workload
```

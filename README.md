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
| `make test` | Run Go tests, frontend type checking, and frontend tests |
| `make lint` | Run Go formatting/vetting and frontend linting |
| `make demo-images` | Build the healthy and unhealthy allowlisted images |
| `make clean` | Stop containers and remove generated web build output |
| `make reset` | Remove managed demo containers, stop services, and delete SQLite data |

`make reset` permanently removes local EnvPilot database state and containers
labeled `envpilot.managed=true`. It does not remove unrelated Docker resources.

SQLite is stored in the named volume `envpilot-data`. Compose uses the explicit
network `envpilot-network` and containers `envpilot-control-plane` and
`envpilot-web-console`.

You can also use Compose directly after building the demo images:

```bash
make demo-images
docker compose up --build
```

## Demo walkthrough

1. Run `make dev` and open <http://localhost:3000>.
2. Create `healthy-demo` with **Healthy demo service** and optional version `1.0.0`.
3. Watch all five create steps complete, then use **Open environment**.
4. Create `failure-demo` with **Simulated health failure** and watch `CHECK_HEALTH` fail while later steps become skipped.
5. Use **Retry** on `failure-demo`. The retry first removes the known failed container and provisions the same saved workload configuration again; because its profile remains intentionally unhealthy, it deterministically demonstrates another health failure.
6. Use **Destroy** on either READY or FAILED environments and confirm the workflow reaches DESTROYED.

Stop the stack with `make clean`. Use `make reset` when you also want to
remove all labeled demo containers and reset SQLite state.

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

The local stack accepts custom Docker image names so operators can test more
than the two bundled demo images. Custom images must already be available to
the Docker Engine and expose an HTTP port. The create form defaults readiness
checks to `/health`, and operators can change that path for each environment.
Operators can also provide up to 100 environment variables in `KEY=VALUE`
format. These values are stored in SQLite and returned by the API in plaintext,
so this feature is intended for non-secret configuration only.
Set `DOCKER_ALLOWED_IMAGES` to a comma-separated allowlist instead of `*` when
custom images are not required. EnvPilot still enforces loopback-only port
publishing, no privileged mode, and no host mounts. Those controls reduce
accidental scope but do not make Docker socket access safe for an untrusted
production service.

## Startup and recovery

The API verifies SQLite initialization and Docker Engine connectivity before it
starts accepting requests. `DOCKER_CONNECT_TIMEOUT` controls the Docker startup
check and defaults to `5s`.

After an unclean process exit, startup marks workflows left `RUNNING` as
`FAILED`, records an interruption message on the running step and environment,
and retains known container information. The environment is therefore visible
as failed and can be retried (which cleans up the known runtime first) or
destroyed. EnvPilot does not silently resume an operation whose side effects
cannot be proven.

When the control plane runs in Compose, health checks reach host-published demo
ports through `host.docker.internal`; browser-facing environment URLs remain on
`localhost`. Compose supplies the Linux `host-gateway` mapping explicitly.

## Future improvements

For a larger production system, useful follow-ups would include OpenTelemetry
traces, Prometheus metrics, and a durable distributed work queue. They are
intentionally excluded from this assessment to keep the control plane small and
the recovery behavior explicit.

## Repository layout

```text
apps/control-plane  Go API, workflows, SQLite, and Docker executor
apps/web-console    React operator console
demo/demo-service   Healthy/unhealthy demo workload
```

# Allow multiple image uploads

> allow user to upload diffrent image not only 2 images

<details><summary>10 previous messages</summary>

> I’ll trace the image-upload path from the API through validation and storage, then remove the two-image restriction and update the relevant tests.
>
> <details><summary>Explored 6 files, a search, a list</summary>
>
> - Searched for `2 images|two images|image|images|upload` in `apps`
> - Listed files
> - Read `./apps/control-plane/internal/config/config.go`
> - Read `./apps/control-plane/cmd/api/main.go`
> - Read `./apps/web-console/src/pages/CreateEnvironmentPage.tsx`
> - Read `./apps/web-console/src/pages/CreateEnvironmentPage.test.tsx`
> - Read `./docker-compose.yml`
> - Read `./README.md`
>
> </details>
>
> The current limitation exists in two places: the console hard-codes the demo image, and the Docker executor rejects anything outside a two-name allowlist. I’m checking the request contract and executor safeguards so arbitrary image names can be accepted without weakening unrelated container restrictions.
>
> <details><summary>Explored 9 files, 2 searches</summary>
>
> - Read `./apps/control-plane/internal/executor/docker/executor.go`
> - Read `./apps/control-plane/internal/usecase/createenvironment/create.go`
> - Read `./apps/control-plane/internal/server/environment_handler.go`
> - Read `./apps/web-console/src/api/environments.ts`
> - Searched for `profile-grid|operator-form|field-help|form-grid` in `styles.css`
> - Searched for `DOCKER_ALLOWED_IMAGES|AllowedImages|DockerImages` in `.`
> - Read `./apps/control-plane/internal/executor/executor_test.go`
> - Read `./apps/control-plane/internal/executor/docker/mapping_test.go`
> - Read `./apps/control-plane/internal/usecase/createenvironment/create_test.go`
> - Read `./apps/web-console/src/styles.css`
> - Read `./Makefile`
>
> </details>
>
> I’ll implement this as a custom Docker-image option (image reference plus container port), while keeping the healthy/unhealthy demo shortcuts. The API will support `*` in its image policy, and the local stack will enable that mode; privilege, mount, and loopback-binding restrictions remain unchanged. Custom images must already be available to the Docker Engine.
>
> <details><summary>Edited 9 files</summary>
>
> - Wrote `./apps/control-plane/internal/config/config.go` (+1 -1)
> - Wrote `./apps/control-plane/internal/executor/docker/executor.go` (+14 -4)
> - Wrote `./apps/control-plane/internal/executor/docker/mapping_test.go` (+14 -3)
> - Wrote `./apps/web-console/src/api/environments.ts` (+2 -2)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.tsx` (+44 -3)
> - Wrote `./docker-compose.yml` (+1 -1)
> - Wrote `./README.md` (+8 -4)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.test.tsx` (+19 -0)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.tsx` (+7 -5)
> - Wrote `./apps/web-console/src/styles.css` (+8 -0)
>
> </details>
>
> The custom path is now wired through the form and API policy. I’m running formatting, type checks, and focused tests next; I’ll also catch any accessibility or layout regressions from the third profile option.
>
> <details><summary>Edited 2 files, ran 5 commands</summary>
>
> - Ran `gofmt -w apps/control-plane/internal/executor/docker/executor.go apps/control-plane/internal/executor/docker/mapping_test.go && git diff --check && git diff -- apps/control-plane/internal/executor/docker/executor.go apps/control-plane/internal/config/config.go apps/web-console/src/pages/CreateEnvironmentPage.tsx apps/web-console/src/pages/CreateEnvironmentPage.test.tsx apps/web-console/src/api/environments.ts apps/web-console/src/styles.css docker-compose.yml README.md && npm --prefix apps/web-console run typecheck && npm --prefix apps/web-console test -- --run apps/web-console/src/pages/CreateEnvironmentPage.test.tsx`
> - Ran `git diff --check && npm --prefix apps/web-console run typecheck && npm --prefix apps/web-console test -- --run src/pages/CreateEnvironmentPage.test.tsx`
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.test.tsx` (+1 -1)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.tsx` (+3 -3)
> - Ran `npm --prefix apps/web-console test -- --run src/pages/CreateEnvironmentPage.test.tsx && npm --prefix apps/web-console run lint && git diff --check`
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.test.tsx` (+1 -1)
> - Ran `docker run --rm -v /Users/jaysonarasteh/Desktop/project/ephemeral/apps/control-plane:/workspace -w /workspace golang:1.24 sh -c 'gofmt -w internal/executor/docker/executor.go internal/executor/docker/mapping_test.go && go test ./internal/executor/docker ./internal/config'`
> - Ran `npm --prefix apps/web-console test -- --run src/pages/CreateEnvironmentPage.test.tsx && npm --prefix apps/web-console run lint && npm --prefix apps/web-console run typecheck`
>
> </details>
>
> The focused checks pass. One test needed a small adjustment because React Query supplies a second mutation context argument; the submitted payload itself is correct. I’m doing the full backend and frontend suites now.
>
> <details><summary>Ran 3 commands</summary>
>
> - Ran `docker run --rm -v /Users/jaysonarasteh/Desktop/project/ephemeral/apps/control-plane:/workspace -w /workspace golang:1.24 go test ./...`
> - Ran `npm --prefix apps/web-console test`
> - Ran `git status --short && git diff --check && git diff --stat && git diff -- apps/control-plane/internal/executor/docker/executor.go apps/web-console/src/pages/CreateEnvironmentPage.tsx apps/web-console/src/styles.css`
>
> </details>
</details>

Implemented support for more than the two demo images.

Changes include:

- Added a “Custom Docker image” profile with editable image name and container port.
- Added wildcard `*` support to `DOCKER_ALLOWED_IMAGES`.
- Enabled custom images by default in local Compose configuration.
- Preserved Docker restrictions: no privileged containers, host mounts, or public port binding.
- Updated API types, documentation, and tests.

Custom images must already exist in the Docker Engine and respond successfully at `/health`.

Validation completed:

- Go test suite passed.
- Frontend: 21 tests passed.
- TypeScript typecheck and ESLint passed.

> allow user to change health check path

<details><summary>6 previous messages</summary>

> I’ll make the health-check path configurable per environment, carry it through persistence and retries, and expose it in the custom-image form while keeping `/health` as the default.
>
> <details><summary>Explored 11 files, a search</summary>
>
> - Read `./apps/control-plane/internal/domain/models.go`
> - Read `./apps/control-plane/internal/usecase/environmentlifecycle/lifecycle.go`
> - Read `./apps/control-plane/internal/storage/sqlite/schema.go`
> - Read `./apps/control-plane/internal/storage/sqlite/environment_repository.go`
> - Read `./apps/control-plane/internal/executor/executor.go`
> - Read `./apps/control-plane/internal/executor/executortest/fake.go`
> - Searched for `EnvironmentSpec|RuntimeInfo|healthPath|HealthPath|containerPort` in `control-plane`
> - Read `./apps/control-plane/internal/usecase/createenvironment/runner.go`
> - Read `./apps/control-plane/internal/usecase/environmentlifecycle/runner.go`
> - Read `./apps/control-plane/internal/storage/sqlite/repository_test.go`
> - Read `./apps/control-plane/internal/server/environment_handler_test.go`
> - Read `./apps/web-console/src/pages/EnvironmentDetailsPage.tsx`
>
> </details>
>
> The executor already carries runtime data between create, start, and health-check steps, so I’m adding the path to that runtime state and the saved environment record. This avoids changing the executor interface and ensures retries use the original selected path.
>
> <details><summary>Edited 18 files, explored 8 files, 2 searches</summary>
>
> - Wrote `./apps/control-plane/internal/domain/models.go` (+3 -0)
> - Wrote `./apps/control-plane/internal/executor/docker/executor.go` (+9 -2)
> - Wrote `./apps/control-plane/internal/server/environment_handler.go` (+5 -1)
> - Wrote `./apps/control-plane/internal/storage/sqlite/environment_repository.go` (+7 -6)
> - Wrote `./apps/control-plane/internal/storage/sqlite/schema.go` (+37 -0)
> - Wrote `./apps/control-plane/internal/usecase/createenvironment/create.go` (+11 -1)
> - Wrote `./apps/control-plane/internal/usecase/createenvironment/runner.go` (+0 -0)
> - Wrote `./apps/control-plane/internal/usecase/environmentlifecycle/runner.go` (+3 -2)
> - Wrote `./apps/web-console/src/api/environments.ts` (+2 -0)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.tsx` (+16 -0)
> - Wrote `./apps/web-console/src/pages/EnvironmentDetailsPage.tsx` (+1 -0)
>
> <details><summary>Explored 5 files, 2 searches</summary>
>
> - Read `./apps/web-console/src/pages/CreateEnvironmentPage.tsx`
> - Read `./apps/control-plane/internal/domain/models.go`
> - Searched for `Environment = \{|Environment\s*=|healthCheckPath|EnvironmentSpec\{` in `web-console`
> - Read `./apps/web-console/src/pages/EnvironmentDetailsPage.test.tsx`
> - Read `./apps/control-plane/internal/usecase/createenvironment/create_test.go`
> - Searched for `testEnvironmentResult|domain.Environment\{` in `environment_handler_test.go`
> - Read `./apps/control-plane/internal/server/environment_handler_test.go`
>
> </details>
>
> - Wrote `./apps/control-plane/internal/server/environment_handler_test.go` (+3 -3)
> - Wrote `./apps/control-plane/internal/usecase/createenvironment/create_test.go` (+20 -0)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.test.tsx` (+17 -0)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.tsx` (+13 -13)
> - Wrote `./apps/web-console/src/pages/EnvironmentDetailsPage.test.tsx` (+1 -0)
>
> <details><summary>Explored 4 files</summary>
>
> - Read `./apps/control-plane/internal/usecase/createenvironment/create_test.go`
> - Read `./apps/control-plane/internal/storage/sqlite/schema.go`
> - Read `./apps/control-plane/internal/storage/sqlite/environment_repository.go`
> - Read `./apps/web-console/src/api/environments.ts`
>
> </details>
>
> - Wrote `./README.md` (+2 -1)
> - Wrote `./apps/control-plane/internal/executor/docker/mapping_test.go` (+6 -3)
> - Wrote `./apps/control-plane/internal/storage/sqlite/repository_test.go` (+4 -1)
> - Wrote `./apps/control-plane/internal/usecase/createenvironment/create_test.go` (+3 -0)
>
> - Wrote `./apps/control-plane/internal/storage/sqlite/repository_test.go` (+4 -3)
>
> </details>
>
> The path is now editable for every workload profile, returned by the API, shown on the details page, stored in SQLite, and reused by retry workflows. I’ve also added validation that it starts with `/` and contains no query string or fragment; now I’m verifying migration and end-to-end behavior through the test suites.
>
> <details><summary>Edited 2 files, explored 3 files, a search, ran 5 commands</summary>
>
> - Ran `docker run --rm -v /Users/jaysonarasteh/Desktop/project/ephemeral/apps/control-plane:/workspace -w /workspace golang:1.24 sh -c 'gofmt -w internal/domain/models.go internal/usecase/createenvironment/create.go internal/usecase/environmentlifecycle/runner.go internal/executor/docker/executor.go internal/executor/docker/mapping_test.go internal/storage/sqlite/schema.go internal/storage/sqlite/environment_repository.go internal/storage/sqlite/repository_test.go internal/server/environment_handler.go internal/server/environment_handler_test.go internal/usecase/createenvironment/create_test.go && go test ./...'`
>
> - Ran `npm --prefix apps/web-console run typecheck && npm --prefix apps/web-console test && npm --prefix apps/web-console run lint`
>
> - Ran `sed -n '95,135p' apps/control-plane/internal/usecase/environmentlifecycle/lifecycle_test.go; rg -n "func failedEnvironment|HealthCheckPath" apps/control-plane/internal/usecase/environmentlifecycle/lifecycle_test.go; git diff --check; git status --short; git diff --stat`
>
> <details><summary>Explored 5 files, a search</summary>
>
> - Searched for `func createEnvironment` in `lifecycle_test.go`
> - Read `./apps/control-plane/internal/usecase/environmentlifecycle/lifecycle_test.go`
> - Read `./apps/control-plane/internal/domain/models.go`
> - Read `./apps/control-plane/internal/executor/docker/executor.go`
> - Read `./apps/control-plane/internal/usecase/environmentlifecycle/lifecycle_test.go`
> - Read `./apps/control-plane/internal/executor/docker/executor.go`
>
> </details>
>
> - Wrote `./apps/control-plane/internal/usecase/createenvironment/create.go` (+3 -0)
> - Wrote `./apps/control-plane/internal/usecase/environmentlifecycle/lifecycle_test.go` (+8 -5)
>
> - Ran `docker run --rm -v /Users/jaysonarasteh/Desktop/project/ephemeral/apps/control-plane:/workspace -w /workspace golang:1.24 sh -c 'gofmt -w internal/usecase/createenvironment/create.go internal/usecase/environmentlifecycle/lifecycle_test.go && go test ./internal/usecase/createenvironment ./internal/usecase/environmentlifecycle ./internal/executor/docker ./internal/storage/sqlite ./internal/server'`
>
> - Ran `git diff --check && git diff --stat && git status --short`
>
> </details>
</details>

Implemented configurable health-check paths per environment.

- Added a “Health check path” field, defaulting to `/health`.
- Validates paths start with `/` and exclude queries/fragments.
- Persists the path in SQLite with automatic migration.
- Uses the saved path during creation and retries.
- Displays the path on the environment details page.
- Updated API types, documentation, and tests.

Verification passed:

- All Go tests.
- 22 frontend tests.
- TypeScript and ESLint checks.

> allow user to set env for the container

<details><summary>8 previous messages</summary>

> I’ll add per-environment variables to the create form and API, persist them for retries, and pass them into Docker alongside the existing managed variables. I’ll validate names and prevent overriding `ENVIRONMENT_NAME` and `APP_VERSION`.
>
> <details><summary>Explored 3 files, a search</summary>
>
> - Read `./apps/control-plane/internal/executor/docker/mapping.go`
> - Read `./apps/control-plane/internal/usecase/createenvironment/create_test.go`
> - Read `./apps/control-plane/internal/storage/sqlite/schema.go`
> - Searched for `textarea|Environment information|applicationVersion` in `web-console`
>
> </details>
>
> I’m using a simple `KEY=VALUE`, one-per-line format in the console. Empty values are supported, duplicate names are rejected, and the saved list will be returned in environment details and reused unchanged on retry.
>
> <details><summary>Edited 18 files, explored 3 files</summary>
>
> <details><summary>Explored 5 files</summary>
>
> - Read `./apps/control-plane/internal/domain/models.go`
> - Read `./apps/control-plane/internal/usecase/createenvironment/create.go`
> - Read `./apps/control-plane/internal/usecase/createenvironment/create.go`
> - Read `./apps/control-plane/internal/usecase/createenvironment/create.go`
> - Read `./apps/control-plane/internal/server/environment_handler.go`
>
> </details>
>
> - Wrote `./apps/control-plane/internal/domain/models.go` (+2 -0)
> - Wrote `./apps/control-plane/internal/executor/docker/mapping.go` (+1 -0)
> - Wrote `./apps/control-plane/internal/usecase/createenvironment/create.go` (+28 -0)
> - Wrote `./apps/control-plane/internal/usecase/environmentlifecycle/runner.go` (+1 -0)
>
> - Wrote `./apps/control-plane/internal/server/environment_handler.go` (+4 -0)
> - Wrote `./apps/control-plane/internal/storage/sqlite/environment_repository.go` (+20 -7)
> - Wrote `./apps/control-plane/internal/storage/sqlite/schema.go` (+37 -0)
>
> - Wrote `./apps/web-console/src/api/environments.ts` (+2 -0)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.tsx` (+28 -0)
> - Wrote `./apps/web-console/src/pages/EnvironmentDetailsPage.test.tsx` (+1 -0)
> - Wrote `./apps/web-console/src/pages/EnvironmentDetailsPage.tsx` (+3 -0)
> - Wrote `./apps/web-console/src/styles.css` (+8 -2)
>
> - Wrote `./apps/control-plane/internal/executor/docker/mapping_test.go` (+2 -1)
> - Wrote `./apps/control-plane/internal/server/environment_handler_test.go` (+2 -2)
> - Wrote `./apps/control-plane/internal/storage/sqlite/repository_test.go` (+3 -1)
> - Wrote `./apps/control-plane/internal/usecase/createenvironment/create_test.go` (+3 -0)
> - Wrote `./apps/control-plane/internal/usecase/environmentlifecycle/lifecycle_test.go` (+2 -1)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.test.tsx` (+13 -0)
>
> </details>
>
> The main data path is in place. I’m tightening two edge cases before testing: empty lists must serialize as `[]` rather than `null`, and the UI will explicitly warn that these values are stored and displayed in plaintext, so they should not be used for secrets.
>
> <details><summary>Edited 6 files, explored 4 files</summary>
>
> - Wrote `./README.md` (+3 -0)
> - Wrote `./apps/control-plane/internal/server/environment_handler.go` (+1 -1)
> - Wrote `./apps/control-plane/internal/storage/sqlite/environment_repository.go` (+2 -2)
> - Wrote `./apps/control-plane/internal/storage/sqlite/repository_test.go` (+3 -0)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.tsx` (+1 -1)
>
> <details><summary>Explored 5 files</summary>
>
> - Read `./apps/control-plane/internal/storage/sqlite/repository_test.go`
> - Read `./apps/control-plane/internal/server/environment_handler.go`
> - Read `./apps/control-plane/internal/server/environment_handler.go`
> - Read `./apps/control-plane/internal/domain/models.go`
> - Read `./apps/control-plane/internal/storage/sqlite/environment_repository.go`
>
> </details>
>
> - Wrote `./apps/control-plane/internal/storage/sqlite/repository_test.go` (+1 -0)
> - Wrote `./apps/control-plane/internal/usecase/createenvironment/create_test.go` (+7 -1)
>
> </details>
>
> Environment variables now flow through validation, SQLite, Docker creation, API responses, details, and retries. I’ve included coverage for reserved names, duplicates, invalid syntax, empty values, and schema migration; I’m running the complete checks now.
>
> <details><summary>Edited 3 files, ran 7 commands</summary>
>
> - Ran `npm --prefix apps/web-console run typecheck && npm --prefix apps/web-console test && npm --prefix apps/web-console run lint`
> - Ran `docker run --rm -v /Users/jaysonarasteh/Desktop/project/ephemeral/apps/control-plane:/workspace -w /workspace golang:1.24 sh -c 'gofmt -w internal/domain/models.go internal/usecase/createenvironment/create.go internal/usecase/createenvironment/create_test.go internal/usecase/environmentlifecycle/runner.go internal/usecase/environmentlifecycle/lifecycle_test.go internal/executor/docker/mapping.go internal/executor/docker/mapping_test.go internal/server/environment_handler.go internal/server/environment_handler_test.go internal/storage/sqlite/schema.go internal/storage/sqlite/environment_repository.go internal/storage/sqlite/repository_test.go && go test ./...'`
> - Wrote `./apps/control-plane/internal/executor/executor_test.go` (+1 -1)
> - Ran `docker run --rm -v /Users/jaysonarasteh/Desktop/project/ephemeral/apps/control-plane:/workspace -w /workspace golang:1.24 sh -c 'gofmt -w internal/executor/executor_test.go && go test ./...'`
> - Ran `git diff --check && git diff --stat && git status --short; sed -n '15,65p' apps/control-plane/internal/usecase/createenvironment/create.go; sed -n '205,255p' apps/control-plane/internal/usecase/createenvironment/create.go; sed -n '385,435p' apps/web-console/src/styles.css`
> - Wrote `./apps/control-plane/internal/usecase/createenvironment/create.go` (+3 -0)
> - Wrote `./apps/web-console/src/pages/CreateEnvironmentPage.tsx` (+1 -0)
> - Ran `docker run --rm -v /Users/jaysonarasteh/Desktop/project/ephemeral/apps/control-plane:/workspace -w /workspace golang:1.24 sh -c 'gofmt -w internal/usecase/createenvironment/create.go && go test ./internal/usecase/createenvironment' && npm --prefix apps/web-console run typecheck && git diff --check`
> - Ran `docker run --rm -v /Users/jaysonarasteh/Desktop/project/ephemeral/apps/control-plane:/workspace -w /workspace golang:1.24 sh -c 'gofmt -w internal/usecase/createenvironment/create.go && go test ./internal/usecase/createenvironment'`
> - Ran `npm --prefix apps/web-console run typecheck && git diff --check`
>
> </details>
</details>

Implemented per-container environment variables.

- Create form accepts one `KEY=VALUE` entry per line.
- Variables are validated, persisted in SQLite, passed to Docker, displayed in details, and preserved during retries.
- Supports empty values and values containing `=`.
- Rejects duplicates, invalid names, oversized entries, and reserved `ENVIRONMENT_NAME`/`APP_VERSION`.
- Limited to 100 variables.
- Values are stored and returned in plaintext, so the UI warns against using secrets.

All Go tests and 23 frontend tests pass. TypeScript and ESLint checks also pass.

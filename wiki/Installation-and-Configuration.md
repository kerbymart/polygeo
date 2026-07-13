# Installation and Configuration

## Requirements

- Go 1.25 or newer for source builds
- Git for cloning the repository
- A readable data directory
- Country packages following the [Country Data Packages](Country-Data-Packages) contract when geolocation data is required

Polygeo starts successfully with an empty readable data directory. In that state, `/status` reports zero countries and country-specific endpoints return `country_not_found`.

## Build from source

```bash
git clone https://github.com/kerbymart/polygeo.git
cd polygeo
go mod download
go build -o bin/polygeo ./cmd/polygeo
```

The equivalent Make target is:

```bash
make build
```

The compiled binary does not contain country data. It reads country packages from the filesystem every time the process starts.

## Run from source

```bash
go run ./cmd/polygeo
```

## Run the compiled binary

```bash
./bin/polygeo
```

## Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `POLYGEO_ADDR` | `:8080` | Address passed to the Echo server. |
| `POLYGEO_DATA_DIR` | `./data` | Directory scanned for country-package subdirectories during startup. |

Example:

```bash
POLYGEO_ADDR=127.0.0.1:9090 \
POLYGEO_DATA_DIR=/var/lib/polygeo/data \
./bin/polygeo
```

The configured data directory must exist and be readable. Polygeo exits with status `1` when it cannot load the directory or any discovered country package.

## Docker build

```bash
docker build -t polygeo .
```

The Dockerfile uses a Go 1.25 Alpine build stage and builds the executable with `CGO_ENABLED=0`, `-trimpath`, and stripped symbol/debug information. The runtime image uses Alpine and runs Polygeo as the non-root `polygeo` user.

## Docker run with a mounted data directory

```bash
docker run --rm \
  -p 8080:8080 \
  -v "$PWD/data:/app/data:ro" \
  polygeo
```

The image configures:

```text
POLYGEO_ADDR=:8080
POLYGEO_DATA_DIR=/app/data
```

The read-only mount supplies country packages independently of the binary. Restart the container after changing data.

## Run with a different host port

```bash
docker run --rm \
  -p 9090:8080 \
  -v "$PWD/data:/app/data:ro" \
  polygeo
```

Polygeo still listens on port `8080` inside the container; Docker maps host port `9090` to it.

## Run with a different container address

```bash
docker run --rm \
  -e POLYGEO_ADDR=:9090 \
  -p 9090:9090 \
  -v "$PWD/data:/app/data:ro" \
  polygeo
```

## Verify a running service

```bash
curl http://localhost:8080/status
curl http://localhost:8080/countries
```

Example status response with two loaded packages:

```json
{
  "status": "OK",
  "countries": 2
}
```

## Startup logging

On successful registry loading, Polygeo logs the listen address, data directory, and country count before starting Echo.

On a registry-loading failure, Polygeo logs the directory and error and exits. The server never starts with a partially loaded registry.

## Updating country data

Polygeo has no file watcher and no runtime reload endpoint.

To apply changes:

1. Update the country directory and files.
2. Restart the process or container.
3. Check `/status` and `/countries`.
4. Verify the affected country description and lookup endpoints.

## Reverse proxy deployment

Polygeo accepts a normal HTTP listen address through `POLYGEO_ADDR`. TLS termination, host routing, rate limiting, and authentication are not implemented by the service and must be provided by the deployment environment when required.

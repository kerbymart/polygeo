# Development and Testing

## Source layout

```text
cmd/polygeo/main.go             Process configuration, registry loading, and server startup
internal/geo/manifest.go        Manifest types and validation
internal/geo/registry.go        Country discovery, GeoJSON loading, registry queries, and locate flow
internal/geo/geometry.go        Polygon parsing, bounds, and point-in-polygon logic
internal/geo/geometry_test.go   Geometry behavior tests
internal/geo/registry_test.go   Package loading and hierarchical lookup tests
internal/httpapi/server.go      Echo routes, handlers, coordinate validation, and error responses
data/README.md                  Runtime data-directory guidance
wiki/                           GitHub Wiki source pages
```

## Local commands

Download dependencies:

```bash
go mod download
```

Format source:

```bash
gofmt -w ./cmd ./internal
```

Or:

```bash
make fmt
```

Run static analysis:

```bash
go vet ./...
```

Or:

```bash
make vet
```

Run tests:

```bash
go test ./...
```

Or:

```bash
make test
```

Run the race detector:

```bash
go test -race ./...
```

Build the executable:

```bash
go build -o bin/polygeo ./cmd/polygeo
```

Or:

```bash
make build
```

## Current test coverage

### Geometry tests

`internal/geo/geometry_test.go` verifies:

- a point inside an exterior ring matches;
- a point inside a polygon hole does not match;
- an outside point does not match;
- an exterior-boundary point matches;
- a hole-boundary point matches.

### Registry tests

`internal/geo/registry_test.go` creates a temporary country package and verifies:

- manifest and GeoJSON loading;
- country lookup by configured aliases;
- region listing;
- parent-filtered child listing;
- hierarchical coordinate lookup;
- path-traversal rejection in manifest file paths.

## GitHub Actions

`.github/workflows/ci.yml` runs on every push and pull request.

The CI job performs:

```text
go mod download
gofmt -d ./cmd ./internal
go vet ./...
go test -race ./...
go build ./cmd/polygeo
```

The formatting step prints differences and fails when source files are not formatted.

## Adding geometry tests

Geometry tests are in the `geo` package and can call the unexported geometry parser and containment methods directly.

Add table-driven cases for:

- Polygon exterior and hole behavior;
- MultiPolygon behavior;
- boundary coordinates;
- malformed rings;
- non-finite coordinates;
- bounding-box rejection.

Keep coordinate ordering consistent with GeoJSON:

```text
[longitude, latitude]
```

## Adding package-loading tests

Use `t.TempDir()` to create an isolated data directory. A complete fixture contains:

```text
<temp>/
  XX/
    country.json
    level-one.geojson
    level-two.geojson
```

Write the manifest and GeoJSON files directly in the test, call `LoadDirectory`, then assert registry behavior.

Tests should cover both successful loading and startup-stopping validation errors.

## Adding HTTP tests

The Echo server is constructed by:

```go
server := httpapi.New(registry)
```

HTTP tests can create a temporary registry, pass the Echo instance to `net/http/httptest`, and verify status codes and JSON responses for both route families.

Important cases include:

- unknown country;
- unknown level;
- missing `level` query parameter;
- invalid coordinates;
- nested level relationship validation;
- empty parent-filter results;
- successful partial and complete locate results.

## Data compatibility checks

Before adding a real country package, verify:

1. Every configured file is a `FeatureCollection`.
2. Every feature is a `Feature`.
3. Required name and parent properties are non-empty strings.
4. Geometry types are `Polygon` or `MultiPolygon`.
5. Coordinate order is longitude, latitude.
6. Parent property values exactly correspond to the intended parent names, ignoring case only.
7. Representative coordinates return the expected hierarchy.
8. Points in holes and points near boundaries behave as required.

## Deterministic behavior

Region sorting and first-match lookup are deterministic. Tests for overlapping regions should assert the alphabetically first normalized region according to the loader's parent/name sort order.

## Wiki publication

Files under `wiki/` are the version-controlled source for the GitHub Wiki. `.github/workflows/publish-wiki.yml` publishes those files to the repository wiki after changes reach `main`.

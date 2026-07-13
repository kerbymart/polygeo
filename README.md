# Polygeo

Polygeo is a data-driven geolocation HTTP API written in Go with Echo. It loads country packages from a filesystem directory, validates their manifests and GeoJSON files, builds an in-memory registry, and exposes country, administrative-region, and coordinate-lookup endpoints.

A country is supported by adding a directory containing `country.json` and the GeoJSON files referenced by that manifest. The loader and HTTP routes are generic; adding a conforming country package does not require country-specific Go code or rebuilding the binary. Restart Polygeo after adding or changing data because packages are loaded once during startup.

## Implemented features

- Filesystem country discovery through `POLYGEO_DATA_DIR`.
- Manifest-defined country codes, names, aliases, administrative levels, parent relationships, source files, and GeoJSON property mappings.
- Case-insensitive country lookup by manifest code, manifest name, directory name, or configured alias.
- Case-insensitive level lookup by level ID, plural name, or configured alias.
- GeoJSON `FeatureCollection` loading for `Polygon` and `MultiPolygon` geometries.
- Point-in-polygon lookup with interior-ring hole handling and per-geometry bounding-box checks.
- Hierarchical coordinate lookup in manifest level order.
- Region listing with optional parent filtering.
- Canonical `/countries/{country}` routes and shorter `/{country}` aliases.
- JSON error responses with stable error codes.
- Startup validation that stops the process when a configured country package is invalid.

## Requirements

- Go 1.25 or newer
- One existing data directory, even when it contains no country packages
- GeoJSON `FeatureCollection` files containing `Polygon` or `MultiPolygon` geometries

## Project layout

```text
cmd/polygeo/          Application entry point
internal/geo/         Manifest validation, country registry, GeoJSON loading, and geometry lookup
internal/httpapi/     Echo routes and handlers
data/                 Runtime country data packages
wiki/                 Source files published to the GitHub Wiki
```

## Build

```bash
go mod download
go build -o bin/polygeo ./cmd/polygeo
```

Or:

```bash
make build
```

## Run

The default data directory is `./data` and the default listen address is `:8080`.

```bash
go run ./cmd/polygeo
```

Run a built binary:

```bash
./bin/polygeo
```

Set a different data directory or listen address:

```bash
POLYGEO_DATA_DIR=/var/lib/polygeo/data \
POLYGEO_ADDR=:9090 \
./bin/polygeo
```

### Configuration

| Environment variable | Default | Behavior |
| --- | --- | --- |
| `POLYGEO_DATA_DIR` | `./data` | Directory scanned for country-package subdirectories during startup. |
| `POLYGEO_ADDR` | `:8080` | Address passed to the Echo HTTP server. |

Polygeo exits with a non-zero status when the data directory cannot be read or any discovered country package fails validation or loading.

### Docker

```bash
docker build -t polygeo .
docker run --rm -p 8080:8080 \
  -v "$PWD/data:/app/data:ro" \
  polygeo
```

The image runs as a non-root `polygeo` user. Mounting `/app/data` keeps country data outside the application binary and allows the same image to run with different datasets. Restart the container after changing mounted data.

## Country data packages

Polygeo scans each immediate subdirectory of the configured data directory.

```text
data/
  PH/
    country.json
    provinces.geojson
    municipalities.geojson
  JP/
    country.json
    prefectures.geojson
    municipalities.geojson
```

Discovery rules:

- A directory is loaded only when it contains `country.json`.
- Directory names beginning with `.` or `_` are ignored.
- Non-directory entries are ignored.
- A directory without `country.json` is ignored.
- An invalid manifest, unreadable referenced file, invalid GeoJSON collection, invalid feature, or unsupported geometry stops startup.
- Country lookup identifiers must not conflict across loaded packages.

### Adding a country

1. Create `data/<directory>/` or a directory under the configured `POLYGEO_DATA_DIR`.
2. Add `country.json` using manifest schema version `1`.
3. Add every GeoJSON file referenced by the manifest.
4. Start or restart Polygeo.
5. Confirm the package through `GET /countries` and `GET /countries/{country}`.

No Go source change or binary rebuild is required.

### Manifest example

This manifest maps the Philippines `NAME_1` and `NAME_2` properties to province and municipality levels:

```json
{
  "schema_version": 1,
  "code": "PH",
  "name": "Philippines",
  "aliases": ["philippines", "phl"],
  "levels": [
    {
      "id": "province",
      "plural": "provinces",
      "file": "provinces.geojson",
      "name_property": "NAME_1"
    },
    {
      "id": "municipality",
      "plural": "municipalities",
      "aliases": ["cities"],
      "file": "municipalities.geojson",
      "name_property": "NAME_2",
      "parent_level": "province",
      "parent_property": "NAME_1"
    }
  ]
}
```

### Manifest contract

| Field | Required | Implemented behavior |
| --- | --- | --- |
| `schema_version` | Yes | Must equal `1`. |
| `code` | Yes | Non-empty canonical country code returned by the API. |
| `name` | Yes | Non-empty display name and country lookup identifier. |
| `aliases` | No | Additional case-insensitive country lookup identifiers. |
| `levels` | Yes | Non-empty ordered list. The order controls hierarchical coordinate lookup. |
| `levels[].id` | Yes | Non-empty canonical level identifier returned in API responses. |
| `levels[].plural` | Yes | Non-empty route alias for the level. |
| `levels[].aliases` | No | Additional route aliases for the level. |
| `levels[].file` | Yes | Relative path to a GeoJSON file inside the country directory. Absolute paths and paths escaping the country directory are rejected. |
| `levels[].name_property` | Yes | Feature property containing a non-empty string region name. |
| `levels[].parent_level` | For child levels | ID of an earlier level in the same manifest. |
| `levels[].parent_property` | For child levels | Feature property containing a non-empty string parent name. It must be set together with `parent_level`. |

Level IDs, plural names, and aliases must be unique after case-insensitive normalization. Parent levels must appear before their children.

### GeoJSON contract

Each referenced file must be a GeoJSON `FeatureCollection`. Every item in `features` must:

- have `type` equal to `Feature`;
- contain the configured `name_property` as a non-empty string;
- contain the configured `parent_property` as a non-empty string when the level has a parent;
- contain a `Polygon` or `MultiPolygon` geometry;
- contain at least one ring per polygon;
- contain at least four positions per ring;
- use finite numeric longitude and latitude values in the first two coordinate positions.

The first ring of a polygon is its exterior. Later rings are holes. A point inside a hole is excluded; points on exterior or hole boundaries are treated as matches. Multiple features with the same case-insensitive parent and region name are combined into one logical region with multiple geometries.

## Endpoint reference

Both route families call the same handlers:

```text
/countries/{country}/...
/{country}/...
```

`{country}` matches the manifest code, manifest name, country directory name, or a configured country alias. Matching is case-insensitive.

### Service status

```http
GET /status
```

```json
{
  "status": "OK",
  "countries": 2
}
```

### List loaded countries

```http
GET /countries
```

Countries are sorted by manifest code.

```json
{
  "count": 2,
  "results": [
    {"code": "JP", "name": "Japan"},
    {"code": "PH", "name": "Philippines"}
  ]
}
```

### Describe a country

```http
GET /countries/PH
GET /PH
GET /philippines
```

The response contains the canonical code, display name, configured country aliases, and administrative levels in manifest order.

### List regions with a query parameter

```http
GET /countries/PH/regions?level=province
GET /PH/regions?level=municipality&parent=Cebu
```

The `level` parameter is required. `parent` is optional and matched case-insensitively.

### List regions with a level path

```http
GET /countries/PH/provinces
GET /PH/municipalities?parent=Cebu
```

The level segment accepts the configured level ID, plural name, or alias.

### List child regions with a nested path

```http
GET /countries/PH/provinces/Cebu/municipalities
GET /PH/provinces/Cebu/municipalities
```

The requested child level must declare the requested parent level in its manifest. Otherwise Polygeo returns `400 invalid_level_relationship`.

Example region response:

```json
{
  "country": "PH",
  "level": "municipality",
  "parent": "Cebu",
  "count": 1,
  "results": ["Cebu City"]
}
```

Region results are sorted case-insensitively by parent name and then region name during startup.

### Locate a coordinate

```http
GET /countries/PH/locate?latitude=10.3157&longitude=123.8854
GET /PH/locate?latitude=10.3157&longitude=123.8854
```

Both parameters are required. Latitude must be between `-90` and `90`; longitude must be between `-180` and `180`.

```json
{
  "country": {
    "code": "PH",
    "name": "Philippines"
  },
  "latitude": 10.3157,
  "longitude": 123.8854,
  "regions": {
    "province": "Cebu",
    "municipality": "Cebu City"
  }
}
```

Coordinate lookup processes levels in manifest order. A child level is searched only after its configured parent level matches, and only child regions belonging to that matched parent are considered. The response contains every level that matched. A coordinate that matches no level returns `404 location_not_found`.

### Errors

Errors use this envelope:

```json
{
  "error": {
    "code": "country_not_found",
    "message": "country data package was not found"
  }
}
```

Implemented API error codes include:

| HTTP status | Code | Condition |
| --- | --- | --- |
| `400` | `level_required` | `/regions` was called without `level`. |
| `400` | `invalid_level_relationship` | A nested child route does not match the manifest parent relationship. |
| `400` | `invalid_region_query` | Region query validation failed. |
| `400` | `invalid_latitude` | Latitude is missing, non-numeric, or outside its allowed range. |
| `400` | `invalid_longitude` | Longitude is missing, non-numeric, or outside its allowed range. |
| `404` | `country_not_found` | No loaded country matches the requested identifier. |
| `404` | `level_not_found` | No level matches the requested identifier. |
| `404` | `location_not_found` | The coordinate matches no loaded administrative level. |

## Runtime behavior

- Country data is read and parsed once during startup.
- Loaded registry and geometry data is used in memory for all requests.
- Data-file changes are not watched or reloaded automatically.
- Restarting the process is required to load changes.
- Polygeo performs no database queries or external geolocation API calls.
- Coordinate lookup first checks each geometry bounding box, then performs point-in-polygon tests.
- When geometries overlap within one level, lookup returns the first matching region in the deterministic startup sort order.

## Test and validation

```bash
go test ./...
go test -race ./...
go vet ./...
```

Or:

```bash
make test
make vet
```

The committed tests cover polygon holes and boundaries, manifest and data-package loading, aliases, region listing, and hierarchical coordinate lookup. GitHub Actions runs formatting checks, `go vet`, race-enabled tests, and a build on every push and pull request.

## Documentation

The [Polygeo Wiki](https://github.com/kerbymart/polygeo/wiki) contains detailed pages for architecture, installation, country packages, API behavior, and development.

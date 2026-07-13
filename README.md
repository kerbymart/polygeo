# Polygeo

Polygeo is a data-driven geolocation API written in Go using the Echo Framework. It loads country packages from a filesystem `data` directory at startup; country geometry is **not embedded in the application binary**.

Adding a supported country should normally require only a new directory containing a manifest and GeoJSON files. Country-specific Go code is not required.

## Requirements

- Go 1.25 or newer
- GeoJSON `FeatureCollection` files containing `Polygon` or `MultiPolygon` geometries

## Project layout

```text
cmd/polygeo/          Application entry point
internal/geo/         Country registry, manifest loader, and geometry lookup
internal/httpapi/     Echo routes and handlers
data/                 Runtime country data packages
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

Use a different data directory or address:

```bash
POLYGEO_DATA_DIR=/var/lib/polygeo/data \
POLYGEO_ADDR=:9090 \
./bin/polygeo
```

### Docker

```bash
docker build -t polygeo .
docker run --rm -p 8080:8080 \
  -v "$PWD/data:/app/data:ro" \
  polygeo
```

Mounting `/app/data` allows country packages to be updated independently from the application image.

## Country data packages

Each immediate subdirectory of `data/` represents one country package:

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

Directories beginning with `.` or `_` are ignored. Other directories without a `country.json` file are also ignored.

Polygeo validates and loads all configured country packages during startup. If a manifest or referenced GeoJSON file is invalid, startup fails instead of serving partial or inconsistent data.

### Manifest example

This manifest works with the existing Philippines GeoJSON property names `NAME_1` and `NAME_2`:

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

### Manifest fields

| Field | Required | Description |
| --- | --- | --- |
| `schema_version` | Yes | Manifest schema. Currently `1`. |
| `code` | Yes | Stable country code used in responses, preferably ISO 3166-1 alpha-2. |
| `name` | Yes | Display name. It can also be used in URL lookups. |
| `aliases` | No | Additional URL lookup names. |
| `levels` | Yes | Ordered administrative levels, from broadest to most specific. |
| `levels[].id` | Yes | Canonical singular level name, such as `province`. |
| `levels[].plural` | Yes | Plural route alias, such as `provinces`. |
| `levels[].aliases` | No | Additional route names for the level. |
| `levels[].file` | Yes | GeoJSON filename relative to the country directory. |
| `levels[].name_property` | Yes | Feature property containing the region name. |
| `levels[].parent_level` | For child levels | Canonical ID of an earlier level. |
| `levels[].parent_property` | For child levels | Feature property containing the parent region name. |

The level order controls reverse lookup. A child level is searched only after its configured parent is found.

### Supported GeoJSON

Each referenced file must contain a GeoJSON `FeatureCollection`. Every feature must contain:

- a `Polygon` or `MultiPolygon` geometry;
- a non-empty string at `name_property`;
- a non-empty string at `parent_property` when the level has a parent.

Interior polygon rings are treated as holes. Multiple features with the same region and parent are combined into one logical region.

## Endpoints

Both canonical `/countries/{country}` routes and shorter `/{country}` aliases are available. `{country}` can be the manifest code, name, directory name, or an explicit alias; matching is case-insensitive.

### Service status

```http
GET /status
```

```json
{
  "status": "OK",
  "countries": 1
}
```

### List loaded countries

```http
GET /countries
```

```json
{
  "count": 1,
  "results": [
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

The response includes the country aliases and available administrative levels.

### List regions

Generic query form:

```http
GET /countries/PH/regions?level=province
GET /PH/regions?level=municipality&parent=Cebu
```

Level-name path form:

```http
GET /countries/PH/provinces
GET /PH/municipalities?parent=Cebu
```

Nested compatibility form:

```http
GET /countries/PH/provinces/Cebu/municipalities
GET /PH/provinces/Cebu/municipalities
```

Example response:

```json
{
  "country": "PH",
  "level": "municipality",
  "parent": "Cebu",
  "count": 1,
  "results": ["Cebu City"]
}
```

### Locate a coordinate

```http
GET /countries/PH/locate?latitude=10.3157&longitude=123.8854
GET /PH/locate?latitude=10.3157&longitude=123.8854
```

Example response:

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

Latitude must be between `-90` and `90`; longitude must be between `-180` and `180`.

### Errors

Errors use a consistent JSON envelope:

```json
{
  "error": {
    "code": "country_not_found",
    "message": "country data package was not found"
  }
}
```

Common statuses are:

- `400` for missing or invalid parameters;
- `404` for an unknown country, level, or coordinate outside loaded regions;
- `500` for unexpected server errors.

## Test

```bash
go test ./...
go test -race ./...
```

Or:

```bash
make test
```

The geometry tests cover Polygon holes and boundaries. Registry tests build a temporary country package and verify manifest loading, aliases, region listing, and hierarchical coordinate lookup.

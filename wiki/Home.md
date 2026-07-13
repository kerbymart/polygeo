# Polygeo Wiki

Polygeo is a data-driven geolocation HTTP API written in Go with Echo. It discovers country packages from a filesystem directory, validates their manifests and GeoJSON files during startup, loads the complete registry into memory, and serves country, region-listing, and coordinate-lookup endpoints.

A supported country consists of a directory containing a valid `country.json` manifest and every GeoJSON file referenced by that manifest. Country packages use the same loader and HTTP handlers; no country-specific Go implementation is required.

## Documentation

- [How It Works](How-It-Works) — startup, registry construction, routing, and coordinate lookup internals.
- [Installation and Configuration](Installation-and-Configuration) — build, run, Docker, and environment variables.
- [Country Data Packages](Country-Data-Packages) — directory layout, manifest schema, GeoJSON contract, and validation rules.
- [API Reference](API-Reference) — routes, request parameters, responses, and error codes.
- [Development and Testing](Development-and-Testing) — source layout, local checks, tests, and CI.

## Runtime guarantees

- Country data is loaded once during startup.
- All discovered packages are validated before the HTTP server starts.
- An invalid discovered package stops startup.
- Loaded country and geometry data is read from memory while serving requests.
- Data changes require a process restart.
- Polygeo does not call a database or an external geolocation service.
- `Polygon` and `MultiPolygon` GeoJSON geometries are supported.
- Polygon holes are respected during coordinate lookup.
- Country and administrative-level identifiers are matched case-insensitively.

## Quick start

```bash
go mod download
go run ./cmd/polygeo
```

The default configuration is:

```text
POLYGEO_ADDR=:8080
POLYGEO_DATA_DIR=./data
```

With a country package installed, verify the service:

```bash
curl http://localhost:8080/status
curl http://localhost:8080/countries
curl http://localhost:8080/countries/PH
```

See [Installation and Configuration](Installation-and-Configuration) for deployment instructions and [Country Data Packages](Country-Data-Packages) for the exact package contract.

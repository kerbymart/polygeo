# API Reference

Polygeo returns JSON for every implemented endpoint.

## Route families

Country endpoints are registered in two equivalent forms:

```text
/countries/{country}/...
/{country}/...
```

Examples:

```text
/countries/PH/locate
/PH/locate
```

Both routes execute the same handler and return the same response shape.

`{country}` accepts the country manifest code, manifest name, directory name, or a configured alias. Matching is case-insensitive.

## `GET /status`

Returns service status and the number of loaded country packages.

```bash
curl http://localhost:8080/status
```

```json
{
  "status": "OK",
  "countries": 2
}
```

## `GET /countries`

Lists loaded countries, sorted by canonical country code.

```bash
curl http://localhost:8080/countries
```

```json
{
  "count": 2,
  "results": [
    {
      "code": "JP",
      "name": "Japan"
    },
    {
      "code": "PH",
      "name": "Philippines"
    }
  ]
}
```

## `GET /countries/{country}`

Short alias:

```text
GET /{country}
```

Returns the country description and administrative levels in manifest order.

```bash
curl http://localhost:8080/countries/PH
```

```json
{
  "code": "PH",
  "name": "Philippines",
  "aliases": ["philippines", "phl"],
  "levels": [
    {
      "id": "province",
      "plural": "provinces"
    },
    {
      "id": "municipality",
      "plural": "municipalities",
      "aliases": ["cities"],
      "parent_level": "province"
    }
  ]
}
```

Unknown countries return:

```json
{
  "error": {
    "code": "country_not_found",
    "message": "country data package was not found"
  }
}
```

with HTTP status `404`.

## `GET /countries/{country}/regions`

Short alias:

```text
GET /{country}/regions
```

Lists regions for an administrative level.

### Query parameters

| Parameter | Required | Description |
| --- | --- | --- |
| `level` | Yes | Level ID, plural name, or configured level alias. |
| `parent` | No | Case-insensitive parent-region filter. |

Examples:

```bash
curl 'http://localhost:8080/countries/PH/regions?level=province'
```

```bash
curl 'http://localhost:8080/PH/regions?level=municipality&parent=Cebu'
```

Response:

```json
{
  "country": "PH",
  "level": "municipality",
  "parent": "Cebu",
  "count": 1,
  "results": ["Cebu City"]
}
```

When the parent filter matches no regions, the endpoint returns HTTP `200` with `count: 0` and an empty `results` array.

Calling this route without `level` returns HTTP `400` with `level_required`.

## `GET /countries/{country}/{level}`

Short alias:

```text
GET /{country}/{level}
```

Path-based form of the region-list endpoint. The level segment accepts a level ID, plural name, or alias.

Examples:

```bash
curl http://localhost:8080/countries/PH/provinces
```

```bash
curl 'http://localhost:8080/PH/municipalities?parent=Cebu'
```

The response shape is identical to `/regions`.

## `GET /countries/{country}/{parentLevel}/{parent}/{childLevel}`

Short alias:

```text
GET /{country}/{parentLevel}/{parent}/{childLevel}
```

Lists child regions for one parent using a nested path.

Example:

```bash
curl http://localhost:8080/countries/PH/provinces/Cebu/municipalities
```

Polygeo resolves both level path segments. The child level must declare the resolved parent level as its manifest parent.

A mismatched relationship returns HTTP `400`:

```json
{
  "error": {
    "code": "invalid_level_relationship",
    "message": "the requested child level does not belong to the requested parent level"
  }
}
```

The parent region name itself is used as a case-insensitive filter. An unknown parent name returns a successful empty list because region filtering does not perform separate parent-existence validation.

## `GET /countries/{country}/locate`

Short alias:

```text
GET /{country}/locate
```

Returns administrative regions containing one coordinate.

### Query parameters

| Parameter | Required | Range |
| --- | --- | --- |
| `latitude` | Yes | `-90` through `90` |
| `longitude` | Yes | `-180` through `180` |

Example:

```bash
curl 'http://localhost:8080/countries/PH/locate?latitude=10.3157&longitude=123.8854'
```

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

The `regions` object contains each manifest level that matched. A parent can be present when no child region matches.

A coordinate matching no level returns HTTP `404`:

```json
{
  "error": {
    "code": "location_not_found",
    "message": "the coordinate is outside the loaded administrative regions"
  }
}
```

## Error envelope

Handled request errors use:

```json
{
  "error": {
    "code": "error_code",
    "message": "human-readable description"
  }
}
```

## Implemented errors

| HTTP status | Code | Trigger |
| --- | --- | --- |
| `400` | `level_required` | `/regions` request has no non-empty `level` query parameter. |
| `400` | `invalid_level_relationship` | Nested route requests a child level whose configured parent differs from the requested parent level. |
| `400` | `invalid_region_query` | Internal region-query validation returns an error. |
| `400` | `invalid_latitude` | Latitude is missing, not a number, or outside `-90...90`. |
| `400` | `invalid_longitude` | Longitude is missing, not a number, or outside `-180...180`. |
| `404` | `country_not_found` | Country identifier does not resolve to a loaded package. |
| `404` | `level_not_found` | Level identifier does not resolve within the selected country. |
| `404` | `location_not_found` | Coordinate matches no loaded administrative level. |

Echo handles unmatched routes and unsupported methods outside these application-specific error codes.

## Matching and ordering rules

- Country and level identifiers are trimmed and matched case-insensitively.
- Parent filters are matched case-insensitively.
- Region results follow the startup sort order: parent name, then region name.
- Coordinate lookup follows manifest level order.
- Child lookup is restricted to the matched parent.
- Boundary points are considered matches.
- Same-level overlaps return the first region in startup sort order.

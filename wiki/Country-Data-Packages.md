# Country Data Packages

A Polygeo country package is one immediate subdirectory of `POLYGEO_DATA_DIR` containing:

- `country.json`;
- every GeoJSON file referenced by `country.json`.

The package contains all country-specific configuration. The Go application uses the same loader, geometry engine, registry, and HTTP handlers for every package.

## Directory layout

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

The directory name is itself a country lookup identifier. It does not need to equal the manifest code, but using a stable uppercase country code keeps the data tree predictable.

## Discovery behavior

Polygeo scans only immediate children of the configured data directory.

| Entry | Behavior |
| --- | --- |
| Regular file | Ignored |
| Directory beginning with `.` | Ignored |
| Directory beginning with `_` | Ignored |
| Directory without `country.json` | Ignored |
| Directory with `country.json` | Loaded and validated |

One invalid discovered package stops startup for the complete service.

## Complete manifest example

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

## Top-level manifest fields

### `schema_version`

Required integer. The implemented schema version is `1`. Any other value stops startup.

### `code`

Required non-empty string. This is the canonical country code returned in API responses and one accepted country identifier.

### `name`

Required non-empty string. This is the display name returned by the API and one accepted country identifier.

### `aliases`

Optional array of additional country identifiers. Aliases are matched case-insensitively.

### `levels`

Required non-empty ordered array of administrative-level definitions.

The array order controls hierarchical coordinate lookup. Parent levels must appear before child levels.

## Administrative-level fields

### `id`

Required non-empty canonical level identifier. It becomes the key in locate responses:

```json
{
  "regions": {
    "province": "Cebu",
    "municipality": "Cebu City"
  }
}
```

### `plural`

Required non-empty route identifier. For example, a level with `plural: "provinces"` is accessible through:

```text
/countries/PH/provinces
```

### `aliases`

Optional route identifiers for the same level. For example, `cities` can resolve to a canonical `municipality` level.

Within one country, every level ID, plural name, and alias must be unique after trimming and case-insensitive normalization.

### `file`

Required relative path to the level's GeoJSON file.

The path must remain inside the country directory. Polygeo rejects absolute paths and paths that escape through `..`.

Valid examples:

```json
"file": "provinces.geojson"
```

```json
"file": "administrative/provinces.geojson"
```

Invalid examples:

```json
"file": "/etc/data.geojson"
```

```json
"file": "../shared/data.geojson"
```

### `name_property`

Required property name. Every feature in the file must contain this property as a non-empty string.

### `parent_level`

Optional canonical ID of an earlier level in the same manifest.

### `parent_property`

Required when `parent_level` is set. Every child feature must contain this property as a non-empty string identifying its parent region.

`parent_level` and `parent_property` must either both be present or both be absent.

## GeoJSON file contract

Each configured file must decode to:

```json
{
  "type": "FeatureCollection",
  "features": []
}
```

Every feature must have:

```json
{
  "type": "Feature",
  "properties": {},
  "geometry": {}
}
```

Supported geometry types are exactly:

- `Polygon`;
- `MultiPolygon`.

Other geometry types stop startup.

## Coordinate structure

Each polygon must have at least one ring. Every ring must have at least four positions. Every position must contain at least two finite numbers:

```json
[longitude, latitude]
```

Polygeo uses the first two values as X and Y. Extra coordinate dimensions do not participate in lookup.

## Polygon rings and holes

For each Polygon:

- ring `0` is the exterior;
- rings `1...n` are holes.

A point inside the exterior and outside all holes matches. A point inside a hole does not match. Exterior and hole boundary points are treated as matches.

For a MultiPolygon, a point matches when any polygon matches.

## Region grouping

A logical region is keyed by the case-insensitive normalized combination of:

```text
parent name + region name
```

When multiple features have the same key, Polygeo stores one region with multiple geometries. This supports islands, detached areas, and datasets that split one administrative region across several features.

## Sort order

After loading a level, Polygeo sorts regions case-insensitively by:

1. parent name;
2. region name.

Region-list responses use this order. Coordinate lookup also scans regions in this order and returns the first match when same-level geometries overlap.

## Country identifiers

A loaded country can be requested by:

- manifest code;
- manifest name;
- directory name;
- any manifest alias.

All identifiers are trimmed and compared case-insensitively. Two country packages cannot claim the same normalized identifier.

## Adding a package

1. Create an immediate child directory under `POLYGEO_DATA_DIR`.
2. Add a schema-version `1` manifest.
3. Add all referenced GeoJSON files.
4. Validate the JSON and property names against the source dataset.
5. Start or restart Polygeo.
6. Confirm the country appears in `GET /countries`.
7. Confirm `GET /countries/{country}` reports the expected levels.
8. Test region listing and representative coordinates.

The package is available as soon as Polygeo starts successfully. No application source change or binary rebuild is part of this process.

## Updating a package

Country files are not watched after startup. Replace or edit the files, then restart the service. Startup validation applies to the complete updated package before requests are accepted.

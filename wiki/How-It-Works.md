# How Polygeo Works

Polygeo separates country data from application code. The Go binary contains the generic manifest loader, registry, geometry engine, and HTTP API. Country definitions and geometry remain in the configured filesystem data directory.

## Process startup

`cmd/polygeo/main.go` performs these steps:

1. Read `POLYGEO_DATA_DIR`, defaulting to `./data`.
2. Read `POLYGEO_ADDR`, defaulting to `:8080`.
3. Call `geo.LoadDirectory` to construct the complete country registry.
4. Exit with status `1` when registry loading fails.
5. Create the Echo server with the loaded registry.
6. Start listening on the configured address.

The HTTP server does not start until registry loading succeeds.

## Country discovery

`geo.LoadDirectory` reads the immediate children of the configured data directory.

An entry is processed when all of these conditions are true:

- it is a directory;
- its name does not start with `.` or `_`;
- it contains `country.json`.

Files, ignored directories, and directories without `country.json` do not become registry entries.

For every discovered package, Polygeo:

1. Decodes `country.json` into a manifest.
2. Validates schema version, required fields, level names, aliases, file paths, and parent ordering.
3. Loads every administrative-level GeoJSON file named in the manifest.
4. Validates the feature collection and every feature.
5. Parses each `Polygon` or `MultiPolygon` into in-memory geometry.
6. Groups repeated features with the same normalized parent and region name into one region.
7. Sorts regions by parent name and then region name.
8. Registers country lookup identifiers.

Any error stops the complete startup. Polygeo does not serve a partially loaded registry.

## Country registry

Each loaded country is indexed by the case-insensitive normalized form of:

- manifest `code`;
- manifest `name`;
- country directory name;
- every manifest alias.

A lookup-name collision between two country packages is a startup error.

Countries returned by `GET /countries` are sorted by manifest code.

## Administrative levels

A country manifest contains an ordered `levels` array. Each level defines:

- a canonical ID;
- a plural route name;
- optional route aliases;
- one GeoJSON file;
- the property containing the region name;
- an optional parent level and parent property.

Level lookup accepts the ID, plural name, or any alias, using case-insensitive matching. All names and aliases within one manifest must be unique after normalization.

A child level can reference only an earlier level in the manifest. This ordering is also the coordinate-lookup order.

## GeoJSON loading

Each level file must be a GeoJSON `FeatureCollection`. Each feature must have:

- `type: "Feature"`;
- a non-empty string at the configured region-name property;
- a non-empty string at the configured parent property when applicable;
- a `Polygon` or `MultiPolygon` geometry.

For each geometry, Polygeo stores:

- polygon rings;
- all polygons in a MultiPolygon;
- one bounding box covering the geometry.

The first ring is the exterior ring. Remaining rings are holes.

## HTTP routing

`internal/httpapi/server.go` registers two equivalent route families:

```text
/countries/:country/...
/:country/...
```

Both route families call the same handlers. `/status` and `/countries` are registered as explicit service routes.

The country route set is:

```text
GET /:country
GET /:country/locate
GET /:country/regions
GET /:country/:level
GET /:country/:parentLevel/:parent/:childLevel
```

The `/countries` prefix is added to the same set for canonical routes.

## Region listing

Region-list endpoints resolve the country and level from the registry. A supplied parent name is compared case-insensitively against each region's loaded parent value.

Results use the deterministic startup sort order. A parent filter that matches no region returns `200` with an empty result list.

The nested route validates the requested relationship. For example:

```text
/countries/PH/provinces/Cebu/municipalities
```

is accepted only when the resolved municipality level declares the resolved province level as its parent.

## Coordinate lookup

The locate handler validates latitude and longitude before calling the country registry:

- latitude: `-90` through `90`;
- longitude: `-180` through `180`.

The country lookup algorithm is:

1. Create a point using longitude as X and latitude as Y.
2. Process administrative levels in manifest order.
3. For a child level, read the already matched parent name.
4. Skip the child level when its parent did not match.
5. Consider only child regions whose parent equals the matched parent.
6. Scan regions in deterministic startup sort order.
7. Check each geometry's bounding box.
8. Run point-in-polygon testing only when the point is within that bounding box.
9. Record the first matching region for the level.
10. Return every level that matched.

A coordinate matching no administrative level returns `404 location_not_found`. A coordinate can return a parent match without a child match.

When two regions at the same level overlap, Polygeo returns the first region in the deterministic startup sort order.

## Point-in-polygon behavior

Polygeo uses a ray-crossing point-in-polygon test.

- A point outside the geometry bounding box is rejected before polygon testing.
- A point inside an exterior ring is included.
- A point inside a hole is excluded.
- A point on an exterior boundary is included.
- A point on a hole boundary is included.
- A MultiPolygon matches when any contained polygon matches.

## Concurrency model

All manifests, regions, and geometry are loaded before Echo starts. Request handlers only read the registry and geometry structures; they do not mutate loaded country data.

This makes concurrent HTTP reads independent of runtime data loading. File changes are not watched, and there is no in-process reload operation.

## Failure model

Startup fails for conditions including:

- unreadable data directory;
- malformed manifest JSON;
- unsupported manifest schema version;
- missing required manifest fields;
- duplicate level IDs, names, or aliases;
- invalid parent-level ordering;
- paths that escape a country directory;
- missing or unreadable GeoJSON files;
- non-`FeatureCollection` data;
- invalid features or required properties;
- unsupported geometry types;
- invalid rings or non-finite coordinates;
- country lookup-name collisions.

HTTP validation and lookup failures use the JSON error envelope documented in [API Reference](API-Reference).

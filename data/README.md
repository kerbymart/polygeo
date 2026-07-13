# Country data packages

Polygeo discovers countries from subdirectories in this directory. The data is read from the filesystem at startup and is not embedded into the Go binary.

Each country directory must contain a `country.json` manifest and every GeoJSON file referenced by that manifest. Directories whose names start with `.` or `_` are ignored.

Example:

```text
PH/
  country.json
  provinces.geojson
  municipalities.geojson
```

See the repository README for the complete manifest schema and API examples.

package geo

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDirectoryAndLocate(t *testing.T) {
	root := t.TempDir()
	countryDir := filepath.Join(root, "PH")
	if err := os.MkdirAll(countryDir, 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := `{
	  "schema_version": 1,
	  "code": "PH",
	  "name": "Philippines",
	  "aliases": ["philippines"],
	  "levels": [
	    {"id":"province","plural":"provinces","file":"provinces.geojson","name_property":"NAME_1"},
	    {"id":"municipality","plural":"municipalities","file":"municipalities.geojson","name_property":"NAME_2","parent_level":"province","parent_property":"NAME_1"}
	  ]
	}`
	provinces := `{
	  "type":"FeatureCollection",
	  "features":[{
	    "type":"Feature",
	    "properties":{"NAME_1":"Cebu"},
	    "geometry":{"type":"Polygon","coordinates":[[[0,0],[10,0],[10,10],[0,10],[0,0]]]}
	  }]
	}`
	municipalities := `{
	  "type":"FeatureCollection",
	  "features":[{
	    "type":"Feature",
	    "properties":{"NAME_1":"Cebu","NAME_2":"Cebu City"},
	    "geometry":{"type":"Polygon","coordinates":[[[1,1],[5,1],[5,5],[1,5],[1,1]]]}
	  }]
	}`
	writeTestFile(t, filepath.Join(countryDir, "country.json"), manifest)
	writeTestFile(t, filepath.Join(countryDir, "provinces.geojson"), provinces)
	writeTestFile(t, filepath.Join(countryDir, "municipalities.geojson"), municipalities)

	registry, err := LoadDirectory(root)
	if err != nil {
		t.Fatalf("LoadDirectory() error = %v", err)
	}
	if got := registry.Count(); got != 1 {
		t.Fatalf("Count() = %d, want 1", got)
	}
	country, ok := registry.Country("philippines")
	if !ok {
		t.Fatal("Country(philippines) was not found")
	}
	regions, err := country.Regions("municipalities", "Cebu")
	if err != nil {
		t.Fatalf("Regions() error = %v", err)
	}
	if want := []string{"Cebu City"}; !reflect.DeepEqual(regions, want) {
		t.Fatalf("Regions() = %#v, want %#v", regions, want)
	}

	location, found := country.Locate(2, 2)
	if !found {
		t.Fatal("Locate() did not find a location")
	}
	if location.Regions["province"] != "Cebu" || location.Regions["municipality"] != "Cebu City" {
		t.Fatalf("Locate() regions = %#v", location.Regions)
	}
}

func TestManifestRejectsPathTraversal(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: 1,
		Code:          "XX",
		Name:          "Example",
		Levels: []LevelManifest{{
			ID:           "region",
			Plural:       "regions",
			File:         "../outside.geojson",
			NameProperty: "name",
		}},
	}
	if err := manifest.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want path traversal rejection")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

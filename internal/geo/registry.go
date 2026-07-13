package geo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const manifestFilename = "country.json"

type Region struct {
	Name       string
	Parent     string
	Geometries []geometry
}

type Level struct {
	Manifest LevelManifest
	Regions  []Region
}

type Country struct {
	Directory string
	Manifest  Manifest
	Levels    map[string]*Level
	levelKeys map[string]string
}

type CountrySummary struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type LevelSummary struct {
	ID          string   `json:"id"`
	Plural      string   `json:"plural"`
	Aliases     []string `json:"aliases,omitempty"`
	ParentLevel string   `json:"parent_level,omitempty"`
}

type CountryDescription struct {
	Code    string         `json:"code"`
	Name    string         `json:"name"`
	Aliases []string       `json:"aliases,omitempty"`
	Levels  []LevelSummary `json:"levels"`
}

type Location struct {
	Country   CountrySummary    `json:"country"`
	Latitude  float64           `json:"latitude"`
	Longitude float64           `json:"longitude"`
	Regions   map[string]string `json:"regions"`
}

type Registry struct {
	countries map[string]*Country
	ordered   []*Country
}

type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Geometry   rawGeometry    `json:"geometry"`
}

func LoadDirectory(root string) (*Registry, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read data directory %q: %w", root, err)
	}

	registry := &Registry{countries: make(map[string]*Country)}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		countryDir := filepath.Join(root, entry.Name())
		manifestPath := filepath.Join(countryDir, manifestFilename)
		if _, err := os.Stat(manifestPath); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("inspect %q: %w", manifestPath, err)
		}

		country, err := loadCountry(countryDir, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("load country directory %q: %w", entry.Name(), err)
		}
		if err := registry.addCountry(country); err != nil {
			return nil, err
		}
	}

	sort.Slice(registry.ordered, func(i, j int) bool {
		return registry.ordered[i].Manifest.Code < registry.ordered[j].Manifest.Code
	})
	return registry, nil
}

func loadCountry(directory, directoryName string) (*Country, error) {
	manifestBytes, err := os.ReadFile(filepath.Join(directory, manifestFilename))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("validate manifest: %w", err)
	}

	country := &Country{
		Directory: directoryName,
		Manifest:  manifest,
		Levels:    make(map[string]*Level, len(manifest.Levels)),
		levelKeys: make(map[string]string, len(manifest.Levels)*3),
	}
	for _, levelManifest := range manifest.Levels {
		level, err := loadLevel(directory, levelManifest)
		if err != nil {
			return nil, fmt.Errorf("load level %q: %w", levelManifest.ID, err)
		}
		canonical := normalizeLookup(levelManifest.ID)
		country.Levels[canonical] = level
		for _, name := range append([]string{levelManifest.ID, levelManifest.Plural}, levelManifest.Aliases...) {
			country.levelKeys[normalizeLookup(name)] = canonical
		}
	}
	return country, nil
}

func loadLevel(countryDirectory string, manifest LevelManifest) (*Level, error) {
	path := filepath.Join(countryDirectory, filepath.Clean(manifest.File))
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", manifest.File, err)
	}
	var collection featureCollection
	if err := json.Unmarshal(content, &collection); err != nil {
		return nil, fmt.Errorf("decode GeoJSON: %w", err)
	}
	if collection.Type != "FeatureCollection" {
		return nil, fmt.Errorf("expected FeatureCollection, got %q", collection.Type)
	}

	regionsByKey := make(map[string]*Region)
	orderedKeys := make([]string, 0, len(collection.Features))
	for i, candidate := range collection.Features {
		if candidate.Type != "Feature" {
			return nil, fmt.Errorf("feature %d has type %q", i, candidate.Type)
		}
		name, err := propertyString(candidate.Properties, manifest.NameProperty)
		if err != nil {
			return nil, fmt.Errorf("feature %d name: %w", i, err)
		}
		parent := ""
		if manifest.ParentProperty != "" {
			parent, err = propertyString(candidate.Properties, manifest.ParentProperty)
			if err != nil {
				return nil, fmt.Errorf("feature %d parent: %w", i, err)
			}
		}
		parsedGeometry, err := parseGeometry(candidate.Geometry)
		if err != nil {
			return nil, fmt.Errorf("feature %d geometry: %w", i, err)
		}

		key := normalizeLookup(parent) + "\x00" + normalizeLookup(name)
		region, exists := regionsByKey[key]
		if !exists {
			region = &Region{Name: name, Parent: parent}
			regionsByKey[key] = region
			orderedKeys = append(orderedKeys, key)
		}
		region.Geometries = append(region.Geometries, parsedGeometry)
	}

	regions := make([]Region, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		regions = append(regions, *regionsByKey[key])
	}
	sort.SliceStable(regions, func(i, j int) bool {
		if normalizeLookup(regions[i].Parent) == normalizeLookup(regions[j].Parent) {
			return strings.ToLower(regions[i].Name) < strings.ToLower(regions[j].Name)
		}
		return strings.ToLower(regions[i].Parent) < strings.ToLower(regions[j].Parent)
	})
	return &Level{Manifest: manifest, Regions: regions}, nil
}

func propertyString(properties map[string]any, property string) (string, error) {
	value, exists := properties[property]
	if !exists {
		return "", fmt.Errorf("property %q is missing", property)
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("property %q must be a non-empty string", property)
	}
	return text, nil
}

func (r *Registry) addCountry(country *Country) error {
	keys := append([]string{country.Manifest.Code, country.Manifest.Name, country.Directory}, country.Manifest.Aliases...)
	for _, key := range keys {
		normalized := normalizeLookup(key)
		if existing, exists := r.countries[normalized]; exists && existing != country {
			return fmt.Errorf("country lookup name %q conflicts between %s and %s", key, existing.Manifest.Code, country.Manifest.Code)
		}
		r.countries[normalized] = country
	}
	r.ordered = append(r.ordered, country)
	return nil
}

func (r *Registry) Country(codeOrAlias string) (*Country, bool) {
	country, ok := r.countries[normalizeLookup(codeOrAlias)]
	return country, ok
}

func (r *Registry) Countries() []CountrySummary {
	result := make([]CountrySummary, 0, len(r.ordered))
	for _, country := range r.ordered {
		result = append(result, CountrySummary{Code: country.Manifest.Code, Name: country.Manifest.Name})
	}
	return result
}

func (r *Registry) Count() int {
	return len(r.ordered)
}

func (c *Country) Description() CountryDescription {
	levels := make([]LevelSummary, 0, len(c.Manifest.Levels))
	for _, level := range c.Manifest.Levels {
		levels = append(levels, LevelSummary{
			ID:          level.ID,
			Plural:      level.Plural,
			Aliases:     append([]string(nil), level.Aliases...),
			ParentLevel: level.ParentLevel,
		})
	}
	return CountryDescription{
		Code:    c.Manifest.Code,
		Name:    c.Manifest.Name,
		Aliases: append([]string(nil), c.Manifest.Aliases...),
		Levels:  levels,
	}
}

func (c *Country) ResolveLevel(name string) (*Level, bool) {
	canonical, exists := c.levelKeys[normalizeLookup(name)]
	if !exists {
		return nil, false
	}
	level, exists := c.Levels[canonical]
	return level, exists
}

func (c *Country) Regions(levelName, parent string) ([]string, error) {
	level, exists := c.ResolveLevel(levelName)
	if !exists {
		return nil, fmt.Errorf("unknown administrative level %q", levelName)
	}
	result := make([]string, 0, len(level.Regions))
	for _, region := range level.Regions {
		if parent != "" && !strings.EqualFold(region.Parent, parent) {
			continue
		}
		result = append(result, region.Name)
	}
	return result, nil
}

func (c *Country) Locate(longitude, latitude float64) (Location, bool) {
	candidate := point{X: longitude, Y: latitude}
	matches := make(map[string]string, len(c.Manifest.Levels))
	for _, levelManifest := range c.Manifest.Levels {
		level := c.Levels[normalizeLookup(levelManifest.ID)]
		expectedParent := ""
		if levelManifest.ParentLevel != "" {
			expectedParent = matches[normalizeLookup(levelManifest.ParentLevel)]
			if expectedParent == "" {
				continue
			}
		}
		for _, region := range level.Regions {
			if expectedParent != "" && !strings.EqualFold(region.Parent, expectedParent) {
				continue
			}
			for _, regionGeometry := range region.Geometries {
				if regionGeometry.contains(candidate) {
					matches[normalizeLookup(levelManifest.ID)] = region.Name
					break
				}
			}
			if matches[normalizeLookup(levelManifest.ID)] != "" {
				break
			}
		}
	}

	if len(matches) == 0 {
		return Location{}, false
	}
	regions := make(map[string]string, len(matches))
	for _, level := range c.Manifest.Levels {
		if match := matches[normalizeLookup(level.ID)]; match != "" {
			regions[level.ID] = match
		}
	}
	return Location{
		Country:   CountrySummary{Code: c.Manifest.Code, Name: c.Manifest.Name},
		Latitude:  latitude,
		Longitude: longitude,
		Regions:   regions,
	}, true
}

func normalizeLookup(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

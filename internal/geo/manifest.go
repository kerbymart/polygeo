package geo

import (
	"fmt"
	"path/filepath"
	"strings"
)

const ManifestSchemaVersion = 1

// Manifest describes one installable country data package.
type Manifest struct {
	SchemaVersion int             `json:"schema_version"`
	Code          string          `json:"code"`
	Name          string          `json:"name"`
	Aliases       []string        `json:"aliases,omitempty"`
	Levels        []LevelManifest `json:"levels"`
}

// LevelManifest maps one administrative level to a GeoJSON file and its
// identifying properties.
type LevelManifest struct {
	ID             string   `json:"id"`
	Plural         string   `json:"plural"`
	Aliases        []string `json:"aliases,omitempty"`
	File           string   `json:"file"`
	NameProperty   string   `json:"name_property"`
	ParentLevel    string   `json:"parent_level,omitempty"`
	ParentProperty string   `json:"parent_property,omitempty"`
}

func (m Manifest) Validate() error {
	if m.SchemaVersion != ManifestSchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", m.SchemaVersion)
	}
	if strings.TrimSpace(m.Code) == "" {
		return fmt.Errorf("code is required")
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if len(m.Levels) == 0 {
		return fmt.Errorf("at least one administrative level is required")
	}

	knownLevels := make(map[string]struct{}, len(m.Levels))
	knownNames := make(map[string]struct{}, len(m.Levels)*2)
	for i, level := range m.Levels {
		if err := level.validate(); err != nil {
			return fmt.Errorf("level %d: %w", i, err)
		}
		id := normalizeLookup(level.ID)
		if _, exists := knownLevels[id]; exists {
			return fmt.Errorf("duplicate level id %q", level.ID)
		}
		knownLevels[id] = struct{}{}

		for _, name := range append([]string{level.ID, level.Plural}, level.Aliases...) {
			key := normalizeLookup(name)
			if _, exists := knownNames[key]; exists {
				return fmt.Errorf("duplicate level name or alias %q", name)
			}
			knownNames[key] = struct{}{}
		}

		if level.ParentLevel != "" {
			if _, exists := knownLevels[normalizeLookup(level.ParentLevel)]; !exists {
				return fmt.Errorf("level %q references unknown or later parent level %q", level.ID, level.ParentLevel)
			}
		}
	}
	return nil
}

func (l LevelManifest) validate() error {
	if strings.TrimSpace(l.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(l.Plural) == "" {
		return fmt.Errorf("plural is required")
	}
	if strings.TrimSpace(l.File) == "" {
		return fmt.Errorf("file is required")
	}
	if filepath.IsAbs(l.File) || strings.Contains(filepath.ToSlash(filepath.Clean(l.File)), "../") || filepath.Clean(l.File) == ".." {
		return fmt.Errorf("file must stay within the country directory")
	}
	if strings.TrimSpace(l.NameProperty) == "" {
		return fmt.Errorf("name_property is required")
	}
	if (l.ParentLevel == "") != (l.ParentProperty == "") {
		return fmt.Errorf("parent_level and parent_property must be set together")
	}
	return nil
}

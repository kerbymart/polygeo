package geo

import (
	"encoding/json"
	"fmt"
	"math"
)

type point struct {
	X float64
	Y float64
}

type bounds struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

type ring []point
type polygon []ring

type geometry struct {
	Polygons []polygon
	Bounds   bounds
}

type rawGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

func parseGeometry(raw rawGeometry) (geometry, error) {
	var polygons []polygon
	switch raw.Type {
	case "Polygon":
		var coords [][][]float64
		if err := json.Unmarshal(raw.Coordinates, &coords); err != nil {
			return geometry{}, fmt.Errorf("decode Polygon coordinates: %w", err)
		}
		p, err := convertPolygon(coords)
		if err != nil {
			return geometry{}, err
		}
		polygons = []polygon{p}
	case "MultiPolygon":
		var coords [][][][]float64
		if err := json.Unmarshal(raw.Coordinates, &coords); err != nil {
			return geometry{}, fmt.Errorf("decode MultiPolygon coordinates: %w", err)
		}
		polygons = make([]polygon, 0, len(coords))
		for i, candidate := range coords {
			p, err := convertPolygon(candidate)
			if err != nil {
				return geometry{}, fmt.Errorf("polygon %d: %w", i, err)
			}
			polygons = append(polygons, p)
		}
	default:
		return geometry{}, fmt.Errorf("unsupported geometry type %q", raw.Type)
	}

	g := geometry{Polygons: polygons}
	g.Bounds = calculateBounds(polygons)
	return g, nil
}

func convertPolygon(coords [][][]float64) (polygon, error) {
	if len(coords) == 0 {
		return nil, fmt.Errorf("polygon has no rings")
	}
	result := make(polygon, 0, len(coords))
	for ringIndex, rawRing := range coords {
		if len(rawRing) < 4 {
			return nil, fmt.Errorf("ring %d has fewer than four positions", ringIndex)
		}
		converted := make(ring, 0, len(rawRing))
		for positionIndex, rawPoint := range rawRing {
			if len(rawPoint) < 2 {
				return nil, fmt.Errorf("ring %d position %d has fewer than two coordinates", ringIndex, positionIndex)
			}
			if math.IsNaN(rawPoint[0]) || math.IsNaN(rawPoint[1]) || math.IsInf(rawPoint[0], 0) || math.IsInf(rawPoint[1], 0) {
				return nil, fmt.Errorf("ring %d position %d contains a non-finite coordinate", ringIndex, positionIndex)
			}
			converted = append(converted, point{X: rawPoint[0], Y: rawPoint[1]})
		}
		result = append(result, converted)
	}
	return result, nil
}

func calculateBounds(polygons []polygon) bounds {
	b := bounds{
		MinX: math.Inf(1),
		MinY: math.Inf(1),
		MaxX: math.Inf(-1),
		MaxY: math.Inf(-1),
	}
	for _, p := range polygons {
		for _, r := range p {
			for _, candidate := range r {
				b.MinX = math.Min(b.MinX, candidate.X)
				b.MinY = math.Min(b.MinY, candidate.Y)
				b.MaxX = math.Max(b.MaxX, candidate.X)
				b.MaxY = math.Max(b.MaxY, candidate.Y)
			}
		}
	}
	return b
}

func (g geometry) contains(candidate point) bool {
	if candidate.X < g.Bounds.MinX || candidate.X > g.Bounds.MaxX || candidate.Y < g.Bounds.MinY || candidate.Y > g.Bounds.MaxY {
		return false
	}
	for _, p := range g.Polygons {
		if polygonContains(p, candidate) {
			return true
		}
	}
	return false
}

func polygonContains(p polygon, candidate point) bool {
	if len(p) == 0 {
		return false
	}
	inside, boundary := ringContains(p[0], candidate)
	if boundary {
		return true
	}
	if !inside {
		return false
	}
	for _, hole := range p[1:] {
		insideHole, onHoleBoundary := ringContains(hole, candidate)
		if onHoleBoundary {
			return true
		}
		if insideHole {
			return false
		}
	}
	return true
}

func ringContains(r ring, candidate point) (inside bool, boundary bool) {
	if len(r) < 3 {
		return false, false
	}
	j := len(r) - 1
	for i := 0; i < len(r); i++ {
		a := r[j]
		b := r[i]
		if pointOnSegment(candidate, a, b) {
			return false, true
		}
		intersects := ((b.Y > candidate.Y) != (a.Y > candidate.Y)) &&
			(candidate.X < (a.X-b.X)*(candidate.Y-b.Y)/(a.Y-b.Y)+b.X)
		if intersects {
			inside = !inside
		}
		j = i
	}
	return inside, false
}

func pointOnSegment(candidate, a, b point) bool {
	const epsilon = 1e-10
	cross := (candidate.Y-a.Y)*(b.X-a.X) - (candidate.X-a.X)*(b.Y-a.Y)
	if math.Abs(cross) > epsilon {
		return false
	}
	lengthSquared := (b.X-a.X)*(b.X-a.X) + (b.Y-a.Y)*(b.Y-a.Y)
	if lengthSquared <= epsilon {
		dx := candidate.X - a.X
		dy := candidate.Y - a.Y
		return dx*dx+dy*dy <= epsilon
	}
	dot := (candidate.X-a.X)*(b.X-a.X) + (candidate.Y-a.Y)*(b.Y-a.Y)
	if dot < -epsilon {
		return false
	}
	return dot <= lengthSquared+epsilon
}

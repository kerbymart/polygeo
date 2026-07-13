package geo

import (
	"encoding/json"
	"testing"
)

func TestPolygonContainsRespectsHoles(t *testing.T) {
	raw := rawGeometry{
		Type: "Polygon",
		Coordinates: json.RawMessage(`[
			[[0,0],[10,0],[10,10],[0,10],[0,0]],
			[[3,3],[7,3],[7,7],[3,7],[3,3]]
		]`),
	}
	g, err := parseGeometry(raw)
	if err != nil {
		t.Fatalf("parseGeometry() error = %v", err)
	}

	tests := []struct {
		name string
		p    point
		want bool
	}{
		{name: "inside exterior", p: point{X: 1, Y: 1}, want: true},
		{name: "inside hole", p: point{X: 5, Y: 5}, want: false},
		{name: "outside", p: point{X: 11, Y: 5}, want: false},
		{name: "outer boundary", p: point{X: 0, Y: 5}, want: true},
		{name: "hole boundary", p: point{X: 3, Y: 5}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := g.contains(tt.p); got != tt.want {
				t.Fatalf("contains(%v) = %v, want %v", tt.p, got, tt.want)
			}
		})
	}
}

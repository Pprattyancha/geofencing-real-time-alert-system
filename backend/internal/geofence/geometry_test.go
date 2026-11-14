package geofence

import (
	"github.com/Pprattyancha/geofencing-real-time-alert-system/backend/internal/models"
	"testing"
)

func TestContains(t *testing.T) {
	p := []models.Coordinate{{37, -122}, {38, -122}, {38, -121}, {37, -121}, {37, -122}}
	if !Contains(p, 37.5, -121.5) {
		t.Fatal("expected inside")
	}
	if Contains(p, 39, -121.5) {
		t.Fatal("expected outside")
	}
}
func TestValidatePolygon(t *testing.T) {
	p := []models.Coordinate{{1, 1}, {1, 2}, {2, 2}, {1, 1}}
	if err := ValidatePolygon(p); err != nil {
		t.Fatalf("valid triangle: %v", err)
	}
	p[3] = models.Coordinate{3, 3}
	if ValidatePolygon(p) == nil {
		t.Fatal("expected open polygon to fail")
	}
}

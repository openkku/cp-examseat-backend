package views

import (
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

func TestRoomPayloadLocation(t *testing.T) {
	lat, lng := 16.1, 102.2
	with := RoomPayload(models.Room{Title: "T", Description: "D", Lat: &lat, Lng: &lng}, false)
	if with["title"] != "T" || with["description"] != "D" || with["lat"] != 16.1 || with["lng"] != 102.2 {
		t.Errorf("payload = %v", with)
	}
	without := RoomPayload(models.Room{Lat: &lat}, false)
	for _, k := range []string{"title", "description", "lat", "lng"} {
		if _, ok := without[k]; ok {
			t.Errorf("%q must be omitted when unset (lat without lng is incomplete): %v", k, without)
		}
	}
}

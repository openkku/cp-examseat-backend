package presenter

import (
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/room"
)

func TestRoomPayloadLocation(t *testing.T) {
	lat, lng := 16.1, 102.2
	with := RoomPayload(room.Room{Title: "T", Description: "D", Lat: &lat, Lng: &lng}, false)
	if with["title"] != "T" || with["description"] != "D" || with["lat"] != 16.1 || with["lng"] != 102.2 {
		t.Errorf("payload = %v", with)
	}
	without := RoomPayload(room.Room{Lat: &lat}, false)
	for _, k := range []string{"title", "description", "lat", "lng"} {
		if _, ok := without[k]; ok {
			t.Errorf("%q must be omitted when unset (lat without lng is incomplete): %v", k, without)
		}
	}
}

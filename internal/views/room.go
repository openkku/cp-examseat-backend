package views

import "github.com/openkku/cp-examseat-backend/internal/models"

// RoomPayload is the JSON representation of one room: its image links and,
// when withLayout is set, the seating layout with its front/back labels.
func RoomPayload(room models.Room, withLayout bool) map[string]any {
	images := room.Images
	if images == nil {
		images = []string{}
	}
	payload := map[string]any{
		"i_layout": room.LayoutImage,
		"i_map":    room.MapURL,
		"i_images": images,
	}
	if room.Title != "" {
		payload["title"] = room.Title
	}
	if room.Description != "" {
		payload["description"] = room.Description
	}
	if room.Lat != nil && room.Lng != nil {
		payload["lat"] = *room.Lat
		payload["lng"] = *room.Lng
	}
	if withLayout {
		payload["layout"] = room.Layout
		for key, value := range room.Labels {
			payload[key] = value
		}
	}
	return payload
}

// RoomsPayload renders a set of rooms keyed by room name.
func RoomsPayload(rooms map[string]models.Room, withLayout bool) map[string]map[string]any {
	out := make(map[string]map[string]any, len(rooms))
	for name, room := range rooms {
		out[name] = RoomPayload(room, withLayout)
	}
	return out
}

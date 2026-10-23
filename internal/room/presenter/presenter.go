// Package presenter renders rooms as the /api/room JSON payload.
package presenter

import "github.com/openkku/cp-examseat-backend/internal/room"

// RoomPayload is the JSON representation of one room: its image links and,
// when withLayout is set, the seating layout with its front/back labels.
func RoomPayload(r room.Room, withLayout bool) map[string]any {
	images := r.Images
	if images == nil {
		images = []string{}
	}
	payload := map[string]any{
		"i_layout": r.LayoutImage,
		"i_map":    r.MapURL,
		"i_images": images,
	}
	if r.Title != "" {
		payload["title"] = r.Title
	}
	if r.Description != "" {
		payload["description"] = r.Description
	}
	if r.Lat != nil && r.Lng != nil {
		payload["lat"] = *r.Lat
		payload["lng"] = *r.Lng
	}
	if withLayout {
		payload["layout"] = r.Layout
		for key, value := range r.Labels {
			payload[key] = value
		}
	}
	return payload
}

// RoomsPayload renders a set of rooms keyed by room name.
func RoomsPayload(rooms map[string]room.Room, withLayout bool) map[string]map[string]any {
	out := make(map[string]map[string]any, len(rooms))
	for name, r := range rooms {
		out[name] = RoomPayload(r, withLayout)
	}
	return out
}

package models

// RoomMeta is one entry of room/metadata.json.
type RoomMeta struct {
	LayoutFile  string   `json:"layout_file"`
	LayoutImage string   `json:"layout_image"`
	MapURL      string   `json:"map_url"`
	Images      []string `json:"images"`

	// Optional display details shown on the room page and campus map.
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Lat         *float64 `json:"lat,omitempty"`
	Lng         *float64 `json:"lng,omitempty"`
}

// Room is an exam room whose seating layout was loaded successfully.
type Room struct {
	Name        string
	LayoutImage string
	MapURL      string
	Images      []string
	// Layout is the raw "layout" block of the room's map file.
	Layout any
	// Labels holds the optional "frontLabel"/"backLabel" entries of the map file.
	Labels map[string]any

	Title       string
	Description string
	Lat, Lng    *float64
}

// RoomCatalog is every room configured in room/metadata.json.
type RoomCatalog struct {
	// Rooms holds the rooms whose layout file loaded, keyed by room name.
	Rooms map[string]Room
	// Configured is the number of entries in metadata.json, including rooms
	// whose layout file is missing or invalid.
	Configured int
}

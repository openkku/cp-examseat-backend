package roomconfig

import "errors"

// ErrRoomNotFound is returned when deleting a room absent from the metadata.
var ErrRoomNotFound = errors.New("room not found")

// ErrMetadataNotFound is returned when the metadata does not exist yet.
var ErrMetadataNotFound = errors.New("metadata config file not found")

// ErrInvalidLayout is returned by SaveLayout for a body that is not a JSON object.
var ErrInvalidLayout = errors.New("invalid layout JSON")

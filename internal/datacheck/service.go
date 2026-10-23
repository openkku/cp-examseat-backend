package datacheck

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/openkku/cp-examseat-backend/internal/roomconfig"
)

// Service runs health checks across round usage, rooms, layouts and images.
type Service struct {
	usage  Usage
	rooms  Rooms
	images Images
}

// NewService creates a datacheck Service.
func NewService(usage Usage, rooms Rooms, images Images) *Service {
	return &Service{
		usage:  usage,
		rooms:  rooms,
		images: images,
	}
}

// Run executes all health checks and returns a Report.
func (s *Service) Run(ctx context.Context) (Report, error) {
	roomSeats, err := s.usage.RoomSeats(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("failed to query room seats: %w", err)
	}

	metaMap, err := s.rooms.Rooms()
	if err != nil {
		return Report{}, fmt.Errorf("failed to load room metadata: %w", err)
	}

	var problems []Problem

	// 1. A room used in a round with no metadata entry.
	for _, rs := range roomSeats {
		if _, ok := metaMap[rs.Room]; !ok {
			problems = append(problems, Problem{
				Kind:     KindMissingMetadata,
				Severity: "error",
				Room:     rs.Room,
				Round:    rs.Round,
				Message:  fmt.Sprintf("Room %q is used in round %s but has no metadata entry", rs.Room, rs.Round),
			})
		}
	}

	// Prepare sorted room list from metadata
	roomNames := make([]string, 0, len(metaMap))
	for name := range metaMap {
		roomNames = append(roomNames, name)
	}
	slices.Sort(roomNames)

	// 2. A layout file that is missing or unparsable.
	parsedLayoutSeats := make(map[string]map[string]bool)

	for _, name := range roomNames {
		meta := metaMap[name]
		if meta.LayoutFile == "" {
			problems = append(problems, Problem{
				Kind:     KindInvalidLayout,
				Severity: "error",
				Room:     name,
				Message:  fmt.Sprintf("Room %s has no layout file configured", name),
			})
			continue
		}

		data, exists, err := s.rooms.RawLayout(meta.LayoutFile)
		if err != nil {
			problems = append(problems, Problem{
				Kind:     KindInvalidLayout,
				Severity: "error",
				Room:     name,
				Message:  fmt.Sprintf("Layout file %q for room %s could not be read: %v", meta.LayoutFile, name, err),
			})
			continue
		}
		if !exists {
			problems = append(problems, Problem{
				Kind:     KindInvalidLayout,
				Severity: "error",
				Room:     name,
				Message:  fmt.Sprintf("Layout file %q for room %s is missing", meta.LayoutFile, name),
			})
			continue
		}

		layout, err := roomconfig.ParseLayout(data)
		if err != nil {
			problems = append(problems, Problem{
				Kind:     KindInvalidLayout,
				Severity: "error",
				Room:     name,
				Message:  fmt.Sprintf("Layout file %q for room %s is unparsable: %v", meta.LayoutFile, name, err),
			})
			continue
		}

		seatSet := make(map[string]bool)
		for _, seatID := range layout.SeatIDs() {
			seatSet[seatID] = true
		}
		parsedLayoutSeats[name] = seatSet
	}

	// 3. Seats used in a round but missing from the layout (at most 10 listed per room).
	for _, rs := range roomSeats {
		layoutSeats, ok := parsedLayoutSeats[rs.Room]
		if !ok {
			continue
		}

		var missing []string
		for _, seat := range rs.Seats {
			if !layoutSeats[seat] {
				missing = append(missing, seat)
			}
		}

		if len(missing) > 0 {
			slices.Sort(missing)
			limit := 10
			if len(missing) < limit {
				limit = len(missing)
			}
			details := missing[:limit]
			problems = append(problems, Problem{
				Kind:     KindMissingSeats,
				Severity: "warning",
				Room:     rs.Room,
				Round:    rs.Round,
				Message: fmt.Sprintf("%d seats used in round %s missing from layout (%s)",
					len(missing), rs.Round, strings.Join(details, ", ")),
				Details: details,
			})
		}
	}

	// 4. A missing layout_image or images file.
	if s.images != nil {
		for _, name := range roomNames {
			meta := metaMap[name]
			if isLocalImage(meta.LayoutImage) && !s.images.Exists(meta.LayoutImage) {
				problems = append(problems, Problem{
					Kind:     KindMissingImage,
					Severity: "warning",
					Room:     name,
					Message:  fmt.Sprintf("Layout image %q for room %s not found", meta.LayoutImage, name),
				})
			}
			for _, img := range meta.Images {
				if isLocalImage(img) && !s.images.Exists(img) {
					problems = append(problems, Problem{
						Kind:     KindMissingImage,
						Severity: "warning",
						Room:     name,
						Message:  fmt.Sprintf("Room image %q for room %s not found", img, name),
					})
				}
			}
		}
	}

	// 5. Info: a room with no seats in any round.
	usedRooms := make(map[string]bool)
	for _, rs := range roomSeats {
		usedRooms[rs.Room] = true
	}

	for _, name := range roomNames {
		if !usedRooms[name] {
			problems = append(problems, Problem{
				Kind:     KindUnusedRoom,
				Severity: "info",
				Room:     name,
				Message:  fmt.Sprintf("Room %s has no seats in any round", name),
			})
		}
	}

	if problems == nil {
		problems = []Problem{}
	}

	return Report{Problems: problems}, nil
}

// isLocalImage reports whether path names a file in the room image folder,
// rather than an external URL (https://…, //cdn…), which is not checked.
func isLocalImage(path string) bool {
	return path != "" && !strings.Contains(path, "://") && !strings.HasPrefix(path, "//")
}

package admin

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// idPattern restricts round and dataset IDs to safe identifiers.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,64}$`)

// maxLabelLength bounds a round label.
const maxLabelLength = 200

// Service implements the admin use cases.
type Service struct {
	repo     Repository
	datasets DatasetPurger
	importer Importer
	reloader Reloader
}

// NewService creates a Service.
func NewService(repo Repository, datasets DatasetPurger, importer Importer, reloader Reloader) *Service {
	return &Service{repo: repo, datasets: datasets, importer: importer, reloader: reloader}
}

// Rounds lists every round with its seat and student counts, newest first.
func (s *Service) Rounds(ctx context.Context) ([]RoundSummary, error) {
	return s.repo.RoundSummaries(ctx)
}

// RenameRound sets the label of a round and returns the stored label.
func (s *Service) RenameRound(ctx context.Context, round, label string) (string, error) {
	label = strings.TrimSpace(label)
	if !idPattern.MatchString(round) || label == "" || len(label) > maxLabelLength {
		return "", invalid("A round ID and a label of at most 200 characters are required")
	}
	if err := s.repo.SetRoundLabel(ctx, round, label); err != nil {
		return "", err
	}
	return label, s.Reload(ctx)
}

// DeleteRound removes a whole round, or only its custom dataset customID when set.
func (s *Service) DeleteRound(ctx context.Context, round, customID string) error {
	if !idPattern.MatchString(round) || (customID != "" && !idPattern.MatchString(customID)) {
		return invalid("Invalid round or dataset ID")
	}
	// A failed check lets the delete itself report the database error.
	if exists, err := s.repo.RoundExists(ctx, round); err == nil && !exists {
		return domain.ErrRoundNotFound
	}

	var err error
	if customID != "" {
		err = s.datasets.PurgeCustomDataset(ctx, round, customID)
	} else {
		err = s.repo.DeleteRound(ctx, round)
	}
	if err != nil {
		return err
	}
	return s.Reload(ctx)
}

// ValidateImport checks the identifiers of an import before its file is read.
func (s *Service) ValidateImport(round, customID, roomLayout string) error {
	if !idPattern.MatchString(round) {
		return invalid("Round ID is required (letters, digits, _ and - only)")
	}
	if (customID != "" && !idPattern.MatchString(customID)) || (roomLayout != "" && !idPattern.MatchString(roomLayout)) {
		return invalid("Invalid custom ID or room layout")
	}
	return nil
}

// ValidateFileType checks that a file with extension ext (lower-case, dotted) can be imported.
func (s *Service) ValidateFileType(ext string) error {
	if !s.importer.SupportedExtension(ext) {
		return invalid("Unsupported file type (use .xlsx, .xls, .pdf or .json)")
	}
	return nil
}

// Import imports a seating file and returns the number of seats saved. An
// empty Display defaults to the round's derived label.
func (s *Service) Import(ctx context.Context, req ImportRequest) (int, error) {
	if err := s.ValidateImport(req.Round, req.CustomID, req.RoomLayout); err != nil {
		return 0, err
	}
	display := req.Display
	if display == "" {
		display = domain.DefaultRoundLabel(req.Round)
	}

	saved, err := s.importer.Ingest(ctx, ingest.Options{
		FilePath:    req.FilePath,
		RoundID:     req.Round,
		DisplayName: display,
		Labels:      req.Labels,
		RoomLayout:  req.RoomLayout,
		CustomID:    req.CustomID,
	})
	if err != nil {
		return 0, err
	}
	return saved, s.Reload(ctx)
}

// PreviewImport validates a seating file and previews its contents and replacement impact
// without committing changes to the database.
func (s *Service) PreviewImport(ctx context.Context, req ImportRequest) (ImportPreview, error) {
	if err := s.ValidateImport(req.Round, req.CustomID, req.RoomLayout); err != nil {
		return ImportPreview{}, err
	}
	display := req.Display
	if display == "" {
		display = domain.DefaultRoundLabel(req.Round)
	}

	seats, err := s.importer.Preview(ingest.Options{
		FilePath:    req.FilePath,
		RoundID:     req.Round,
		DisplayName: display,
		Labels:      req.Labels,
		RoomLayout:  req.RoomLayout,
		CustomID:    req.CustomID,
	})
	if err != nil {
		return ImportPreview{}, err
	}

	studentSet := make(map[string]bool)
	roomSet := make(map[string]bool)
	dateSet := make(map[string]bool)
	sessionSet := make(map[string]bool)
	customIDSet := make(map[string]bool)

	if req.CustomID != "" {
		customIDSet[req.CustomID] = true
	}

	for _, seat := range seats {
		if seat.StudentID != "" {
			studentSet[seat.StudentID] = true
		}
		if seat.Room != "" {
			roomSet[seat.Room] = true
		}
		if seat.Date != "" {
			dateSet[seat.Date] = true
		}
		if seat.CustomID != "" {
			customIDSet[seat.CustomID] = true
		}

		cat := seat.ResolveCategory()
		d := seat.Date
		tStart := seat.TimeStart
		tEnd := seat.TimeEnd
		if tStart == "" && seat.Time != "" {
			tStart, tEnd = domain.SplitTime(seat.Time)
		}
		sessionKey := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
			req.Round, cat, seat.CustomID, seat.Sheet, d, tStart, tEnd,
			seat.Room, seat.Subject, seat.Section, seat.Note, seat.RoomLayout)
		sessionSet[sessionKey] = true
	}

	rooms := make([]string, 0, len(roomSet))
	for r := range roomSet {
		rooms = append(rooms, r)
	}
	slices.Sort(rooms)

	dates := make([]string, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}
	slices.Sort(dates)

	customIDs := make([]string, 0, len(customIDSet))
	for cid := range customIDSet {
		customIDs = append(customIDs, cid)
	}
	slices.Sort(customIDs)

	var replaces []ReplaceTarget
	summaries, err := s.repo.RoundSummaries(ctx)
	if err == nil {
		var roundSummary *RoundSummary
		for i := range summaries {
			if summaries[i].ID == req.Round {
				roundSummary = &summaries[i]
				break
			}
		}

		if len(customIDs) > 0 {
			for _, cid := range customIDs {
				currentSeats := 0
				if roundSummary != nil {
					for _, cd := range roundSummary.CustomDatasets {
						if cd.ID == cid {
							currentSeats = cd.Seats
							break
						}
					}
				}
				replaces = append(replaces, ReplaceTarget{
					Type:  "custom_dataset",
					ID:    cid,
					Seats: currentSeats,
				})
			}
		} else {
			inSchedule := 0
			if roundSummary != nil {
				inSchedule = roundSummary.InScheduleSeats
			}
			replaces = append(replaces, ReplaceTarget{
				Type:  "in_schedule",
				ID:    req.Round,
				Seats: inSchedule,
			})
		}
	}

	sampleLimit := 20
	if len(seats) < sampleLimit {
		sampleLimit = len(seats)
	}
	rows := make([]PreviewRow, sampleLimit)
	for i, seat := range seats[:sampleLimit] {
		t := seat.Time
		if t == "" && seat.TimeStart != "" {
			t = seat.TimeStart
			if seat.TimeEnd != "" {
				t += "-" + seat.TimeEnd
			}
		}
		labels := seat.Labels
		if labels == nil {
			labels = []string{}
		}
		rows[i] = PreviewRow{
			Sheet: seat.Sheet, Date: seat.Date, Time: t, Room: seat.Room,
			Subject: seat.Subject, SubjectName: seat.SubjectName, Section: seat.Section,
			StudentID: seat.StudentID, Seat: seat.Seat, CustomID: seat.CustomID, Labels: labels,
		}
	}

	return ImportPreview{
		Seats:     len(seats),
		Students:  len(studentSet),
		Rooms:     rooms,
		Dates:     dates,
		Sessions:  len(sessionSet),
		CustomIDs: customIDs,
		Replaces:  replaces,
		Rows:      rows,
	}, nil
}

// Reload asks the server to reload; a failure is returned as *ReloadError.
func (s *Service) Reload(ctx context.Context) error {
	if err := s.reloader.Reload(ctx); err != nil {
		return &ReloadError{Err: err}
	}
	return nil
}

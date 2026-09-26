package models

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var timeSpaceRegex = regexp.MustCompile(`\s*-\s*`)

// backToBackSlots are the morning and afternoon slots of the official timetable.
var backToBackSlots = map[string]bool{"08.30-11.30": true, "13.00-16.00": true}

func normalizeTime(t string) string {
	t = strings.Trim(t, "'\" \t")
	if t == "" {
		return ""
	}
	t = timeSpaceRegex.ReplaceAllString(t, "-")
	if !strings.Contains(t, "-") {
		return ""
	}
	return t
}

func departmentOf(subject string) string {
	subject = strings.TrimSpace(strings.ToUpper(subject))
	if len(subject) < 2 {
		return "Other"
	}
	switch subject[:2] {
	case "CP":
		return "CP (Computer)"
	case "SC":
		return "SC (Science)"
	case "LI":
		return "LI (Language)"
	case "BS":
		return "BS (Basic Science)"
	case "EN":
		return "EN (Engineering)"
	case "HS":
		return "HS (Humanities)"
	case "TE":
		return "TE (Technology)"
	case "GE":
		return "GE (General Ed)"
	default:
		return "Other"
	}
}

// statAccumulator collects raw counts for one bucket before they are summarised.
type statAccumulator struct {
	uniqueStudents     map[string]bool // The Source of Truth for Headcount
	rooms              map[string]bool
	subjects           map[string]string
	subCounts          map[string]int
	timeslotCounts     map[string]int
	roomSeats          map[string]int
	roomDays           map[string]map[string]bool
	roomSubjects       map[string]map[string]bool
	departmentSeatings map[string]int
	departmentSubjects map[string]map[string]bool
	dayExamCounts      map[string]int
	dayStudents        map[string]map[string]bool
	dayRooms           map[string]map[string]bool
	studentDayTimes    map[string]map[string]map[string]bool
	totalSeatings      int
}

func newStatAccumulator() *statAccumulator {
	return &statAccumulator{
		uniqueStudents:     make(map[string]bool),
		rooms:              make(map[string]bool),
		subjects:           make(map[string]string),
		subCounts:          make(map[string]int),
		timeslotCounts:     make(map[string]int),
		roomSeats:          make(map[string]int),
		roomDays:           make(map[string]map[string]bool),
		roomSubjects:       make(map[string]map[string]bool),
		departmentSeatings: make(map[string]int),
		departmentSubjects: make(map[string]map[string]bool),
		dayExamCounts:      make(map[string]int),
		dayStudents:        make(map[string]map[string]bool),
		dayRooms:           make(map[string]map[string]bool),
		studentDayTimes:    make(map[string]map[string]map[string]bool),
	}
}

func addToSet(m map[string]map[string]bool, key, value string) {
	if m[key] == nil {
		m[key] = make(map[string]bool)
	}
	if value != "" {
		m[key][value] = true
	}
}

func (b *statAccumulator) add(seat Seat) {
	b.totalSeatings++

	studentID := strings.TrimSpace(seat.StudentID)
	if studentID != "" {
		b.uniqueStudents[studentID] = true
	}

	roomName := strings.TrimSpace(seat.Room)
	if roomName != "" {
		b.rooms[roomName] = true
		b.roomSeats[roomName]++
		addToSet(b.roomDays, roomName, seat.Date)
		addToSet(b.roomSubjects, roomName, seat.Subject)
	}

	if seat.Subject != "" {
		b.subjects[seat.Subject] = seat.SubjectName
		b.subCounts[seat.Subject]++

		dept := departmentOf(seat.Subject)
		b.departmentSeatings[dept]++
		addToSet(b.departmentSubjects, dept, seat.Subject)
	}

	normTime := normalizeTime(seat.Time)
	if normTime != "" {
		b.timeslotCounts[normTime]++
	}

	if seat.Date != "" {
		b.dayExamCounts[seat.Date]++
		addToSet(b.dayStudents, seat.Date, studentID)
		addToSet(b.dayRooms, seat.Date, roomName)
	}

	// Student date timeslot for back-to-back detection
	if studentID != "" && seat.Date != "" && normTime != "" {
		if b.studentDayTimes[studentID] == nil {
			b.studentDayTimes[studentID] = make(map[string]map[string]bool)
		}
		addToSet(b.studentDayTimes[studentID], seat.Date, normTime)
	}
}

func (b *statAccumulator) summarize() StatBucket {
	out := StatBucket{
		StudentCount:         len(b.uniqueStudents),
		RoomCount:            len(b.rooms),
		TopSubjects:          make([]SubjectStat, 0),
		YearDistribution:     make([]YearStat, 0),
		TimeslotDistribution: make([]TimeslotStat, 0),
		RoomUtilization:      make([]RoomStat, 0),
		DepartmentBreakdown:  make([]DepartmentStat, 0),
	}

	// A. Subjects (High -> Low)
	for code, count := range b.subCounts {
		name := b.subjects[code]
		if name == "" {
			name = code
		}
		out.TopSubjects = append(out.TopSubjects, SubjectStat{Code: code, Name: name, Count: count})
	}
	sort.Slice(out.TopSubjects, func(i, j int) bool {
		a, c := out.TopSubjects[i], out.TopSubjects[j]
		if a.Count != c.Count {
			return a.Count > c.Count
		}
		return a.Code < c.Code
	})

	// B. Year distribution from UNIQUE students only (Newest -> Oldest)
	yearCounts := make(map[string]int)
	for studentID := range b.uniqueStudents {
		s := strings.TrimLeft(studentID, "'\"")
		if len(s) >= 2 {
			yearCounts[s[:2]]++
		}
	}
	for y, count := range yearCounts {
		out.YearDistribution = append(out.YearDistribution, YearStat{Year: y, Count: count})
	}
	sort.Slice(out.YearDistribution, func(i, j int) bool {
		return out.YearDistribution[i].Year > out.YearDistribution[j].Year
	})

	// C. Timeslots
	for slot, count := range b.timeslotCounts {
		out.TimeslotDistribution = append(out.TimeslotDistribution, TimeslotStat{Time: slot, Count: count})
	}
	sort.Slice(out.TimeslotDistribution, func(i, j int) bool {
		return out.TimeslotDistribution[i].Time < out.TimeslotDistribution[j].Time
	})

	// D. Room utilization
	for r, count := range b.roomSeats {
		out.RoomUtilization = append(out.RoomUtilization, RoomStat{
			Room:       r,
			SeatCount:  count,
			DaysActive: len(b.roomDays[r]),
			Subjects:   len(b.roomSubjects[r]),
		})
	}
	sort.Slice(out.RoomUtilization, func(i, j int) bool {
		a, c := out.RoomUtilization[i], out.RoomUtilization[j]
		if a.SeatCount != c.SeatCount {
			return a.SeatCount > c.SeatCount
		}
		return a.Room < c.Room
	})

	// E. Department breakdown
	for d, count := range b.departmentSeatings {
		out.DepartmentBreakdown = append(out.DepartmentBreakdown, DepartmentStat{
			Department: d,
			Seatings:   count,
			Subjects:   len(b.departmentSubjects[d]),
		})
	}
	sort.Slice(out.DepartmentBreakdown, func(i, j int) bool {
		a, c := out.DepartmentBreakdown[i], out.DepartmentBreakdown[j]
		if a.Seatings != c.Seatings {
			return a.Seatings > c.Seatings
		}
		return a.Department < c.Department
	})

	// F. Peak day (deterministic date ordering for tie breaking)
	dates := make([]string, 0, len(b.dayExamCounts))
	for d := range b.dayExamCounts {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	var peakDate string
	var peakCount int
	for _, d := range dates {
		if count := b.dayExamCounts[d]; count > peakCount {
			peakCount = count
			peakDate = d
		}
	}
	if peakDate != "" {
		out.PeakDay = PeakDayStat{
			Date:     peakDate,
			Count:    peakCount,
			Students: len(b.dayStudents[peakDate]),
			Rooms:    len(b.dayRooms[peakDate]),
		}
	}

	// G. Back-to-back: students sitting both official slots on the same day
	for _, days := range b.studentDayTimes {
		for _, slots := range days {
			valid := 0
			for slot := range slots {
				if backToBackSlots[slot] {
					valid++
				}
			}
			if valid >= 2 {
				out.BackToBackCount++
			}
		}
	}

	// H. Averages and totals
	out.TotalSeatings = b.totalSeatings
	if len(b.uniqueStudents) > 0 {
		out.AvgExamsPerStudent = float64(b.totalSeatings) / float64(len(b.uniqueStudents))
	}

	return out
}

// BuildDashboard aggregates seats into per-round and global analytics.
// roundLabels maps round IDs to display labels; rounds without seats still get an (empty) bucket.
func BuildDashboard(seats []Seat, roundLabels map[string]string) *Dashboard {
	dashboard := &Dashboard{
		Options: []Round{{ID: GlobalStatsID, Label: "Global View (All Rounds)"}},
		Stats:   make(map[string]StatBucket),
	}

	buckets := map[string]*statAccumulator{GlobalStatsID: newStatAccumulator()}
	ensureRound := func(roundID string) {
		if _, exists := buckets[roundID]; exists {
			return
		}
		buckets[roundID] = newStatAccumulator()
		label := roundLabels[roundID]
		if label == "" {
			label = fmt.Sprintf("Round: %s", roundID)
		}
		dashboard.Options = append(dashboard.Options, Round{ID: roundID, Label: label})
	}

	for roundID := range roundLabels {
		if roundID != "" {
			ensureRound(roundID)
		}
	}

	for _, seat := range seats {
		roundID := seat.ExamRound
		if roundID == "" {
			roundID = seat.Sheet
		}
		if roundID == "" {
			continue
		}
		ensureRound(roundID)
		buckets[GlobalStatsID].add(seat)
		buckets[roundID].add(seat)
	}

	for id, bucket := range buckets {
		dashboard.Stats[id] = bucket.summarize()
	}

	// Sort the non-global options chronologically descending
	rounds := dashboard.Options[1:]
	sort.Slice(rounds, func(i, j int) bool {
		return CompareRounds(rounds[i].ID, rounds[j].ID)
	})

	return dashboard
}

package models

import "testing"

func TestBuildDashboard(t *testing.T) {
	seats := []Seat{
		{ExamRound: "mid_1_2569", StudentID: "6533801234", Date: "2026-09-01", Time: "08.30-11.30", Room: "CP.9127", Subject: "CP1001", SubjectName: "Intro"},
		{ExamRound: "mid_1_2569", StudentID: "6533801234", Date: "2026-09-01", Time: "13.00 - 16.00", Room: "SC.1101", Subject: "SC2002", SubjectName: "Physics"},
		{ExamRound: "mid_1_2569", StudentID: "6633801111", Date: "2026-09-02", Time: "08.30-11.30", Room: "CP.9127", Subject: "CP1001", SubjectName: "Intro"},
		{ExamRound: "final_2_2568", StudentID: "6533801234", Date: "2026-03-01", Time: "13.00", Room: "CP.9127", Subject: "GE100"},
		{Sheet: "legacy_sheet", StudentID: "6733800000", Room: "X"},
	}
	labels := map[string]string{"mid_1_2569": "กลางภาค 1/2569", "final_2_2568": "ปลายภาค 2/2568", "empty_1_2560": ""}

	d := BuildDashboard(seats, labels)

	wantOptions := []string{GlobalStatsID, "mid_1_2569", "final_2_2568", "empty_1_2560", "legacy_sheet"}
	if len(d.Options) != len(wantOptions) {
		t.Fatalf("options = %+v; want ids %v", d.Options, wantOptions)
	}
	for i, id := range wantOptions {
		if d.Options[i].ID != id {
			t.Errorf("option[%d] = %q; want %q", i, d.Options[i].ID, id)
		}
	}
	if d.Options[3].Label != "Round: empty_1_2560" {
		t.Errorf("unlabelled round label = %q", d.Options[3].Label)
	}

	global := d.Stats[GlobalStatsID]
	if global.StudentCount != 3 || global.TotalSeatings != 5 || global.RoomCount != 3 {
		t.Errorf("global = %+v", global)
	}

	mid := d.Stats["mid_1_2569"]
	if mid.StudentCount != 2 || mid.RoomCount != 2 || mid.TotalSeatings != 3 {
		t.Errorf("mid counts = %+v", mid)
	}
	if mid.BackToBackCount != 1 {
		t.Errorf("mid back-to-back = %d; want 1", mid.BackToBackCount)
	}
	if mid.TopSubjects[0].Code != "CP1001" || mid.TopSubjects[0].Count != 2 {
		t.Errorf("mid top subject = %+v", mid.TopSubjects[0])
	}
	if mid.PeakDay.Date != "2026-09-01" || mid.PeakDay.Count != 2 || mid.PeakDay.Students != 1 || mid.PeakDay.Rooms != 2 {
		t.Errorf("mid peak day = %+v", mid.PeakDay)
	}
	if len(mid.YearDistribution) != 2 || mid.YearDistribution[0].Year != "66" {
		t.Errorf("mid years = %+v", mid.YearDistribution)
	}
	if mid.AvgExamsPerStudent != 1.5 {
		t.Errorf("mid avg = %v; want 1.5", mid.AvgExamsPerStudent)
	}
	if len(mid.TimeslotDistribution) != 2 || mid.TimeslotDistribution[1].Time != "13.00-16.00" {
		t.Errorf("timeslots should normalise spacing: %+v", mid.TimeslotDistribution)
	}

	final := d.Stats["final_2_2568"]
	if len(final.TimeslotDistribution) != 0 {
		t.Errorf("open-ended times are not timeslots: %+v", final.TimeslotDistribution)
	}
	if final.TopSubjects[0].Name != "GE100" || final.DepartmentBreakdown[0].Department != "GE (General Ed)" {
		t.Errorf("final subjects = %+v / %+v", final.TopSubjects, final.DepartmentBreakdown)
	}

	empty := d.Stats["empty_1_2560"]
	if empty.StudentCount != 0 || empty.TopSubjects == nil {
		t.Errorf("empty round bucket should be zero-valued with empty slices: %+v", empty)
	}
}

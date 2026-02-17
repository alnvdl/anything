package app_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/alnvdl/anything/internal/app"
)

// errorContains checks that err contains the substring want. If want is empty,
// it checks that err is nil.
func errorContains(err error, want string) bool {
	if want == "" {
		return err == nil
	}
	return err != nil && strings.Contains(err.Error(), want)
}

// testEntries returns a set of test entries.
func testEntries() []app.Entry {
	return []app.Entry{{
		Name:  "Pizza Place",
		Group: "Downtown",
		Open: map[string][]string{
			"mon": {"lunch", "dinner"},
			"tue": {"lunch"},
			"wed": {"lunch", "dinner"},
		},
		Cost: 2,
	}, {
		Name:  "Burger Joint",
		Group: "Downtown",
		Open: map[string][]string{
			"mon": {"lunch", "dinner"},
			"tue": {"lunch", "dinner"},
		},
		Cost: 1,
	}, {
		Name:  "Sushi Bar",
		Group: "Uptown",
		Open: map[string][]string{
			"mon": {"dinner"},
			"fri": {"lunch", "dinner"},
		},
		Cost: 4,
	}, {
		Name:  "Taco Stand",
		Group: "Uptown",
		Open: map[string][]string{
			"mon": {"breakfast", "lunch"},
			"tue": {"breakfast", "lunch"},
		},
		Cost: 1,
	}}
}

// testPeople returns test people config.
func testPeople() map[string]string {
	return map[string]string{
		"alice": "tokenA",
		"bob":   "tokenB",
	}
}

// testPeriods returns test periods config.
func testPeriods() app.Periods {
	return app.Periods{
		"breakfast": {0, 10},
		"lunch":     {10, 15},
		"dinner":    {15, 0},
	}
}

// newTestApp creates an App for testing. If entries are provided, they are
// used instead of the default testEntries().
func newTestApp(t *testing.T, entries ...app.Entry) *app.App {
	t.Helper()
	if len(entries) == 0 {
		entries = testEntries()
	}
	a, err := app.New(app.Params{
		Entries:  entries,
		People:   testPeople(),
		Timezone: time.UTC,
		Periods:  testPeriods(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestPersonForToken(t *testing.T) {
	a := newTestApp(t)

	var tests = []struct {
		desc   string
		token  string
		want   string
		wantOK bool
	}{{
		desc:   "valid token for alice",
		token:  "tokenA",
		want:   "alice",
		wantOK: true,
	}, {
		desc:   "valid token for bob",
		token:  "tokenB",
		want:   "bob",
		wantOK: true,
	}, {
		desc:   "invalid token",
		token:  "badtoken",
		want:   "",
		wantOK: false,
	}, {
		desc:   "empty token",
		token:  "",
		want:   "",
		wantOK: false,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got, ok := a.PersonForToken(test.token)
			if ok != test.wantOK {
				t.Fatalf("PersonForToken(%q) ok = %v, want %v", test.token, ok, test.wantOK)
			}
			if got != test.want {
				t.Errorf("PersonForToken(%q) = %q, want %q", test.token, got, test.want)
			}
		})
	}
}

func TestUpdateVotes(t *testing.T) {
	var tests = []struct {
		desc      string
		entries   []app.Entry
		person    string
		votes     map[string]string
		wantCount int
	}{{
		desc:   "valid votes",
		person: "alice",
		votes: map[string]string{
			"Downtown|Pizza Place":  "strong-yes",
			"Downtown|Burger Joint": "no",
		},
		wantCount: 2,
	}, {
		desc:   "cleans invalid entry names",
		person: "alice",
		votes: map[string]string{
			"Downtown|Pizza Place":  "yes",
			"Downtown|Nonexistent":  "yes",
			"Nowhere|Also Not Real": "no",
		},
		wantCount: 1,
	}, {
		desc:   "cleans invalid vote values",
		person: "bob",
		votes: map[string]string{
			"Downtown|Pizza Place":  "invalid-vote",
			"Downtown|Burger Joint": "yes",
		},
		wantCount: 1,
	}, {
		desc:      "empty votes",
		person:    "alice",
		votes:     map[string]string{},
		wantCount: 0,
	}, {
		desc:   "rejects entry with wrong group",
		person: "alice",
		votes: map[string]string{
			"Uptown|Pizza Place": "yes",
		},
		wantCount: 0,
	}, {
		desc:   "rejects entry without group separator",
		person: "alice",
		votes: map[string]string{
			"Pizza Place": "yes",
		},
		wantCount: 0,
	}, {
		desc: "accepts same entry name in different groups",
		entries: []app.Entry{{
			Name:  "Shared Name",
			Group: "GroupA",
			Open:  map[string][]string{"mon": {"lunch"}},
			Cost:  1,
		}, {
			Name:  "Shared Name",
			Group: "GroupB",
			Open:  map[string][]string{"mon": {"lunch"}},
			Cost:  2,
		}},
		person: "alice",
		votes: map[string]string{
			"GroupA|Shared Name": "strong-yes",
			"GroupB|Shared Name": "no",
		},
		wantCount: 2,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			a := newTestApp(t, test.entries...)
			a.UpdateVotes(test.person, test.votes)

			personVotes := a.Votes()[test.person]
			count := 0
			for _, gv := range personVotes {
				count += len(gv)
			}
			if count != test.wantCount {
				t.Errorf("UpdateVotes stored %d votes, want %d", count, test.wantCount)
			}
		})
	}
}

func TestUpdateVotesOverwrites(t *testing.T) {
	a := newTestApp(t)

	// First submission.
	a.UpdateVotes("alice", map[string]string{
		"Downtown|Pizza Place":  "strong-yes",
		"Downtown|Burger Joint": "no",
	})

	// Second submission overwrites.
	a.UpdateVotes("alice", map[string]string{
		"Uptown|Sushi Bar": "strong-no",
	})

	votes := a.Votes()["alice"]
	count := 0
	for _, gv := range votes {
		count += len(gv)
	}
	if count != 1 {
		t.Fatalf("expected 1 vote after overwrite, got %d", count)
	}
	if votes["Uptown"]["Sushi Bar"] != "strong-no" {
		t.Errorf("expected Sushi Bar vote to be strong-no, got %q", votes["Uptown"]["Sushi Bar"])
	}
	if dv, ok := votes["Downtown"]; ok {
		if _, ok := dv["Pizza Place"]; ok {
			t.Error("Pizza Place vote should have been removed on overwrite")
		}
	}
}

func TestVotePageData(t *testing.T) {
	a := newTestApp(t)
	a.UpdateVotes("alice", map[string]string{
		"Downtown|Pizza Place": "strong-yes",
		"Uptown|Sushi Bar":     "no",
	})

	groups := a.VotePageData("alice")

	// Should have 2 groups: Downtown and Uptown (alphabetical).
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Name != "Downtown" {
		t.Errorf("first group = %q, want Downtown", groups[0].Name)
	}
	if groups[1].Name != "Uptown" {
		t.Errorf("second group = %q, want Uptown", groups[1].Name)
	}

	// Downtown entries should be sorted alphabetically: Burger Joint, Pizza Place.
	if len(groups[0].Entries) != 2 {
		t.Fatalf("Downtown should have 2 entries, got %d", len(groups[0].Entries))
	}
	if groups[0].Entries[0].Name != "Burger Joint" {
		t.Errorf("first Downtown entry = %q, want Burger Joint", groups[0].Entries[0].Name)
	}
	if groups[0].Entries[1].Name != "Pizza Place" {
		t.Errorf("second Downtown entry = %q, want Pizza Place", groups[0].Entries[1].Name)
	}

	// Check current votes.
	if groups[0].Entries[1].CurrentVote != "strong-yes" {
		t.Errorf("Pizza Place vote = %q, want strong-yes", groups[0].Entries[1].CurrentVote)
	}

	// Uptown entries: Sushi Bar, Taco Stand.
	if groups[1].Entries[0].Name != "Sushi Bar" {
		t.Errorf("first Uptown entry = %q, want Sushi Bar", groups[1].Entries[0].Name)
	}
	if groups[1].Entries[0].CurrentVote != "no" {
		t.Errorf("Sushi Bar vote = %q, want no", groups[1].Entries[0].CurrentVote)
	}
}

func TestVotePageDataNoVotes(t *testing.T) {
	a := newTestApp(t)

	groups := a.VotePageData("alice")

	// All votes should be empty.
	for _, g := range groups {
		for _, e := range g.Entries {
			if e.CurrentVote != "" {
				t.Errorf("entry %q has vote %q, want empty", e.Name, e.CurrentVote)
			}
		}
	}
}

func TestSortGroupNames(t *testing.T) {
	var tests = []struct {
		desc       string
		names      []string
		groupOrder []string
		want       []string
	}{{
		desc:       "no group order sorts alphabetically",
		names:      []string{"Uptown", "Downtown", "Midtown"},
		groupOrder: nil,
		want:       []string{"Downtown", "Midtown", "Uptown"},
	}, {
		desc:       "full group order",
		names:      []string{"Uptown", "Downtown", "Midtown"},
		groupOrder: []string{"Midtown", "Uptown", "Downtown"},
		want:       []string{"Midtown", "Uptown", "Downtown"},
	}, {
		desc:       "partial group order puts unmatched at end alphabetically",
		names:      []string{"Uptown", "Downtown", "Midtown", "Suburbs"},
		groupOrder: []string{"Midtown"},
		want:       []string{"Midtown", "Downtown", "Suburbs", "Uptown"},
	}, {
		desc:       "group order with entries not in names is ignored",
		names:      []string{"Uptown", "Downtown"},
		groupOrder: []string{"Nonexistent", "Uptown", "Downtown"},
		want:       []string{"Uptown", "Downtown"},
	}, {
		desc:       "empty group order sorts alphabetically",
		names:      []string{"Uptown", "Downtown"},
		groupOrder: []string{},
		want:       []string{"Downtown", "Uptown"},
	}, {
		desc:       "single group",
		names:      []string{"Downtown"},
		groupOrder: []string{"Downtown"},
		want:       []string{"Downtown"},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			names := make([]string, len(test.names))
			copy(names, test.names)
			app.SortGroupNames(names, test.groupOrder)
			for i, got := range names {
				if got != test.want[i] {
					t.Errorf("names[%d] = %q, want %q (full result: %v)", i, got, test.want[i], names)
					break
				}
			}
		})
	}
}

func TestVotePageDataWithGroupOrder(t *testing.T) {
	a, err := app.New(app.Params{
		Entries:  testEntries(),
		People:   testPeople(),
		Timezone: time.UTC,
		Periods:  testPeriods(),
	})
	if err != nil {
		t.Fatal(err)
	}

	a.UpdateGroupOrder([]string{"Uptown", "Downtown"})

	groups := a.VotePageData("alice")

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Name != "Uptown" {
		t.Errorf("first group = %q, want Uptown", groups[0].Name)
	}
	if groups[1].Name != "Downtown" {
		t.Errorf("second group = %q, want Downtown", groups[1].Name)
	}
}

func TestTallyDataWithGroupOrder(t *testing.T) {
	a, err := app.New(app.Params{
		Entries:  testEntries(),
		People:   testPeople(),
		Timezone: time.UTC,
		Periods:  testPeriods(),
	})
	if err != nil {
		t.Fatal(err)
	}

	a.UpdateGroupOrder([]string{"Uptown", "Downtown"})

	groups := a.TallyData(time.Monday, "lunch")

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Name != "Uptown" {
		t.Errorf("first group = %q, want Uptown", groups[0].Name)
	}
	if groups[1].Name != "Downtown" {
		t.Errorf("second group = %q, want Downtown", groups[1].Name)
	}
}

func TestTallyData(t *testing.T) {
	a := newTestApp(t)

	// Alice votes.
	a.UpdateVotes("alice", map[string]string{
		"Downtown|Pizza Place":  "strong-yes", // 3.
		"Downtown|Burger Joint": "no",         // 1.
		"Uptown|Sushi Bar":      "strong-no",  // 0.
		"Uptown|Taco Stand":     "yes",        // 2.
	})

	// Bob votes.
	a.UpdateVotes("bob", map[string]string{
		"Downtown|Pizza Place":  "yes",        // 2.
		"Downtown|Burger Joint": "strong-yes", // 3.
		// Sushi Bar: no vote = yes (2).
		// Taco Stand: no vote = yes (2).
	})

	// Tally for Monday lunch.
	// Pizza Place: (3+2)*3 - 2 = 15 - 2 = 13, open.
	// Burger Joint: (1+3)*3 - 1 = 12 - 1 = 11, open.
	// Sushi Bar: (0+2)*3 - 4 = 6 - 4 = 2, closed (only dinner on mon).
	// Taco Stand: (2+2)*3 - 1 = 12 - 1 = 11, open.

	groups := a.TallyData(time.Monday, "lunch")

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// Downtown group: open entries sorted by score desc, then cost, then name.
	// Pizza Place: 13 (open), Burger Joint: 11 (open).
	dt := groups[0]
	if dt.Name != "Downtown" {
		t.Fatalf("first group = %q, want Downtown", dt.Name)
	}
	if len(dt.Entries) != 2 {
		t.Fatalf("Downtown should have 2 entries, got %d", len(dt.Entries))
	}
	if dt.Entries[0].Name != "Pizza Place" || dt.Entries[0].Score != 13 {
		t.Errorf("Downtown[0] = %q score %d, want Pizza Place score 13", dt.Entries[0].Name, dt.Entries[0].Score)
	}
	if dt.Entries[1].Name != "Burger Joint" || dt.Entries[1].Score != 11 {
		t.Errorf("Downtown[1] = %q score %d, want Burger Joint score 11", dt.Entries[1].Name, dt.Entries[1].Score)
	}
	if dt.Entries[0].Closed || dt.Entries[1].Closed {
		t.Error("Downtown entries should be open on Monday lunch")
	}

	// Uptown group: Taco Stand (11, open), Sushi Bar (2, closed).
	ut := groups[1]
	if ut.Name != "Uptown" {
		t.Fatalf("second group = %q, want Uptown", ut.Name)
	}
	if len(ut.Entries) != 2 {
		t.Fatalf("Uptown should have 2 entries, got %d", len(ut.Entries))
	}
	if ut.Entries[0].Name != "Taco Stand" || ut.Entries[0].Score != 11 {
		t.Errorf("Uptown[0] = %q score %d, want Taco Stand score 11", ut.Entries[0].Name, ut.Entries[0].Score)
	}
	if ut.Entries[0].Closed {
		t.Error("Taco Stand should be open on Monday lunch")
	}
	if ut.Entries[1].Name != "Sushi Bar" || ut.Entries[1].Score != 2 {
		t.Errorf("Uptown[1] = %q score %d, want Sushi Bar score 2", ut.Entries[1].Name, ut.Entries[1].Score)
	}
	if !ut.Entries[1].Closed {
		t.Error("Sushi Bar should be closed on Monday lunch")
	}
}

func TestTallyDataCostDisplay(t *testing.T) {
	a := newTestApp(t)
	groups := a.TallyData(time.Monday, "lunch")

	// Find Pizza Place (cost 2) and check display.
	for _, g := range groups {
		for _, e := range g.Entries {
			if e.Name == "Pizza Place" && e.CostDisplay != "$$" {
				t.Errorf("Pizza Place CostDisplay = %q, want $$", e.CostDisplay)
			}
			if e.Name == "Sushi Bar" && e.CostDisplay != "$$$$" {
				t.Errorf("Sushi Bar CostDisplay = %q, want $$$$", e.CostDisplay)
			}
			if e.Name == "Burger Joint" && e.CostDisplay != "$" {
				t.Errorf("Burger Joint CostDisplay = %q, want $", e.CostDisplay)
			}
		}
	}
}

func TestTallyDataSortingTiebreakers(t *testing.T) {
	// Create entries with same score to test tiebreakers.
	entries := []app.Entry{{
		Name:  "B Place",
		Group: "Group",
		Open:  map[string][]string{"mon": {"lunch"}},
		Cost:  2,
	}, {
		Name:  "A Place",
		Group: "Group",
		Open:  map[string][]string{"mon": {"lunch"}},
		Cost:  2,
	}, {
		Name:  "C Place",
		Group: "Group",
		Open:  map[string][]string{"mon": {"lunch"}},
		Cost:  1,
	}}

	a, err := app.New(app.Params{
		Entries:  entries,
		People:   map[string]string{"alice": "t1"},
		Timezone: time.UTC,
		Periods:  testPeriods(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// No votes = all default to yes (2).
	// All entries have score: 2*3 - cost.
	// B Place: 6-2=4, A Place: 6-2=4, C Place: 6-1=5.
	groups := a.TallyData(time.Monday, "lunch")

	if len(groups) != 1 || len(groups[0].Entries) != 3 {
		t.Fatalf("expected 1 group with 3 entries")
	}

	// C Place (score 5) first, then A Place (score 4, cost 2, name "A"), then B Place (score 4, cost 2, name "B").
	if groups[0].Entries[0].Name != "C Place" {
		t.Errorf("entry[0] = %q, want C Place", groups[0].Entries[0].Name)
	}
	if groups[0].Entries[1].Name != "A Place" {
		t.Errorf("entry[1] = %q, want A Place", groups[0].Entries[1].Name)
	}
	if groups[0].Entries[2].Name != "B Place" {
		t.Errorf("entry[2] = %q, want B Place", groups[0].Entries[2].Name)
	}
}

func TestTallyDataClosedAtEnd(t *testing.T) {
	// Create entries where a closed entry has a higher score than an open one.
	entries := []app.Entry{{
		Name:  "Open Low",
		Group: "Group",
		Open:  map[string][]string{"mon": {"lunch"}},
		Cost:  4,
	}, {
		Name:  "Closed High",
		Group: "Group",
		Open:  map[string][]string{"mon": {"dinner"}}, // Closed for lunch.
		Cost:  1,
	}}

	a, err := app.New(app.Params{
		Entries:  entries,
		People:   map[string]string{"alice": "t1"},
		Timezone: time.UTC,
		Periods:  testPeriods(),
	})
	if err != nil {
		t.Fatal(err)
	}

	groups := a.TallyData(time.Monday, "lunch")

	// Open Low should come first despite lower score.
	if groups[0].Entries[0].Name != "Open Low" {
		t.Errorf("entry[0] = %q, want Open Low (open entries first)", groups[0].Entries[0].Name)
	}
	if groups[0].Entries[1].Name != "Closed High" {
		t.Errorf("entry[1] = %q, want Closed High", groups[0].Entries[1].Name)
	}
	if !groups[0].Entries[1].Closed {
		t.Error("Closed High should be marked as closed")
	}
}

func TestTallyDataDefaultVotes(t *testing.T) {
	// Test that missing votes are interpreted as "yes" (2).
	entries := []app.Entry{{
		Name:  "Test Entry",
		Group: "Group",
		Open:  map[string][]string{"mon": {"lunch"}},
		Cost:  1,
	}}

	a, err := app.New(app.Params{
		Entries:  entries,
		People:   map[string]string{"alice": "t1", "bob": "t2", "carol": "t3"},
		Timezone: time.UTC,
		Periods:  testPeriods(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// No votes submitted. All 3 people default to yes (2).
	// Score = (2+2+2)*3 - 1 = 18 - 1 = 17.
	groups := a.TallyData(time.Monday, "lunch")

	if groups[0].Entries[0].Score != 17 {
		t.Errorf("score = %d, want 17", groups[0].Entries[0].Score)
	}
}

func TestPeriodForHour(t *testing.T) {
	periods := app.Periods{
		"breakfast": {0, 10},
		"lunch":     {10, 15},
		"dinner":    {15, 0},
	}

	var tests = []struct {
		desc string
		hour int
		want string
	}{{
		desc: "midnight is breakfast",
		hour: 0,
		want: "breakfast",
	}, {
		desc: "9am is breakfast",
		hour: 9,
		want: "breakfast",
	}, {
		desc: "10am is lunch",
		hour: 10,
		want: "lunch",
	}, {
		desc: "2pm is lunch",
		hour: 14,
		want: "lunch",
	}, {
		desc: "3pm is dinner",
		hour: 15,
		want: "dinner",
	}, {
		desc: "11pm is dinner",
		hour: 23,
		want: "dinner",
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := app.PeriodForHour(periods, test.hour)
			if got != test.want {
				t.Errorf("PeriodForHour(%d) = %q, want %q", test.hour, got, test.want)
			}
		})
	}
}

func TestPeriodForHourGap(t *testing.T) {
	// Periods with a gap at hour 12-17.
	periods := app.Periods{
		"morning": {6, 12},
		"evening": {18, 22},
	}

	got := app.PeriodForHour(periods, 14)
	if got != "" {
		t.Errorf("PeriodForHour(14) = %q, want empty (gap)", got)
	}
}

func TestPeriodTallyWeekday(t *testing.T) {
	a := newTestApp(t)

	// Reference: 2024-01-01 is a Monday in UTC.
	makeTime := func(weekday time.Weekday, hour int) time.Time {
		day := 1 + (int(weekday)-int(time.Monday)+7)%7
		return time.Date(2024, 1, day, hour, 0, 0, 0, time.UTC)
	}

	var tests = []struct {
		desc           string
		currentHour    int
		currentWeekday time.Weekday
		period         string
		want           time.Weekday
	}{{
		desc:           "requesting current period returns same day",
		currentHour:    12,
		currentWeekday: time.Monday,
		period:         "lunch",
		want:           time.Monday,
	}, {
		desc:           "requesting future period returns same day",
		currentHour:    12,
		currentWeekday: time.Monday,
		period:         "dinner",
		want:           time.Monday,
	}, {
		desc:           "requesting past period returns next day",
		currentHour:    12,
		currentWeekday: time.Monday,
		period:         "breakfast",
		want:           time.Tuesday,
	}, {
		desc:           "requesting past period on saturday returns sunday",
		currentHour:    20,
		currentWeekday: time.Saturday,
		period:         "lunch",
		want:           time.Sunday,
	}, {
		desc:           "requesting past period on saturday returns sunday for breakfast",
		currentHour:    20,
		currentWeekday: time.Saturday,
		period:         "breakfast",
		want:           time.Sunday,
	}, {
		desc:           "dinner on saturday night returns saturday",
		currentHour:    20,
		currentWeekday: time.Saturday,
		period:         "dinner",
		want:           time.Saturday,
	}, {
		desc:           "early morning requesting dinner returns same day",
		currentHour:    2,
		currentWeekday: time.Sunday,
		period:         "dinner",
		want:           time.Sunday,
	}, {
		desc:           "early morning requesting lunch returns same day",
		currentHour:    2,
		currentWeekday: time.Sunday,
		period:         "lunch",
		want:           time.Sunday,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			a.SetNowFunc(func() time.Time {
				return makeTime(test.currentWeekday, test.currentHour)
			})
			got := a.PeriodTallyWeekday(test.period)
			if got != test.want {
				t.Errorf("PeriodTallyWeekday(period=%q) with hour=%d, weekday=%v = %v, want %v",
					test.period, test.currentHour, test.currentWeekday, got, test.want)
			}
		})
	}
}

func TestPeriodTallyWeekdayWithGaps(t *testing.T) {
	// Periods with a gap: no period covers hours 12-17.
	a, err := app.New(app.Params{
		Entries:  testEntries(),
		People:   testPeople(),
		Timezone: time.UTC,
		Periods: app.Periods{
			"morning": {6, 12},
			"evening": {18, 22},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// In a gap, we can't determine the current period, so we return the same day.
	a.SetNowFunc(func() time.Time {
		return time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC) // Monday 14:00.
	})
	got := a.PeriodTallyWeekday("morning")
	if got != time.Monday {
		t.Errorf("PeriodTallyWeekday in gap = %v, want Monday", got)
	}
}

func TestSave(t *testing.T) {
	a := newTestApp(t)
	a.UpdateVotes("alice", map[string]string{
		"Downtown|Pizza Place":  "strong-yes",
		"Downtown|Burger Joint": "no",
	})
	a.UpdateVotes("bob", map[string]string{
		"Uptown|Sushi Bar": "yes",
	})
	a.UpdateGroupOrder([]string{"Uptown", "Downtown"})

	var buf bytes.Buffer
	if err := a.Save(&buf); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	saved := buf.String()
	for _, want := range []string{"entries", "votes", "alice", "bob", "Downtown", "Uptown", "Pizza Place", "strong-yes", "Burger Joint", "no", "Sushi Bar", "yes", "groupOrder"} {
		if !strings.Contains(saved, want) {
			t.Errorf("saved data does not contain %q", want)
		}
	}
}

func TestLoad(t *testing.T) {
	var tests = []struct {
		desc      string
		input     string
		wantVotes map[string]app.PersonVote
		wantErr   string
	}{{
		desc:  "valid data",
		input: `{"entries":[{"Name":"A","Group":"G","Open":{},"Cost":1}],"votes":{"alice":{"Downtown":{"Pizza Place":"strong-yes","Burger Joint":"no"}},"bob":{"Uptown":{"Sushi Bar":"yes"}}},"groupOrder":["Uptown","Downtown"]}`,
		wantVotes: map[string]app.PersonVote{
			"alice": {"Downtown": app.GroupVote{"Pizza Place": "strong-yes", "Burger Joint": "no"}},
			"bob":   {"Uptown": app.GroupVote{"Sushi Bar": "yes"}},
		},
	}, {
		desc:      "empty file",
		input:     "",
		wantVotes: map[string]app.PersonVote{},
	}, {
		desc:      "empty JSON object",
		input:     "{}",
		wantVotes: map[string]app.PersonVote{},
	}, {
		desc:    "invalid JSON",
		input:   "{not valid json",
		wantErr: "cannot deserialize data",
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			a := newTestApp(t)

			err := a.Load(strings.NewReader(test.input))
			if !errorContains(err, test.wantErr) {
				t.Fatalf("Load() err = %v, wantErr = %q", err, test.wantErr)
			}
			if test.wantErr != "" {
				return
			}

			votes := a.Votes()
			if len(votes) != len(test.wantVotes) {
				t.Fatalf("Load() resulted in %d people, want %d", len(votes), len(test.wantVotes))
			}
			for person, wantPV := range test.wantVotes {
				gotPV := votes[person]
				if len(gotPV) != len(wantPV) {
					t.Errorf("person %q has %d groups, want %d", person, len(gotPV), len(wantPV))
					continue
				}
				for group, wantGV := range wantPV {
					gotGV := gotPV[group]
					if len(gotGV) != len(wantGV) {
						t.Errorf("person %q group %q has %d votes, want %d", person, group, len(gotGV), len(wantGV))
						continue
					}
					for entry, wantVote := range wantGV {
						if gotGV[entry] != wantVote {
							t.Errorf("person %q group %q entry %q = %q, want %q", person, group, entry, gotGV[entry], wantVote)
						}
					}
				}
			}
		})
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	a := newTestApp(t)
	a.UpdateVotes("alice", map[string]string{
		"Downtown|Pizza Place":  "strong-yes",
		"Downtown|Burger Joint": "no",
	})
	a.UpdateVotes("bob", map[string]string{
		"Uptown|Sushi Bar":  "yes",
		"Uptown|Taco Stand": "strong-no",
	})
	a.UpdateGroupOrder([]string{"Uptown", "Downtown"})

	// Save.
	var buf bytes.Buffer
	if err := a.Save(&buf); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load into a fresh app with no config entries (simulates file having entries).
	a2, err := app.New(app.Params{
		People:   testPeople(),
		Timezone: time.UTC,
		Periods:  testPeriods(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := a2.Load(&buf); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Compare votes.
	origVotes := a.Votes()
	loadedVotes := a2.Votes()
	if len(loadedVotes) != len(origVotes) {
		t.Fatalf("round-trip: %d people, want %d", len(loadedVotes), len(origVotes))
	}
	for person, origPV := range origVotes {
		loadedPV := loadedVotes[person]
		if len(loadedPV) != len(origPV) {
			t.Errorf("person %q: %d groups, want %d", person, len(loadedPV), len(origPV))
			continue
		}
		for group, origGV := range origPV {
			loadedGV := loadedPV[group]
			if len(loadedGV) != len(origGV) {
				t.Errorf("person %q group %q: %d votes, want %d", person, group, len(loadedGV), len(origGV))
				continue
			}
			for entry, origVote := range origGV {
				if loadedGV[entry] != origVote {
					t.Errorf("person %q group %q entry %q = %q, want %q", person, group, entry, loadedGV[entry], origVote)
				}
			}
		}
	}

	// Compare entries.
	origEntries := a.Entries()
	loadedEntries := a2.Entries()
	if len(loadedEntries) != len(origEntries) {
		t.Fatalf("round-trip entries: %d, want %d", len(loadedEntries), len(origEntries))
	}

	// Compare group order.
	origOrder := a.GroupOrder()
	loadedOrder := a2.GroupOrder()
	if len(loadedOrder) != len(origOrder) {
		t.Fatalf("round-trip group order: %d, want %d", len(loadedOrder), len(origOrder))
	}
	for i, want := range origOrder {
		if loadedOrder[i] != want {
			t.Errorf("group order[%d] = %q, want %q", i, loadedOrder[i], want)
		}
	}
}

func TestUpdateEntries(t *testing.T) {
	var tests = []struct {
		desc      string
		entries   []app.Entry
		wantCount int
	}{{
		desc: "single entry",
		entries: []app.Entry{{
			Name:  "MyEntry",
			Group: "MyGroup",
			Cost:  2,
			Open:  map[string][]string{"mon": {"lunch", "dinner"}, "tue": {"lunch"}},
		}},
		wantCount: 1,
	}, {
		desc: "multiple entries",
		entries: []app.Entry{{
			Name:  "Entry1",
			Group: "G1",
			Cost:  1,
			Open:  map[string][]string{"mon": {"lunch"}},
		}, {
			Name:  "Entry2",
			Group: "G1",
			Cost:  3,
			Open:  map[string][]string{"tue": {"dinner"}},
		}, {
			Name:  "Entry3",
			Group: "G2",
			Cost:  4,
			Open:  map[string][]string{"fri": {"lunch", "dinner"}},
		}},
		wantCount: 3,
	}, {
		desc: "entry with no schedule",
		entries: []app.Entry{{
			Name:  "Entry1",
			Group: "G1",
			Cost:  1,
		}},
		wantCount: 1,
	}, {
		desc:      "empty entries",
		entries:   []app.Entry{},
		wantCount: 0,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			a := newTestApp(t)
			a.UpdateEntries(test.entries)

			entries := a.Entries()
			if len(entries) != test.wantCount {
				t.Errorf("UpdateEntries stored %d entries, want %d", len(entries), test.wantCount)
			}
		})
	}
}

func TestUpdateEntriesReplacesExisting(t *testing.T) {
	a := newTestApp(t)

	// Initially has testEntries (4 entries).
	if len(a.Entries()) != 4 {
		t.Fatalf("expected 4 initial entries, got %d", len(a.Entries()))
	}

	// Update with fewer entries.
	a.UpdateEntries([]app.Entry{{
		Name:  "NewEntry",
		Group: "NewGroup",
		Cost:  1,
		Open:  map[string][]string{"mon": {"lunch"}},
	}})

	entries := a.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after update, got %d", len(entries))
	}
	if entries[0].Name != "NewEntry" || entries[0].Group != "NewGroup" {
		t.Errorf("entry = %q/%q, want NewEntry/NewGroup", entries[0].Name, entries[0].Group)
	}
	if entries[0].Cost != 1 {
		t.Errorf("cost = %d, want 1", entries[0].Cost)
	}
}

func TestUpdateEntriesUpdatesVoteValidation(t *testing.T) {
	a := newTestApp(t)

	// Replace entries with new ones.
	a.UpdateEntries([]app.Entry{{
		Name:  "NewEntry",
		Group: "NewGroup",
		Cost:  1,
		Open:  map[string][]string{"mon": {"lunch"}},
	}})

	// Voting for old entries should fail.
	a.UpdateVotes("alice", map[string]string{
		"Downtown|Pizza Place": "yes",
		"NewGroup|NewEntry":    "strong-yes",
	})

	votes := a.Votes()["alice"]
	count := 0
	for _, gv := range votes {
		count += len(gv)
	}
	if count != 1 {
		t.Errorf("expected 1 valid vote, got %d", count)
	}
	if votes["NewGroup"]["NewEntry"] != "strong-yes" {
		t.Errorf("NewEntry vote = %q, want strong-yes", votes["NewGroup"]["NewEntry"])
	}
}

func TestNewImportsConfigEntries(t *testing.T) {
	// When no file is loaded (no autosave), entries come from config.
	a, err := app.New(app.Params{
		Entries:  testEntries(),
		People:   testPeople(),
		Timezone: time.UTC,
		Periods:  testPeriods(),
	})
	if err != nil {
		t.Fatal(err)
	}

	entries := a.Entries()
	if len(entries) != 4 {
		t.Errorf("expected 4 entries from config, got %d", len(entries))
	}
}

func TestNewWithNoEntries(t *testing.T) {
	// App can be created with no entries.
	a, err := app.New(app.Params{
		People:   testPeople(),
		Timezone: time.UTC,
		Periods:  testPeriods(),
	})
	if err != nil {
		t.Fatal(err)
	}

	entries := a.Entries()
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

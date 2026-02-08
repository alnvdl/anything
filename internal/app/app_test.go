package app_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alnvdl/anything/internal/app"
)

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

// newTestApp creates an App for testing.
func newTestApp(t *testing.T) *app.App {
	t.Helper()
	a, err := app.New(app.Params{
		Entries:  testEntries(),
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
		person    string
		votes     map[string]string
		wantCount int
	}{{
		desc:   "valid votes",
		person: "alice",
		votes: map[string]string{
			"Pizza Place":  "strong-yes",
			"Burger Joint": "no",
		},
		wantCount: 2,
	}, {
		desc:   "cleans invalid entry names",
		person: "alice",
		votes: map[string]string{
			"Pizza Place":   "yes",
			"Nonexistent":   "yes",
			"Also Not Real": "no",
		},
		wantCount: 1,
	}, {
		desc:   "cleans invalid vote values",
		person: "bob",
		votes: map[string]string{
			"Pizza Place":  "invalid-vote",
			"Burger Joint": "yes",
		},
		wantCount: 1,
	}, {
		desc:      "empty votes",
		person:    "alice",
		votes:     map[string]string{},
		wantCount: 0,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			a := newTestApp(t)
			a.UpdateVotes(test.person, test.votes)

			personVotes := a.Votes()[test.person]
			if len(personVotes) != test.wantCount {
				t.Errorf("UpdateVotes stored %d votes, want %d", len(personVotes), test.wantCount)
			}
		})
	}
}

func TestUpdateVotesOverwrites(t *testing.T) {
	a := newTestApp(t)

	// First submission.
	a.UpdateVotes("alice", map[string]string{
		"Pizza Place":  "strong-yes",
		"Burger Joint": "no",
	})

	// Second submission overwrites.
	a.UpdateVotes("alice", map[string]string{
		"Sushi Bar": "strong-no",
	})

	votes := a.Votes()["alice"]
	if len(votes) != 1 {
		t.Fatalf("expected 1 vote after overwrite, got %d", len(votes))
	}
	if votes["Sushi Bar"] != "strong-no" {
		t.Errorf("expected Sushi Bar vote to be strong-no, got %q", votes["Sushi Bar"])
	}
	if _, ok := votes["Pizza Place"]; ok {
		t.Error("Pizza Place vote should have been removed on overwrite")
	}
}

func TestVotePageData(t *testing.T) {
	a := newTestApp(t)
	a.UpdateVotes("alice", map[string]string{
		"Pizza Place": "strong-yes",
		"Sushi Bar":   "no",
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

func TestTallyData(t *testing.T) {
	a := newTestApp(t)

	// Alice votes.
	a.UpdateVotes("alice", map[string]string{
		"Pizza Place":  "strong-yes", // 3.
		"Burger Joint": "no",         // 1.
		"Sushi Bar":    "strong-no",  // 0.
		"Taco Stand":   "yes",        // 2.
	})

	// Bob votes.
	a.UpdateVotes("bob", map[string]string{
		"Pizza Place":  "yes",        // 2.
		"Burger Joint": "strong-yes", // 3.
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

func TestWeekdayString(t *testing.T) {
	var tests = []struct {
		desc string
		day  time.Weekday
		want string
	}{{
		desc: "sunday",
		day:  time.Sunday,
		want: "sun",
	}, {
		desc: "monday",
		day:  time.Monday,
		want: "mon",
	}, {
		desc: "tuesday",
		day:  time.Tuesday,
		want: "tue",
	}, {
		desc: "wednesday",
		day:  time.Wednesday,
		want: "wed",
	}, {
		desc: "thursday",
		day:  time.Thursday,
		want: "thu",
	}, {
		desc: "friday",
		day:  time.Friday,
		want: "fri",
	}, {
		desc: "saturday",
		day:  time.Saturday,
		want: "sat",
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := app.WeekdayString(test.day)
			if got != test.want {
				t.Errorf("WeekdayString(%v) = %q, want %q", test.day, got, test.want)
			}
		})
	}
}

func TestHandleVote(t *testing.T) {
	a := newTestApp(t)

	var tests = []struct {
		desc       string
		token      string
		wantStatus int
		wantBody   []string
	}{{
		desc:       "valid token",
		token:      "tokenA",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Anything", "alice", "Pizza Place", "Burger Joint", "Sushi Bar", "Taco Stand"},
	}, {
		desc:       "invalid token",
		token:      "bad",
		wantStatus: http.StatusForbidden,
	}, {
		desc:       "no token",
		token:      "",
		wantStatus: http.StatusForbidden,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?token="+test.token, nil)
			w := httptest.NewRecorder()
			a.ServeHTTP(w, req)

			if w.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, test.wantStatus)
			}

			body := w.Body.String()
			for _, s := range test.wantBody {
				if !strings.Contains(body, s) {
					t.Errorf("body does not contain %q", s)
				}
			}
		})
	}
}

func TestHandleVoteShowsCurrentVotes(t *testing.T) {
	a := newTestApp(t)
	a.UpdateVotes("alice", map[string]string{
		"Pizza Place": "strong-yes",
	})

	req := httptest.NewRequest("GET", "/?token=tokenA", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	body := w.Body.String()
	// The radio button for strong-yes on Pizza Place should be checked.
	if !strings.Contains(body, `value="strong-yes" checked`) {
		t.Error("expected strong-yes to be checked for Pizza Place")
	}
}

func TestHandleTallyGet(t *testing.T) {
	a := newTestApp(t)

	now := time.Now().In(time.UTC)
	weekday := app.WeekdayFullNames[now.Weekday()]

	var tests = []struct {
		desc       string
		token      string
		period     string
		wantStatus int
		wantBody   []string
	}{{
		desc:       "valid request",
		token:      "tokenA",
		period:     "lunch",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Anything", "for lunch on", weekday, "Downtown", "Uptown"},
	}, {
		desc:       "invalid token",
		token:      "bad",
		period:     "lunch",
		wantStatus: http.StatusForbidden,
	}, {
		desc:       "invalid period",
		token:      "tokenA",
		period:     "brunch",
		wantStatus: http.StatusBadRequest,
	}, {
		desc:       "missing period",
		token:      "tokenA",
		period:     "",
		wantStatus: http.StatusBadRequest,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/votes?period="+test.period+"&token="+test.token, nil)
			w := httptest.NewRecorder()
			a.ServeHTTP(w, req)

			if w.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, test.wantStatus)
			}

			body := w.Body.String()
			for _, s := range test.wantBody {
				if !strings.Contains(body, s) {
					t.Errorf("body does not contain %q", s)
				}
			}
		})
	}
}

func TestHandleTallyPost(t *testing.T) {
	a := newTestApp(t)

	form := url.Values{}
	form.Set("Pizza Place", "strong-yes")
	form.Set("Burger Joint", "no")

	req := httptest.NewRequest("POST", "/votes?token=tokenA", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Anything") {
		t.Error("body does not contain Anything title")
	}

	// Verify votes were stored.
	aliceVotes := a.Votes()["alice"]
	if aliceVotes["Pizza Place"] != "strong-yes" {
		t.Errorf("alice Pizza Place vote = %q, want strong-yes", aliceVotes["Pizza Place"])
	}
	if aliceVotes["Burger Joint"] != "no" {
		t.Errorf("alice Burger Joint vote = %q, want no", aliceVotes["Burger Joint"])
	}
}

func TestHandleTallyPostInvalidToken(t *testing.T) {
	a := newTestApp(t)

	req := httptest.NewRequest("POST", "/votes?token=bad", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestStaticFileServing(t *testing.T) {
	a := newTestApp(t)

	var tests = []struct {
		desc       string
		path       string
		wantStatus int
		wantBody   string
	}{{
		desc:       "lightwebapp.css",
		path:       "/static/lightwebapp.css",
		wantStatus: http.StatusOK,
		wantBody:   "--lwa-font-size",
	}, {
		desc:       "anything.css",
		path:       "/static/anything.css",
		wantStatus: http.StatusOK,
		wantBody:   ".entry-list",
	}, {
		desc:       "bootstrap-reboot.css",
		path:       "/static/bootstrap-reboot.css",
		wantStatus: http.StatusOK,
		wantBody:   "Bootstrap Reboot",
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest("GET", test.path, nil)
			w := httptest.NewRecorder()
			a.ServeHTTP(w, req)

			if w.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, test.wantStatus)
			}

			body := w.Body.String()
			if !strings.Contains(body, test.wantBody) {
				t.Errorf("body does not contain %q", test.wantBody)
			}
		})
	}
}

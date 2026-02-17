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
		wantBody:   []string{"Anything", "alice", "Pizza Place", "Burger Joint", "Sushi Bar", "Taco Stand", "Downtown|Pizza Place", "Uptown|Sushi Bar"},
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
		"Downtown|Pizza Place": "strong-yes",
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

	// Fix time to Monday at 12pm (lunch period).
	a.SetNowFunc(func() time.Time {
		return time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	})

	var tests = []struct {
		desc       string
		token      string
		period     string
		weekday    string
		wantStatus int
		wantBody   []string
	}{{
		desc:       "current period shows today",
		token:      "tokenA",
		period:     "lunch",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Anything", "for lunch on", "Monday", "Downtown", "Uptown", "weekday=sun", "weekday=tue"},
	}, {
		desc:       "past period shows next day",
		token:      "tokenA",
		period:     "breakfast",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Anything", "for breakfast on", "Tuesday", "weekday=mon", "weekday=wed"},
	}, {
		desc:       "future period shows today",
		token:      "tokenA",
		period:     "dinner",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Anything", "for dinner on", "Monday", "weekday=sun", "weekday=tue"},
	}, {
		desc:       "explicit weekday overrides default",
		token:      "tokenA",
		period:     "lunch",
		weekday:    "fri",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Anything", "for lunch on", "Friday", "weekday=thu", "weekday=sat"},
	}, {
		desc:       "explicit weekday sunday wraps to saturday and monday",
		token:      "tokenA",
		period:     "lunch",
		weekday:    "sun",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Sunday", "weekday=sat", "weekday=mon"},
	}, {
		desc:       "explicit weekday saturday wraps to friday and sunday",
		token:      "tokenA",
		period:     "dinner",
		weekday:    "sat",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Saturday", "weekday=fri", "weekday=sun"},
	}, {
		desc:       "invalid weekday",
		token:      "tokenA",
		period:     "lunch",
		weekday:    "xyz",
		wantStatus: http.StatusBadRequest,
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
			u := "/votes?period=" + test.period + "&token=" + test.token
			if test.weekday != "" {
				u += "&weekday=" + test.weekday
			}
			req := httptest.NewRequest("GET", u, nil)
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

	// Fix time to Monday at 12pm (lunch period).
	a.SetNowFunc(func() time.Time {
		return time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	})

	form := url.Values{}
	form.Set("Downtown|Pizza Place", "strong-yes")
	form.Set("Downtown|Burger Joint", "no")

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

	// Verify day navigation links appear.
	for _, s := range []string{"weekday=sun", "weekday=tue"} {
		if !strings.Contains(body, s) {
			t.Errorf("body does not contain %q", s)
		}
	}

	// Verify votes were stored.
	aliceVotes := a.Votes()["alice"]
	if aliceVotes["Downtown"]["Pizza Place"] != "strong-yes" {
		t.Errorf("alice Pizza Place vote = %q, want strong-yes", aliceVotes["Downtown"]["Pizza Place"])
	}
	if aliceVotes["Downtown"]["Burger Joint"] != "no" {
		t.Errorf("alice Burger Joint vote = %q, want no", aliceVotes["Downtown"]["Burger Joint"])
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

			cc := w.Header().Get("Cache-Control")
			if cc != "max-age=604800, public" {
				t.Errorf("Cache-Control = %q, want %q", cc, "max-age=604800, public")
			}
		})
	}
}

func TestHandleEntriesGet(t *testing.T) {
	a := newTestApp(t)

	var tests = []struct {
		desc       string
		token      string
		wantStatus int
		wantBody   []string
	}{{
		desc:       "valid token shows entries form",
		token:      "tokenA",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Anything", "Downtown", "Uptown", "Pizza Place", "Burger Joint", "Sushi Bar", "Taco Stand", "Save", "Add entry", "Add group", "mon", "tue", "wed", "thu", "fri", "sat", "sun", "breakfast", "lunch", "dinner", "move-group-up", "move-group-down"},
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
			req := httptest.NewRequest("GET", "/entries?token="+test.token, nil)
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

// entryMatches checks if an entry matches the expected values.
func entryMatches(got, want app.Entry) bool {
	if got.Name != want.Name || got.Group != want.Group || got.Cost != want.Cost {
		return false
	}
	if len(got.Open) != len(want.Open) {
		return false
	}
	for day, wantPeriods := range want.Open {
		gotPeriods, ok := got.Open[day]
		if !ok || len(gotPeriods) != len(wantPeriods) {
			return false
		}
		for i, p := range wantPeriods {
			if gotPeriods[i] != p {
				return false
			}
		}
	}
	return true
}

// findEntry returns the entry with the given group and name, or false.
func findEntry(entries []app.Entry, group, name string) (app.Entry, bool) {
	for _, e := range entries {
		if e.Group == group && e.Name == name {
			return e, true
		}
	}
	return app.Entry{}, false
}

func TestHandleEntriesPost(t *testing.T) {
	var tests = []struct {
		desc         string
		token        string
		form         url.Values
		wantStatus   int
		wantLocation string
		wantEntries  []app.Entry
		wantOrder    []string
	}{{
		desc:  "valid post updates entries",
		token: "tokenA",
		form: url.Values{
			"NewGroup|NewEntry":     {"2;mon:lunch,dinner;tue:lunch"},
			"NewGroup|AnotherEntry": {"1;fri:dinner"},
			"_groupOrder":           {"NewGroup"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries: []app.Entry{{
			Name:  "NewEntry",
			Group: "NewGroup",
			Cost:  2,
			Open:  map[string][]string{"mon": {"lunch", "dinner"}, "tue": {"lunch"}},
		}, {
			Name:  "AnotherEntry",
			Group: "NewGroup",
			Cost:  1,
			Open:  map[string][]string{"fri": {"dinner"}},
		}},
		wantOrder: []string{"NewGroup"},
	}, {
		desc:       "invalid token",
		token:      "bad",
		wantStatus: http.StatusForbidden,
	}, {
		desc:  "no pipe separator in key is discarded",
		token: "tokenA",
		form: url.Values{
			"NoPipeSeparator": {"1;mon:lunch"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries:  []app.Entry{},
	}, {
		desc:  "empty group is discarded",
		token: "tokenA",
		form: url.Values{
			"|EntryName": {"1;mon:lunch"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries:  []app.Entry{},
	}, {
		desc:  "empty name is discarded",
		token: "tokenA",
		form: url.Values{
			"GroupName|": {"1;mon:lunch"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries:  []app.Entry{},
	}, {
		desc:  "non-numeric cost is discarded",
		token: "tokenA",
		form: url.Values{
			"G|Entry": {"abc;mon:lunch"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries:  []app.Entry{},
	}, {
		desc:  "cost below range is discarded",
		token: "tokenA",
		form: url.Values{
			"G|Entry": {"0;mon:lunch"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries:  []app.Entry{},
	}, {
		desc:  "cost above range is discarded",
		token: "tokenA",
		form: url.Values{
			"G|Entry": {"5;mon:lunch"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries:  []app.Entry{},
	}, {
		desc:  "empty schedule parts are skipped",
		token: "tokenA",
		form: url.Values{
			"G|Entry": {"2;;mon:lunch;;"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries: []app.Entry{{
			Name:  "Entry",
			Group: "G",
			Cost:  2,
			Open:  map[string][]string{"mon": {"lunch"}},
		}},
	}, {
		desc:  "schedule part without colon is skipped",
		token: "tokenA",
		form: url.Values{
			"G|Entry": {"2;nocolon;mon:lunch"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries: []app.Entry{{
			Name:  "Entry",
			Group: "G",
			Cost:  2,
			Open:  map[string][]string{"mon": {"lunch"}},
		}},
	}, {
		desc:  "schedule part with empty periods string is skipped",
		token: "tokenA",
		form: url.Values{
			"G|Entry": {"2;mon:;tue:lunch"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries: []app.Entry{{
			Name:  "Entry",
			Group: "G",
			Cost:  2,
			Open:  map[string][]string{"tue": {"lunch"}},
		}},
	}, {
		desc:  "mix of valid and invalid entries",
		token: "tokenA",
		form: url.Values{
			"G|Valid":     {"2;mon:lunch,dinner"},
			"NoPipe":      {"1;mon:lunch"},
			"|EmptyGroup": {"1;mon:lunch"},
			"G|":          {"1;mon:lunch"},
			"G|BadCost":   {"abc;mon:lunch"},
			"G|CostZero":  {"0;mon:lunch"},
			"G|CostFive":  {"5;mon:lunch"},
			"G|AlsoValid": {"1;tue:dinner"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries: []app.Entry{{
			Name:  "Valid",
			Group: "G",
			Cost:  2,
			Open:  map[string][]string{"mon": {"lunch", "dinner"}},
		}, {
			Name:  "AlsoValid",
			Group: "G",
			Cost:  1,
			Open:  map[string][]string{"tue": {"dinner"}},
		}},
	}, {
		desc:  "entry with no schedule",
		token: "tokenA",
		form: url.Values{
			"G|Entry": {"3"},
		},
		wantStatus:   http.StatusSeeOther,
		wantLocation: "/?token=tokenA",
		wantEntries: []app.Entry{{
			Name:  "Entry",
			Group: "G",
			Cost:  3,
			Open:  map[string][]string{},
		}},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			a := newTestApp(t)

			u := "/entries?token=" + test.token
			var req *http.Request
			if test.form != nil {
				req = httptest.NewRequest("POST", u, strings.NewReader(test.form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req = httptest.NewRequest("POST", u, nil)
			}
			w := httptest.NewRecorder()
			a.ServeHTTP(w, req)

			if w.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, test.wantStatus)
			}

			if test.wantLocation != "" {
				loc := w.Header().Get("Location")
				if loc != test.wantLocation {
					t.Errorf("Location = %q, want %q", loc, test.wantLocation)
				}
			}

			if test.wantEntries != nil {
				entries := a.Entries()
				if len(entries) != len(test.wantEntries) {
					t.Fatalf("expected %d entries, got %d", len(test.wantEntries), len(entries))
				}
				for _, want := range test.wantEntries {
					got, ok := findEntry(entries, want.Group, want.Name)
					if !ok {
						t.Errorf("missing entry %s|%s", want.Group, want.Name)
						continue
					}
					if !entryMatches(got, want) {
						t.Errorf("entry %s|%s = %+v, want %+v", want.Group, want.Name, got, want)
					}
				}
			}

			if test.wantOrder != nil {
				order := a.GroupOrder()
				if len(order) != len(test.wantOrder) {
					t.Fatalf("expected %d group order entries, got %d", len(test.wantOrder), len(order))
				}
				for i, want := range test.wantOrder {
					if order[i] != want {
						t.Errorf("group order[%d] = %q, want %q", i, order[i], want)
					}
				}
			}
		})
	}
}

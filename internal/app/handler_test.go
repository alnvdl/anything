package app_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
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

	// Fix time to Monday at 12pm (lunch period).
	a.SetNowFunc(func() time.Time {
		return time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	})

	var tests = []struct {
		desc       string
		token      string
		period     string
		wantStatus int
		wantBody   []string
	}{{
		desc:       "current period shows today",
		token:      "tokenA",
		period:     "lunch",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Anything", "for lunch on", "Monday", "Downtown", "Uptown"},
	}, {
		desc:       "past period shows next day",
		token:      "tokenA",
		period:     "breakfast",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Anything", "for breakfast on", "Tuesday"},
	}, {
		desc:       "future period shows today",
		token:      "tokenA",
		period:     "dinner",
		wantStatus: http.StatusOK,
		wantBody:   []string{"Anything", "for dinner on", "Monday"},
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

	// Fix time to Monday at 12pm (lunch period).
	a.SetNowFunc(func() time.Time {
		return time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	})

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

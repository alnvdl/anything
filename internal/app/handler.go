package app

import (
	"net/http"

	"github.com/alnvdl/anything/internal/version"
)

// authenticate extracts the token from the request and resolves it to a person.
func (a *App) authenticate(r *http.Request) (string, bool) {
	token := r.URL.Query().Get("token")
	return a.personForToken(token)
}

// handleVote serves the voting page.
func (a *App) handleVote(w http.ResponseWriter, r *http.Request) {
	person, ok := a.authenticate(r)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	token := r.URL.Query().Get("token")
	groups := a.votePageData(person)

	data := pageData{
		Title:   "Anything",
		Token:   token,
		Person:  person,
		Periods: a.periodList,
		Groups:  groups,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.voteTmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleTallyGet serves the tally page for a given period.
func (a *App) handleTallyGet(w http.ResponseWriter, r *http.Request) {
	person, ok := a.authenticate(r)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	token := r.URL.Query().Get("token")
	period := r.URL.Query().Get("period")
	if _, ok := a.periods[period]; !ok {
		http.Error(w, "Bad Request: invalid period", http.StatusBadRequest)
		return
	}

	wd := a.periodTallyWeekday(period)
	groups := a.tallyData(wd, period)

	data := pageData{
		Title:   "Anything",
		Token:   token,
		Person:  person,
		Period:  period,
		Weekday: weekdayFullNames[wd],
		Periods: a.periodList,
		Groups:  groups,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tallyTmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleTallyPost handles vote submission and shows the tally.
func (a *App) handleTallyPost(w http.ResponseWriter, r *http.Request) {
	person, ok := a.authenticate(r)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	token := r.URL.Query().Get("token")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Extract votes from form data.
	votes := make(map[string]string)
	for name := range r.PostForm {
		votes[name] = r.PostForm.Get(name)
	}

	a.updateVotes(person, votes)

	now := a.nowFunc().In(a.timezone)
	period := periodForHour(a.periods, now.Hour())

	if period == "" {
		http.Error(w, "No active period", http.StatusBadRequest)
		return
	}

	groups := a.tallyData(now.Weekday(), period)

	data := pageData{
		Title:   "Anything",
		Token:   token,
		Person:  person,
		Period:  period,
		Weekday: weekdayFullNames[now.Weekday()],
		Periods: a.periodList,
		Groups:  groups,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tallyTmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (a *App) handleStatus(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, version.Version(), http.StatusOK)
}

// ServeHTTP implements http.Handler.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

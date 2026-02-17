package app

import (
	"net/http"
	"strconv"
	"strings"
	"time"

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
	groups := a.entriesData(person)

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
		Weekday: weekdays[wd].Full,
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
		Weekday: weekdays[now.Weekday()].Full,
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

// handleEntriesGet serves the entries editing page.
func (a *App) handleEntriesGet(w http.ResponseWriter, r *http.Request) {
	_, ok := a.authenticate(r)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	token := r.URL.Query().Get("token")
	groups := a.entriesData("")

	wds := make([]weekdayInfo, 7)
	for wd := time.Sunday; wd <= time.Saturday; wd++ {
		wds[wd] = weekdays[wd]
	}

	data := pageData{
		Title:    "Anything",
		Token:    token,
		Periods:  a.periodList,
		Weekdays: wds,
		Groups:   groups,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.entriesTmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleEntriesPost handles entry editing form submission.
func (a *App) handleEntriesPost(w http.ResponseWriter, r *http.Request) {
	_, ok := a.authenticate(r)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	token := r.URL.Query().Get("token")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var entries []Entry
	for key := range r.PostForm {
		value := r.PostForm.Get(key)
		group, name, ok := strings.Cut(key, "|")
		if !ok || group == "" || name == "" {
			continue
		}

		parts := strings.Split(value, ";")
		if len(parts) < 1 {
			continue
		}

		cost, err := strconv.Atoi(parts[0])
		if err != nil || cost < 1 || cost > 4 {
			continue
		}

		open := make(map[string][]string)
		for _, part := range parts[1:] {
			if part == "" {
				continue
			}
			day, periodsStr, ok := strings.Cut(part, ":")
			if !ok || periodsStr == "" {
				continue
			}
			periods := strings.Split(periodsStr, ",")
			open[day] = periods
		}

		entries = append(entries, Entry{
			Name:  name,
			Group: group,
			Cost:  cost,
			Open:  open,
		})
	}

	groupOrder := r.PostForm["_groupOrder"]

	a.updateEntries(entries)
	a.updateGroupOrder(groupOrder)

	http.Redirect(w, r, "/?token="+token, http.StatusSeeOther)
}

// ServeHTTP implements http.Handler.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

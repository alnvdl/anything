// Package app implements the Anything voting application.
package app

import (
	"cmp"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/alnvdl/autosave"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// voteScores maps vote values to their numeric scores.
var voteScores = map[string]int{
	"strong-no":  0,
	"no":         1,
	"yes":        2,
	"strong-yes": 3,
}

// weekdayShortNames maps Go weekdays to short names used in config.
var weekdayShortNames = map[time.Weekday]string{
	time.Sunday:    "sun",
	time.Monday:    "mon",
	time.Tuesday:   "tue",
	time.Wednesday: "wed",
	time.Thursday:  "thu",
	time.Friday:    "fri",
	time.Saturday:  "sat",
}

// weekdayFullNames maps short weekday names to full names for display.
var weekdayFullNames = map[time.Weekday]string{
	time.Sunday:    "Sunday",
	time.Monday:    "Monday",
	time.Tuesday:   "Tuesday",
	time.Wednesday: "Wednesday",
	time.Thursday:  "Thursday",
	time.Friday:    "Friday",
	time.Saturday:  "Saturday",
}

// Entry represents a voting entry from config.
type Entry struct {
	Name  string              `json:"name"`
	Group string              `json:"group"`
	Open  map[string][]string `json:"open"`
	Cost  int                 `json:"cost"`
}

// Periods maps period names to [start_hour, end_hour).
type Periods map[string][2]int

// Params contains all parameters needed to create an App.
type Params struct {
	Entries  []Entry
	People   map[string]string
	Timezone *time.Location
	Periods  Periods

	// AutoSaveParams is the configuration for auto-save. If FilePath is
	// empty, auto-save will be disabled and votes will only be kept in
	// memory. The LoaderSaver field will be set to the created App, so any
	// value set by the caller will be ignored.
	AutoSaveParams autosave.Params
}

// pageData holds template data for rendering pages.
type pageData struct {
	Title   string
	Token   string
	Person  string
	Period  string
	Weekday string
	Periods []string
	Groups  []groupData
}

// groupData holds a group of entries for template rendering.
type groupData struct {
	Name    string
	Entries []entryData
}

// entryData holds a single entry for template rendering.
type entryData struct {
	Name        string
	CurrentVote string
	Score       int
	CostDisplay string
	Closed      bool
}

// App is the core application struct.
type App struct {
	entries    []Entry
	entryNames map[string]bool
	people     map[string]string
	tokenMap   map[string]string
	timezone   *time.Location
	periods    Periods
	periodList []string
	nowFunc    func() time.Time

	mu    sync.RWMutex
	votes map[string]map[string]string

	autoSaver *autosave.AutoSaver

	mux       *http.ServeMux
	voteTmpl  *template.Template
	tallyTmpl *template.Template
}

// New creates a new App with the given parameters.
func New(params Params) (*App, error) {
	a := &App{
		entries:    params.Entries,
		entryNames: make(map[string]bool),
		people:     params.People,
		tokenMap:   make(map[string]string),
		timezone:   params.Timezone,
		periods:    params.Periods,
		votes:      make(map[string]map[string]string),
		nowFunc:    time.Now,
	}

	for _, e := range a.entries {
		a.entryNames[e.Name] = true
	}

	for person, token := range a.people {
		a.tokenMap[token] = person
	}

	// Build period list sorted by start time for consistent display.
	for name := range a.periods {
		a.periodList = append(a.periodList, name)
	}
	slices.SortFunc(a.periodList, func(a, b string) int {
		return cmp.Compare(params.Periods[a][0], params.Periods[b][0])
	})

	// Parse templates.
	funcMap := template.FuncMap{
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
	}

	var err error
	a.voteTmpl, err = template.New("").Funcs(funcMap).ParseFS(templateFS,
		"templates/layout.html",
		"templates/nav.html",
		"templates/entrylist.html",
		"templates/vote.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parsing vote templates: %w", err)
	}

	a.tallyTmpl, err = template.New("").Funcs(funcMap).ParseFS(templateFS,
		"templates/layout.html",
		"templates/nav.html",
		"templates/entrylist.html",
		"templates/tally.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parsing tally templates: %w", err)
	}

	// Initialize auto-save if configured.
	if params.AutoSaveParams.FilePath != "" {
		params.AutoSaveParams.LoaderSaver = a

		var asErr error
		a.autoSaver, asErr = autosave.New(params.AutoSaveParams)
		if asErr != nil {
			return nil, fmt.Errorf("cannot initialize auto-saver: %v", asErr)
		}
	}

	// Set up routes.
	a.mux = http.NewServeMux()

	staticContent, _ := fs.Sub(staticFS, "static")
	a.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticContent)))
	a.mux.HandleFunc("GET /{$}", a.handleVote)
	a.mux.HandleFunc("GET /votes", a.handleTallyGet)
	a.mux.HandleFunc("POST /votes", a.handleTallyPost)

	return a, nil
}

// ServeHTTP implements http.Handler.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

// personForToken returns the person name for a given token.
func (a *App) personForToken(token string) (string, bool) {
	person, ok := a.tokenMap[token]
	return person, ok
}

// delayAutoSave calls Delay on the autoSaver if it is not nil.
func (a *App) delayAutoSave() {
	if a.autoSaver != nil {
		a.autoSaver.Delay()
	}
}

// Load deserializes votes from the given reader.
func (a *App) Load(r io.Reader) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	dec := json.NewDecoder(r)
	var data map[string]map[string]string
	err := dec.Decode(&data)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		// Ignoring a corrupted or empty file is intentional: we prefer to
		// lose all votes than prevent the application from starting.
		return nil
	} else if err != nil {
		return fmt.Errorf("cannot deserialize votes: %w", err)
	}
	a.votes = data
	return nil
}

// Save serializes votes to the given writer.
func (a *App) Save(w io.Writer) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	enc := json.NewEncoder(w)
	if err := enc.Encode(a.votes); err != nil {
		return fmt.Errorf("cannot serialize votes: %w", err)
	}
	return nil
}

// Close stops the auto-save mechanism and waits for it to finish.
func (a *App) Close() {
	if a.autoSaver != nil {
		a.autoSaver.Close()
	}
}

// updateVotes saves votes for a person, cleaning invalid entries and vote values.
func (a *App) updateVotes(person string, votes map[string]string) {
	defer a.delayAutoSave()
	a.mu.Lock()
	defer a.mu.Unlock()

	cleaned := make(map[string]string)
	for name, vote := range votes {
		if !a.entryNames[name] {
			continue
		}
		if _, ok := voteScores[vote]; !ok {
			continue
		}
		cleaned[name] = vote
	}
	a.votes[person] = cleaned
}

// votePageData returns grouped entries with current votes for the vote page.
func (a *App) votePageData(person string) []groupData {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Group entries by group name.
	groupMap := make(map[string][]Entry)
	for _, e := range a.entries {
		groupMap[e.Group] = append(groupMap[e.Group], e)
	}

	groupNames := make([]string, 0, len(groupMap))
	for name := range groupMap {
		groupNames = append(groupNames, name)
	}
	slices.Sort(groupNames)

	personVotes := a.votes[person]

	var result []groupData
	for _, gName := range groupNames {
		entries := slices.Clone(groupMap[gName])
		slices.SortFunc(entries, func(a, b Entry) int {
			return cmp.Compare(a.Name, b.Name)
		})

		var eds []entryData
		for _, e := range entries {
			vote := ""
			if personVotes != nil {
				vote = personVotes[e.Name]
			}
			eds = append(eds, entryData{
				Name:        e.Name,
				CurrentVote: vote,
			})
		}

		result = append(result, groupData{
			Name:    gName,
			Entries: eds,
		})
	}

	return result
}

// tallyData computes the tally for a given weekday and period.
func (a *App) tallyData(weekday time.Weekday, period string) []groupData {
	a.mu.RLock()
	defer a.mu.RUnlock()

	type scored struct {
		entry  Entry
		score  int
		closed bool
	}

	var items []scored
	for _, e := range a.entries {
		sum := 0
		for person := range a.people {
			voteVal := 2 // Default: yes.
			if personVotes, ok := a.votes[person]; ok {
				if v, ok := personVotes[e.Name]; ok {
					voteVal = voteScores[v]
				}
			}
			sum += voteVal
		}
		score := sum*3 - e.Cost

		// Check if the entry is open for this weekday and period.
		closed := true
		if periods, ok := e.Open[weekdayShortNames[weekday]]; ok {
			if slices.Contains(periods, period) {
				closed = false
			}
		}

		items = append(items, scored{e, score, closed})
	}

	// Group by group name.
	groupMap := make(map[string][]scored)
	for _, s := range items {
		groupMap[s.entry.Group] = append(groupMap[s.entry.Group], s)
	}

	groupNames := make([]string, 0, len(groupMap))
	for name := range groupMap {
		groupNames = append(groupNames, name)
	}
	slices.Sort(groupNames)

	sortEntries := func(a, b scored) int {
		// Score descending.
		if c := cmp.Compare(b.score, a.score); c != 0 {
			return c
		}
		// Cost ascending.
		if c := cmp.Compare(a.entry.Cost, b.entry.Cost); c != 0 {
			return c
		}
		// Name ascending.
		return cmp.Compare(a.entry.Name, b.entry.Name)
	}

	var result []groupData
	for _, gName := range groupNames {
		entries := groupMap[gName]

		// Separate open and closed entries.
		var open, closedEntries []scored
		for _, s := range entries {
			if s.closed {
				closedEntries = append(closedEntries, s)
			} else {
				open = append(open, s)
			}
		}

		slices.SortFunc(open, sortEntries)
		slices.SortFunc(closedEntries, sortEntries)

		// Open entries first, then closed.
		combined := append(open, closedEntries...)

		var eds []entryData
		for _, s := range combined {
			eds = append(eds, entryData{
				Name:        s.entry.Name,
				Score:       s.score,
				CostDisplay: strings.Repeat("$", s.entry.Cost),
				Closed:      s.closed,
			})
		}

		result = append(result, groupData{
			Name:    gName,
			Entries: eds,
		})
	}

	return result
}

// weekdayForTally returns the appropriate weekday for displaying a tally.
// If the requested period has already passed for the current day, it returns
// the next day's weekday.
func weekdayForTally(periods Periods, periodList []string, currentHour int, currentWeekday time.Weekday, period string) time.Weekday {
	currentPeriod := periodForHour(periods, currentHour)
	if currentPeriod == period {
		return currentWeekday
	}

	currentIdx := slices.Index(periodList, currentPeriod)
	requestedIdx := slices.Index(periodList, period)

	if currentIdx >= 0 && requestedIdx >= 0 && requestedIdx < currentIdx {
		return (currentWeekday + 1) % 7
	}

	return currentWeekday
}

// periodForHour returns the period name for a given hour.
func periodForHour(periods Periods, hour int) string {
	for name, bounds := range periods {
		start, end := bounds[0], bounds[1]
		if start < end {
			if hour >= start && hour < end {
				return name
			}
		} else if start > end {
			// Wraps around midnight.
			if hour >= start || hour < end {
				return name
			}
		}
	}
	return ""
}

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

	now := a.nowFunc().In(a.timezone)
	wd := weekdayForTally(a.periods, a.periodList, now.Hour(), now.Weekday(), period)
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

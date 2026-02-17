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

// Entry represents a voting entry with its name, group, cost and schedule.
type Entry struct {
	Name  string
	Group string
	Open  map[string][]string
	Cost  int
}

// Periods maps period names to [start_hour, end_hour).
type Periods map[string][2]int

// EntryVote represents a vote value for a single entry.
type EntryVote string

// GroupVote maps entry names to their votes within a group.
type GroupVote map[string]EntryVote

// PersonVote maps group names to group votes for a person.
type PersonVote map[string]GroupVote

// voteScores maps vote values to their numeric scores.
var voteScores = map[EntryVote]int{
	"strong-no":  0,
	"no":         1,
	"yes":        2,
	"strong-yes": 3,
}

// weekdayInfo holds display information for a weekday.
type weekdayInfo struct {
	Short string
	Full  string
}

// weekdays maps Go weekdays to their short and full names.
var weekdays = map[time.Weekday]weekdayInfo{
	time.Sunday:    {Short: "sun", Full: "Sunday"},
	time.Monday:    {Short: "mon", Full: "Monday"},
	time.Tuesday:   {Short: "tue", Full: "Tuesday"},
	time.Wednesday: {Short: "wed", Full: "Wednesday"},
	time.Thursday:  {Short: "thu", Full: "Thursday"},
	time.Friday:    {Short: "fri", Full: "Friday"},
	time.Saturday:  {Short: "sat", Full: "Saturday"},
}

// db holds all persistent data for the application in memory, and it can be
// persisted to disk in JSON format by the auto-save mechanism.
type db struct {
	Entries    []Entry               `json:"entries"`
	Votes      map[string]PersonVote `json:"votes"`
	GroupOrder []string              `json:"groupOrder"`
}

// entryGroups returns a map from entry names to their set of groups.
func (d *db) entryGroups() map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	for _, e := range d.Entries {
		if result[e.Name] == nil {
			result[e.Name] = make(map[string]bool)
		}
		result[e.Name][e.Group] = true
	}
	return result
}

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
	Title    string
	Token    string
	Person   string
	Period   string
	Weekday  string
	Periods  []string
	Weekdays []weekdayInfo
	Groups   []groupData
}

// groupData holds a group of entries for template rendering.
type groupData struct {
	Name    string
	Entries []entryData
}

// entryData holds a single entry for template rendering.
type entryData struct {
	Name        string
	Group       string
	CurrentVote string
	Score       int
	Cost        int
	CostDisplay string
	Open        map[string][]string
	Closed      bool
	StrongNo    bool
}

// App is the core application struct.
type App struct {
	people     map[string]string
	tokens     map[string]string
	timezone   *time.Location
	periods    Periods
	periodList []string
	nowFunc    func() time.Time

	mu sync.RWMutex
	db db

	autoSaver *autosave.AutoSaver

	mux         *http.ServeMux
	voteTmpl    *template.Template
	tallyTmpl   *template.Template
	entriesTmpl *template.Template
}

var tmplFuncs = template.FuncMap{
	"title": func(s string) string {
		if len(s) == 0 {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	},
	"contains": func(slice []string, item string) bool {
		return slices.Contains(slice, item)
	},
}

// New creates a new App with the given parameters.
func New(params Params) (*App, error) {
	a := &App{
		people:   params.People,
		tokens:   make(map[string]string),
		timezone: params.Timezone,
		periods:  params.Periods,
		db: db{
			Votes: make(map[string]PersonVote),
		},
		nowFunc: time.Now,
	}

	for person, token := range a.people {
		a.tokens[token] = person
	}

	// Build period list sorted by start time for consistent display.
	for name := range a.periods {
		a.periodList = append(a.periodList, name)
	}
	slices.SortFunc(a.periodList, func(a, b string) int {
		return cmp.Compare(params.Periods[a][0], params.Periods[b][0])
	})

	var err error
	a.voteTmpl, err = template.New("").Funcs(tmplFuncs).ParseFS(templateFS,
		"templates/layout.html",
		"templates/nav.html",
		"templates/entrylist.html",
		"templates/icons.html",
		"templates/vote.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parsing vote templates: %w", err)
	}

	a.tallyTmpl, err = template.New("").Funcs(tmplFuncs).ParseFS(templateFS,
		"templates/layout.html",
		"templates/nav.html",
		"templates/entrylist.html",
		"templates/icons.html",
		"templates/tally.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parsing tally templates: %w", err)
	}

	a.entriesTmpl, err = template.New("").Funcs(tmplFuncs).ParseFS(templateFS,
		"templates/layout.html",
		"templates/nav.html",
		"templates/entries.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parsing entries templates: %w", err)
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

	// Import entries from config if none were loaded from file.
	if len(a.db.Entries) == 0 {
		a.db.Entries = params.Entries
	}

	// Set up routes.
	a.mux = http.NewServeMux()

	staticContent, _ := fs.Sub(staticFS, "static")
	staticHandler := http.StripPrefix("/static/", http.FileServerFS(staticContent))
	a.mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=604800, public")
		staticHandler.ServeHTTP(w, r)
	})
	a.mux.HandleFunc("GET /{$}", a.handleVote)
	a.mux.HandleFunc("GET /votes", a.handleTallyGet)
	a.mux.HandleFunc("POST /votes", a.handleTallyPost)
	a.mux.HandleFunc("GET /entries", a.handleEntriesGet)
	a.mux.HandleFunc("POST /entries", a.handleEntriesPost)
	a.mux.HandleFunc("GET /status", a.handleStatus)

	return a, nil
}

// personForToken returns the person name for a given token.
func (a *App) personForToken(token string) (string, bool) {
	person, ok := a.tokens[token]
	return person, ok
}

// delayAutoSave calls Delay on the autoSaver if it is not nil.
func (a *App) delayAutoSave() {
	if a.autoSaver != nil {
		a.autoSaver.Delay()
	}
}

// Load deserializes data from the given reader.
func (a *App) Load(r io.Reader) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	dec := json.NewDecoder(r)
	var data db
	err := dec.Decode(&data)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		// Ignoring a corrupted or empty file is intentional: we prefer to
		// lose all data than prevent the application from starting.
		return nil
	} else if err != nil {
		return fmt.Errorf("cannot deserialize data: %w", err)
	}
	if data.Votes != nil {
		a.db.Votes = data.Votes
	}
	if data.Entries != nil {
		a.db.Entries = data.Entries
	}
	if data.GroupOrder != nil {
		a.db.GroupOrder = data.GroupOrder
	}
	return nil
}

// Save serializes data to the given writer.
func (a *App) Save(w io.Writer) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	enc := json.NewEncoder(w)
	if err := enc.Encode(a.db); err != nil {
		return fmt.Errorf("cannot serialize data: %w", err)
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
// Form keys are expected in "Group|Entry" format.
func (a *App) updateVotes(person string, votes map[string]string) {
	defer a.delayAutoSave()
	a.mu.Lock()
	defer a.mu.Unlock()

	entryGroup := a.db.entryGroups()
	pv := make(PersonVote)
	for key, vote := range votes {
		group, name, ok := strings.Cut(key, "|")
		if !ok {
			continue
		}
		groups, exists := entryGroup[name]
		if !exists || !groups[group] {
			continue
		}
		if _, ok := voteScores[EntryVote(vote)]; !ok {
			continue
		}
		if pv[group] == nil {
			pv[group] = make(GroupVote)
		}
		pv[group][name] = EntryVote(vote)
	}
	a.db.Votes[person] = pv
}

// updateEntries replaces all entries.
func (a *App) updateEntries(entries []Entry) {
	defer a.delayAutoSave()
	a.mu.Lock()
	defer a.mu.Unlock()

	a.db.Entries = entries
}

// updateGroupOrder replaces the group ordering.
func (a *App) updateGroupOrder(order []string) {
	defer a.delayAutoSave()
	a.mu.Lock()
	defer a.mu.Unlock()

	a.db.GroupOrder = order
}

// entriesData returns grouped entries for rendering templates (vote and edit).
func (a *App) entriesData(person string) []groupData {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Group entries by group name.
	groupMap := make(map[string][]Entry)
	for _, e := range a.db.Entries {
		groupMap[e.Group] = append(groupMap[e.Group], e)
	}

	groupNames := make([]string, 0, len(groupMap))
	for name := range groupMap {
		groupNames = append(groupNames, name)
	}
	sortGroupNames(groupNames, a.db.GroupOrder)

	personVotes := a.db.Votes[person]

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
				if gv, ok := personVotes[e.Group]; ok {
					vote = string(gv[e.Name])
				}
			}
			eds = append(eds, entryData{
				Name:        e.Name,
				Group:       e.Group,
				CurrentVote: vote,
				Cost:        e.Cost,
				Open:        e.Open,
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
		entry    Entry
		score    int
		closed   bool
		strongNo bool
	}

	var items []scored
	for _, e := range a.db.Entries {
		sum := 0
		strongNo := false
		for person := range a.people {
			voteVal := 2 // Default: yes.
			if personVotes, ok := a.db.Votes[person]; ok {
				if gv, ok := personVotes[e.Group]; ok {
					if v, ok := gv[e.Name]; ok {
						voteVal = voteScores[v]
						if v == "strong-no" {
							strongNo = true
						}
					}
				}
			}
			sum += voteVal
		}
		score := sum*3 - e.Cost

		// Check if the entry is open for this weekday and period.
		closed := true
		if periods, ok := e.Open[weekdays[weekday].Short]; ok {
			if slices.Contains(periods, period) {
				closed = false
			}
		}

		items = append(items, scored{e, score, closed, strongNo})
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
	sortGroupNames(groupNames, a.db.GroupOrder)

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
				StrongNo:    s.strongNo,
			})
		}

		result = append(result, groupData{
			Name:    gName,
			Entries: eds,
		})
	}

	return result
}

// periodTallyWeekday returns the appropriate weekday for displaying a tally.
// If the requested period has already passed for the current day, it returns
// the next day's weekday.
func (a *App) periodTallyWeekday(period string) time.Weekday {
	now := a.nowFunc().In(a.timezone)
	currentHour := now.Hour()
	currentWeekday := now.Weekday()

	currentPeriod := periodForHour(a.periods, currentHour)
	if currentPeriod == period {
		return currentWeekday
	}

	currentIdx := slices.Index(a.periodList, currentPeriod)
	requestedIdx := slices.Index(a.periodList, period)

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

// sortGroupNames sorts group names in place: groups present in groupOrder come
// first (in the order specified), followed by remaining groups sorted
// alphabetically.
func sortGroupNames(names []string, groupOrder []string) {
	orderIndex := make(map[string]int, len(groupOrder))
	for i, name := range groupOrder {
		orderIndex[name] = i
	}
	slices.SortFunc(names, func(a, b string) int {
		idxA, okA := orderIndex[a]
		idxB, okB := orderIndex[b]
		switch {
		case okA && okB:
			return cmp.Compare(idxA, idxB)
		case okA:
			return -1
		case okB:
			return 1
		default:
			return cmp.Compare(a, b)
		}
	})
}

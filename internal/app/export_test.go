package app

import "time"

// GroupData is an exported alias for groupData, for use in tests.
type GroupData = groupData

// EntryData is an exported alias for entryData, for use in tests.
type EntryData = entryData

// PersonForToken exposes personForToken for testing.
func (a *App) PersonForToken(token string) (string, bool) {
	return a.personForToken(token)
}

// UpdateVotes exposes updateVotes for testing.
func (a *App) UpdateVotes(person string, votes map[string]string) {
	a.updateVotes(person, votes)
}

// VotePageData exposes votePageData for testing.
func (a *App) VotePageData(person string) []GroupData {
	return a.entriesData(person)
}

// TallyData exposes tallyData for testing.
func (a *App) TallyData(weekday time.Weekday, period string) []GroupData {
	return a.tallyData(weekday, period)
}

// PeriodForHour exposes periodForHour for testing.
func PeriodForHour(periods Periods, hour int) string {
	return periodForHour(periods, hour)
}

// Weekdays exposes weekdays for testing.
var Weekdays = weekdays

// SetNowFunc overrides the time function used by the App for testing.
func (a *App) SetNowFunc(f func() time.Time) {
	a.nowFunc = f
}

// PeriodTallyWeekday exposes periodTallyWeekday for testing.
func (a *App) PeriodTallyWeekday(period string) time.Weekday {
	return a.periodTallyWeekday(period)
}

// Votes returns the current votes map for testing.
func (a *App) Votes() map[string]PersonVote {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.db.Votes
}

// Entries returns the current entries for testing.
func (a *App) Entries() []Entry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.db.Entries
}

// UpdateEntries exposes updateEntries for testing.
func (a *App) UpdateEntries(entries []Entry) {
	a.updateEntries(entries)
}

// UpdateGroupOrder exposes updateGroupOrder for testing.
func (a *App) UpdateGroupOrder(order []string) {
	a.updateGroupOrder(order)
}

// GroupOrder returns the current group order for testing.
func (a *App) GroupOrder() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.db.GroupOrder
}

// SortGroupNames exposes sortGroupNames for testing.
func SortGroupNames(names []string, groupOrder []string) {
	sortGroupNames(names, groupOrder)
}

// WeekdayForShort exposes weekdayForShort for testing.
func WeekdayForShort(short string) (time.Weekday, bool) {
	return weekdayForShort(short)
}

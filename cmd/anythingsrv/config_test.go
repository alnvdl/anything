package main

import (
	"testing"
)

func TestPort(t *testing.T) {
	var tests = []struct {
		desc    string
		env     string
		want    int
		wantErr bool
	}{{
		desc: "valid port",
		env:  "8080",
		want: 8080,
	}, {
		desc:    "not set",
		env:     "",
		wantErr: true,
	}, {
		desc:    "not a number",
		env:     "abc",
		wantErr: true,
	}, {
		desc:    "zero",
		env:     "0",
		wantErr: true,
	}, {
		desc:    "too large",
		env:     "99999",
		wantErr: true,
	}, {
		desc:    "negative",
		env:     "-1",
		wantErr: true,
	}, {
		desc: "boundary low",
		env:  "1",
		want: 1,
	}, {
		desc: "boundary high",
		env:  "65535",
		want: 65535,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Setenv("PORT", test.env)
			got, err := Port()
			if (err != nil) != test.wantErr {
				t.Fatalf("Port() err = %v, wantErr = %v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("Port() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestEntries(t *testing.T) {
	var tests = []struct {
		desc      string
		env       string
		wantCount int
		wantErr   bool
	}{{
		desc:      "valid entries",
		env:       `[{"name":"A","group":"G1","open":{"mon":["lunch"]},"cost":2}]`,
		wantCount: 1,
	}, {
		desc:    "not set",
		env:     "",
		wantErr: true,
	}, {
		desc:    "invalid JSON",
		env:     `not json`,
		wantErr: true,
	}, {
		desc:      "multiple entries",
		env:       `[{"name":"A","group":"G1","open":{},"cost":1},{"name":"B","group":"G2","open":{},"cost":3}]`,
		wantCount: 2,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Setenv("ENTRIES", test.env)
			got, err := Entries()
			if (err != nil) != test.wantErr {
				t.Fatalf("Entries() err = %v, wantErr = %v", err, test.wantErr)
			}
			if len(got) != test.wantCount {
				t.Errorf("Entries() returned %d entries, want %d", len(got), test.wantCount)
			}
		})
	}
}

func TestPeople(t *testing.T) {
	var tests = []struct {
		desc      string
		env       string
		wantCount int
		wantErr   bool
	}{{
		desc:      "valid people",
		env:       `{"alice":"token1","bob":"token2"}`,
		wantCount: 2,
	}, {
		desc:    "not set",
		env:     "",
		wantErr: true,
	}, {
		desc:    "invalid JSON",
		env:     `{bad}`,
		wantErr: true,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Setenv("PEOPLE", test.env)
			got, err := People()
			if (err != nil) != test.wantErr {
				t.Fatalf("People() err = %v, wantErr = %v", err, test.wantErr)
			}
			if len(got) != test.wantCount {
				t.Errorf("People() returned %d entries, want %d", len(got), test.wantCount)
			}
		})
	}
}

func TestTimezone(t *testing.T) {
	var tests = []struct {
		desc    string
		env     string
		wantErr bool
	}{{
		desc: "valid timezone",
		env:  "America/New_York",
	}, {
		desc:    "not set",
		env:     "",
		wantErr: true,
	}, {
		desc:    "invalid timezone",
		env:     "Not/A/Timezone",
		wantErr: true,
	}, {
		desc: "utc",
		env:  "UTC",
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Setenv("TIMEZONE", test.env)
			got, err := Timezone()
			if (err != nil) != test.wantErr {
				t.Fatalf("Timezone() err = %v, wantErr = %v", err, test.wantErr)
			}
			if !test.wantErr && got == nil {
				t.Error("Timezone() returned nil location")
			}
		})
	}
}

func TestPeriods(t *testing.T) {
	var tests = []struct {
		desc    string
		env     string
		wantErr bool
	}{{
		desc: "valid periods",
		env:  `{"breakfast":[0,10],"lunch":[10,15],"dinner":[15,0]}`,
	}, {
		desc:    "not set",
		env:     "",
		wantErr: true,
	}, {
		desc:    "invalid JSON",
		env:     `{bad}`,
		wantErr: true,
	}, {
		desc:    "overlapping periods",
		env:     `{"breakfast":[0,12],"lunch":[10,15]}`,
		wantErr: true,
	}, {
		desc:    "equal start and end",
		env:     `{"allday":[5,5]}`,
		wantErr: true,
	}, {
		desc: "non-contiguous periods",
		env:  `{"morning":[6,12],"evening":[18,22]}`,
	}, {
		desc:    "wrap-around overlap",
		env:     `{"night":[22,6],"early":[4,10]}`,
		wantErr: true,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Setenv("PERIODS", test.env)
			got, err := Periods()
			if (err != nil) != test.wantErr {
				t.Fatalf("Periods() err = %v, wantErr = %v", err, test.wantErr)
			}
			if !test.wantErr && got == nil {
				t.Error("Periods() returned nil")
			}
		})
	}
}

func TestHoursForPeriod(t *testing.T) {
	var tests = []struct {
		desc  string
		start int
		end   int
		want  []int
	}{{
		desc:  "simple range",
		start: 0,
		end:   3,
		want:  []int{0, 1, 2},
	}, {
		desc:  "wrap around midnight",
		start: 22,
		end:   2,
		want:  []int{22, 23, 0, 1},
	}, {
		desc:  "single hour",
		start: 5,
		end:   6,
		want:  []int{5},
	}, {
		desc:  "large range",
		start: 0,
		end:   10,
		want:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := hoursForPeriod(test.start, test.end)
			if len(got) != len(test.want) {
				t.Fatalf("hoursForPeriod(%d, %d) = %v, want %v", test.start, test.end, got, test.want)
			}
			for i, h := range got {
				if h != test.want[i] {
					t.Errorf("hoursForPeriod(%d, %d)[%d] = %d, want %d", test.start, test.end, i, h, test.want[i])
				}
			}
		})
	}
}

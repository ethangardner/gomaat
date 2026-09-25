package model

import "time"

type Commit struct {
	Rev        string
	Date       string // "YYYY-MM-DD"
	Author     string
	Entity     string
	LocAdded   int
	LocDeleted int
}

type Options struct {
	MinRevs          int
	MinSharedRevs    int
	MinCoupling      float64
	MaxCoupling      float64
	MaxChangesetSize int
	AgeTimeNow       time.Time
	ReworkWindow     time.Duration
	ReworkTimeNow    time.Time
	VerboseResults   bool
	FormerAuthors    map[string]struct{}
}

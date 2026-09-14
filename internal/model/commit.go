package model

import "time"

type Commit struct {
	Rev        string
	Date       string // "YYYY-MM-DD"
	Author     string
	Entity     string
	Message    string // full commit message: subject + body + trailers, raw
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
	VerboseResults   bool
}

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
	VerboseResults   bool
}

func DefaultOptions() Options {
	return Options{
		MinRevs:          5,
		MinSharedRevs:    5,
		MinCoupling:      30,
		MaxCoupling:      100,
		MaxChangesetSize: 30,
		AgeTimeNow:       time.Now(),
	}
}

package analysis

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/ethangardner/gomaat/internal/model"
)

type BusFactorResult struct {
	Entity    string
	BusFactor int
	TopOwners []string
	Ownership float64 // combined share of TopOwners, as a percentage
}

// BusFactor returns, per entity, the fewest authors who together own more
// than 50% of its added lines. Entities with no added lines are skipped.
func BusFactor(commits []model.Commit, _ model.Options) []BusFactorResult {
	byKey, totals := sumPerEntityAuthor(commits, linesAdded)

	type share struct {
		author string
		added  int
	}
	sharesByEntity := map[string][]share{}
	for k, added := range byKey {
		if added > 0 {
			sharesByEntity[k.entity] = append(sharesByEntity[k.entity], share{k.author, added})
		}
	}

	results := make([]BusFactorResult, 0, len(sharesByEntity))
	for entity, shares := range sharesByEntity {
		total := totals[entity]
		slices.SortFunc(shares, func(a, b share) int {
			return cmp.Or(cmp.Compare(b.added, a.added), cmp.Compare(a.author, b.author))
		})
		var owners []string
		var owned int
		for _, s := range shares {
			owners = append(owners, s.author)
			owned += s.added
			if owned*2 > total {
				break
			}
		}
		results = append(results, BusFactorResult{entity, len(owners), owners, percent(owned, total)})
	}
	slices.SortFunc(results, func(a, b BusFactorResult) int {
		return cmp.Or(cmp.Compare(a.BusFactor, b.BusFactor), cmp.Compare(a.Entity, b.Entity))
	})
	return results
}

func FormatBusFactor(results []BusFactorResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "bus-factor", "top-owners", "ownership"}}
	for _, r := range results {
		out = append(out, []string{
			r.Entity, fmt.Sprint(r.BusFactor),
			strings.Join(r.TopOwners, ";"), fmt.Sprintf("%.2f", r.Ownership),
		})
	}
	return out
}

type KnowledgeLossResult struct {
	Entity      string
	FormerAdded int
	TotalAdded  int
	Loss        float64 // FormerAdded / TotalAdded, as a percentage
}

// KnowledgeLoss returns, per entity, the share of added lines written by
// authors in opts.FormerAuthors. Entities with no added lines are skipped.
func KnowledgeLoss(commits []model.Commit, opts model.Options) []KnowledgeLossResult {
	byKey, totals := sumPerEntityAuthor(commits, linesAdded)

	former := map[string]int{}
	for k, added := range byKey {
		if _, ok := opts.FormerAuthors[k.author]; ok {
			former[k.entity] += added
		}
	}

	results := make([]KnowledgeLossResult, 0, len(totals))
	for entity, total := range totals {
		if total == 0 {
			continue
		}
		lost := former[entity]
		results = append(results, KnowledgeLossResult{entity, lost, total, percent(lost, total)})
	}
	slices.SortFunc(results, func(a, b KnowledgeLossResult) int {
		return cmp.Or(cmp.Compare(b.Loss, a.Loss), cmp.Compare(a.Entity, b.Entity))
	})
	return results
}

func FormatKnowledgeLoss(results []KnowledgeLossResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "former-added", "total-added", "knowledge-loss"}}
	for _, r := range results {
		out = append(out, []string{
			r.Entity, fmt.Sprint(r.FormerAdded),
			fmt.Sprint(r.TotalAdded), fmt.Sprintf("%.2f", r.Loss),
		})
	}
	return out
}

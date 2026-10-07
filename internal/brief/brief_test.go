package brief

import (
	"strings"
	"testing"
	"time"

	"github.com/ethangardner/gomaat/internal/model"
)

// briefCommits couples cli/root.go and cli/flags.go across two commits and
// leaves docs/old.md untouched for over a year before opts.Now.
var briefCommits = []model.Commit{
	{Rev: "r1", Date: "2023-01-10", Author: "Alice", Entity: "docs/old.md", LocAdded: 40},
	{Rev: "r2", Date: "2024-05-01", Author: "Bob", Entity: "cli/root.go", LocAdded: 50},
	{Rev: "r2", Date: "2024-05-01", Author: "Bob", Entity: "cli/flags.go", LocAdded: 20},
	{Rev: "r3", Date: "2024-06-01", Author: "Alice", Entity: "cli/root.go", LocAdded: 5},
	{Rev: "r3", Date: "2024-06-01", Author: "Alice", Entity: "cli/flags.go", LocAdded: 5},
}

var briefOpts = Options{Top: 10, Now: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), DormantMonths: 12}

func TestRender(t *testing.T) {
	got := Render(briefCommits, briefOpts)
	for _, want := range []string{
		"# Architecture brief\n",
		"## Summary\n",
		"- Commits: 3\n- Entities: 3\n- Authors: 2\n- Date range: 2023-01-10 to 2024-06-01\n",
		"## Hotspots\n",
		"| cli/flags.go | 2 |\n| cli/root.go | 2 |\n| docs/old.md | 1 |\n",
		"## Coupling\n",
		"| cli/flags.go | cli/root.go | 100 | 2 |\n",
		"## Main developers\n",
		"| component | main-dev | added | total-added | ownership |\n",
		"| cli | Bob | 70 | 80 | 87.50 |\n",
		"## Activity\n",
		"2 entities changed in the last 12 months; 1 untouched for 12 months or more.\n",
		"### Recently active\n",
		"### Dormant\n",
		"| docs/old.md | 18 |\n",
		"## Suggested reading order\n",
		"1. cli/flags.go\n2. cli/root.go\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "3. docs/old.md") {
		t.Errorf("reading order should skip files that never change with others, got:\n%s", got)
	}
}

func TestRenderGroupedKeepsEntitiesAsComponents(t *testing.T) {
	got := Render(briefCommits, Options{Top: 10, Now: briefOpts.Now, DormantMonths: 12, Grouped: true})
	if !strings.Contains(got, "| cli/root.go | Bob | 50 | 55 | 90.91 |\n") {
		t.Errorf("expected per-entity main developers when grouped, got:\n%s", got)
	}
}

func TestRenderTopLimitsTables(t *testing.T) {
	got := Render(briefCommits, Options{Top: 1, Now: briefOpts.Now, DormantMonths: 12})
	if strings.Contains(got, "| cli/root.go | 2 |") {
		t.Errorf("expected --top 1 to keep only the first hotspot, got:\n%s", got)
	}
}

func TestRenderEmpty(t *testing.T) {
	got := Render(nil, briefOpts)
	for _, want := range []string{"- Date range: none\n", "_None._\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

package cli

import (
	"strings"
	"testing"
)

func TestBriefCmdRunE(t *testing.T) {
	got, err := runOnFixture(t, newBriefCmd(), map[string]string{"age-time-now": "2024-06-01"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "| foo.go | 1 |") {
		t.Errorf("expected foo.go hotspot row, got:\n%s", got)
	}
}

func TestBriefCmdRejectsFlags(t *testing.T) {
	tests := []struct {
		name    string
		set     func()
		flags   map[string]string
		wantErr string
	}{
		{name: "--format", set: func() { outputFormat = "json" }, wantErr: "--format"},
		{name: "--rows", set: func() { maxRows = 5 }, wantErr: "--top"},
		{name: "bad --age-time-now", set: func() {}, flags: map[string]string{"age-time-now": "soon"}, wantErr: "--age-time-now"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t)
			tt.set()
			assertErrContains(t, runCmd(t, newBriefCmd(), tt.flags), tt.wantErr)
		})
	}
}

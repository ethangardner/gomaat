package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hhatto/gocloc"
)

// makeResult builds a gocloc.Result from the provided languages and files.
// Total counts are derived from the file map so tests don't have to maintain them manually.
func makeResult(langs map[string]*gocloc.Language, files map[string]*gocloc.ClocFile) *gocloc.Result {
	total := &gocloc.Language{Name: "TOTAL"}
	for _, f := range files {
		total.Code += f.Code
		total.Comments += f.Comments
		total.Blanks += f.Blanks
		total.Total++
	}
	return &gocloc.Result{
		Total:     total,
		Languages: langs,
		Files:     files,
	}
}

func TestPathMatchesAnyExclude(t *testing.T) {
	tests := []struct {
		path     string
		excludes []string
		want     bool
	}{
		{"vendor/foo.go", []string{"vendor/"}, true},
		{"src/foo.go", []string{"vendor/"}, false},
		{"src/foo.pb.go", []string{"*.pb.go"}, true},
		{"src/foo.go", []string{"*.pb.go"}, false},
		{"src/foo.go", []string{}, false},
		{"vendor/foo.go", []string{"*.pb.go", "vendor/"}, true}, // second pattern matches
		{"src/foo.pb.go", []string{"*.pb.go", "vendor/"}, true}, // first pattern matches
		{"src/foo.go", []string{"*.pb.go", "vendor/"}, false},   // neither matches
	}

	for _, tt := range tests {
		got := pathMatchesAnyExclude(tt.path, tt.excludes)
		if got != tt.want {
			t.Errorf("pathMatchesAnyExclude(%q, %v) = %v, want %v", tt.path, tt.excludes, got, tt.want)
		}
	}
}

func TestApplyClocExcludesNoOp(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{
			"Go": {Name: "Go", Files: []string{"src/main.go"}, Code: 50, Comments: 5, Blanks: 3},
		},
		map[string]*gocloc.ClocFile{
			"src/main.go": {Name: "src/main.go", Lang: "Go", Code: 50, Comments: 5, Blanks: 3},
		},
	)

	applyClocExcludes(result, nil)

	if len(result.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(result.Files))
	}
	if result.Total.Code != 50 {
		t.Errorf("expected total code 50, got %d", result.Total.Code)
	}
}

func TestApplyClocExcludesRemovesFile(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{
			"Go": {
				Name:     "Go",
				Files:    []string{"src/main.go", "vendor/lib.go"},
				Code:     100,
				Comments: 10,
				Blanks:   8,
			},
		},
		map[string]*gocloc.ClocFile{
			"src/main.go":   {Name: "src/main.go", Lang: "Go", Code: 60, Comments: 6, Blanks: 4},
			"vendor/lib.go": {Name: "vendor/lib.go", Lang: "Go", Code: 40, Comments: 4, Blanks: 4},
		},
	)

	applyClocExcludes(result, []string{"vendor/"})

	if _, ok := result.Files["vendor/lib.go"]; ok {
		t.Error("vendor/lib.go should have been removed from Files")
	}
	if _, ok := result.Files["src/main.go"]; !ok {
		t.Error("src/main.go should still be in Files")
	}

	lang := result.Languages["Go"]
	if lang.Code != 60 {
		t.Errorf("language Code: got %d, want 60", lang.Code)
	}
	if lang.Comments != 6 {
		t.Errorf("language Comments: got %d, want 6", lang.Comments)
	}
	if lang.Blanks != 4 {
		t.Errorf("language Blanks: got %d, want 4", lang.Blanks)
	}
	if len(lang.Files) != 1 || lang.Files[0] != "src/main.go" {
		t.Errorf("language Files: got %v, want [src/main.go]", lang.Files)
	}

	if result.Total.Code != 60 {
		t.Errorf("total Code: got %d, want 60", result.Total.Code)
	}
	if result.Total.Total != 1 {
		t.Errorf("total file count: got %d, want 1", result.Total.Total)
	}
}

func TestApplyClocExcludesAllFilesOfLanguage(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{
			"Go":   {Name: "Go", Files: []string{"vendor/a.go", "vendor/b.go"}, Code: 80, Comments: 4, Blanks: 6},
			"YAML": {Name: "YAML", Files: []string{"config.yaml"}, Code: 30, Comments: 0, Blanks: 2},
		},
		map[string]*gocloc.ClocFile{
			"vendor/a.go": {Name: "vendor/a.go", Lang: "Go", Code: 50, Comments: 3, Blanks: 4},
			"vendor/b.go": {Name: "vendor/b.go", Lang: "Go", Code: 30, Comments: 1, Blanks: 2},
			"config.yaml": {Name: "config.yaml", Lang: "YAML", Code: 30, Comments: 0, Blanks: 2},
		},
	)

	applyClocExcludes(result, []string{"vendor/"})

	goLang := result.Languages["Go"]
	if len(goLang.Files) != 0 {
		t.Errorf("Go files should be empty after excluding vendor/, got %v", goLang.Files)
	}
	if goLang.Code != 0 {
		t.Errorf("Go Code should be 0, got %d", goLang.Code)
	}

	if result.Total.Code != 30 {
		t.Errorf("total Code: got %d, want 30", result.Total.Code)
	}
	if result.Total.Total != 1 {
		t.Errorf("total file count: got %d, want 1", result.Total.Total)
	}
}

func TestApplyClocExcludesGlob(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{
			"Go": {
				Name:  "Go",
				Files: []string{"src/types.pb.go", "src/main.go"},
				Code:  70,
			},
		},
		map[string]*gocloc.ClocFile{
			"src/types.pb.go": {Name: "src/types.pb.go", Lang: "Go", Code: 40},
			"src/main.go":     {Name: "src/main.go", Lang: "Go", Code: 30},
		},
	)

	applyClocExcludes(result, []string{"*.pb.go"})

	if _, ok := result.Files["src/types.pb.go"]; ok {
		t.Error("src/types.pb.go should have been excluded by *.pb.go")
	}
	if result.Languages["Go"].Code != 30 {
		t.Errorf("language Code: got %d, want 30", result.Languages["Go"].Code)
	}
}

func TestClocLanguageRowsHeader(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{
			"Go": {Name: "Go", Files: []string{"main.go"}, Code: 10},
		},
		map[string]*gocloc.ClocFile{
			"main.go": {Name: "main.go", Lang: "Go", Code: 10},
		},
	)

	rows := clocLanguageRows(result)

	want := []string{"Language", "Files", "Blank", "Comment", "Code"}
	if len(rows[0]) != len(want) {
		t.Fatalf("header length: got %d, want %d", len(rows[0]), len(want))
	}
	for i, col := range want {
		if rows[0][i] != col {
			t.Errorf("header[%d]: got %q, want %q", i, rows[0][i], col)
		}
	}
}

func TestClocLanguageRowsSortedByCode(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{
			"YAML": {Name: "YAML", Files: []string{"a.yaml"}, Code: 10},
			"Go":   {Name: "Go", Files: []string{"a.go", "b.go"}, Code: 200},
			"JSON": {Name: "JSON", Files: []string{"a.json"}, Code: 50},
		},
		map[string]*gocloc.ClocFile{
			"a.yaml": {Name: "a.yaml", Lang: "YAML", Code: 10},
			"a.go":   {Name: "a.go", Lang: "Go", Code: 100},
			"b.go":   {Name: "b.go", Lang: "Go", Code: 100},
			"a.json": {Name: "a.json", Lang: "JSON", Code: 50},
		},
	)

	rows := clocLanguageRows(result)

	// rows[0] is header; last row is TOTAL
	dataRows := rows[1 : len(rows)-1]
	if dataRows[0][0] != "Go" {
		t.Errorf("first language: got %q, want Go (highest code)", dataRows[0][0])
	}
	if dataRows[1][0] != "JSON" {
		t.Errorf("second language: got %q, want JSON", dataRows[1][0])
	}
	if dataRows[2][0] != "YAML" {
		t.Errorf("third language: got %q, want YAML (lowest code)", dataRows[2][0])
	}
}

func TestClocLanguageRowsSkipsEmptyLanguages(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{
			"Go":   {Name: "Go", Files: []string{"main.go"}, Code: 50},
			"YAML": {Name: "YAML", Files: []string{}, Code: 0}, // no files
		},
		map[string]*gocloc.ClocFile{
			"main.go": {Name: "main.go", Lang: "Go", Code: 50},
		},
	)

	rows := clocLanguageRows(result)

	// header + Go + TOTAL = 3 rows; YAML should be skipped
	if len(rows) != 3 {
		t.Errorf("expected 3 rows (header, Go, TOTAL), got %d", len(rows))
	}
}

func TestClocLanguageRowsTotalRow(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{
			"Go": {Name: "Go", Files: []string{"a.go", "b.go"}, Code: 80, Comments: 8, Blanks: 6},
		},
		map[string]*gocloc.ClocFile{
			"a.go": {Name: "a.go", Lang: "Go", Code: 40, Comments: 4, Blanks: 3},
			"b.go": {Name: "b.go", Lang: "Go", Code: 40, Comments: 4, Blanks: 3},
		},
	)

	rows := clocLanguageRows(result)
	total := rows[len(rows)-1]

	if total[0] != "TOTAL" {
		t.Errorf("last row label: got %q, want TOTAL", total[0])
	}
	if total[1] != "2" {
		t.Errorf("TOTAL files: got %q, want 2", total[1])
	}
	if total[4] != "80" {
		t.Errorf("TOTAL code: got %q, want 80", total[4])
	}
}

func TestClocFileRowsHeader(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{},
		map[string]*gocloc.ClocFile{
			"main.go": {Name: "main.go", Lang: "Go", Code: 10},
		},
	)

	rows := clocFileRows(result)

	want := []string{"File", "Language", "Blank", "Comment", "Code"}
	if len(rows[0]) != len(want) {
		t.Fatalf("header length: got %d, want %d", len(rows[0]), len(want))
	}
	for i, col := range want {
		if rows[0][i] != col {
			t.Errorf("header[%d]: got %q, want %q", i, rows[0][i], col)
		}
	}
}

func TestClocFileRowsSortedByName(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{},
		map[string]*gocloc.ClocFile{
			"src/z.go": {Name: "src/z.go", Lang: "Go", Code: 10},
			"src/a.go": {Name: "src/a.go", Lang: "Go", Code: 20},
			"src/m.go": {Name: "src/m.go", Lang: "Go", Code: 15},
		},
	)

	rows := clocFileRows(result)

	if rows[1][0] != "src/a.go" {
		t.Errorf("row 1 file: got %q, want src/a.go", rows[1][0])
	}
	if rows[2][0] != "src/m.go" {
		t.Errorf("row 2 file: got %q, want src/m.go", rows[2][0])
	}
	if rows[3][0] != "src/z.go" {
		t.Errorf("row 3 file: got %q, want src/z.go", rows[3][0])
	}
}

func TestClocFileRowsColumns(t *testing.T) {
	result := makeResult(
		map[string]*gocloc.Language{},
		map[string]*gocloc.ClocFile{
			"foo.go": {Name: "foo.go", Lang: "Go", Code: 42, Comments: 7, Blanks: 3},
		},
	)

	rows := clocFileRows(result)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (header + 1 file), got %d", len(rows))
	}
	row := rows[1]
	checks := []struct{ col, want string }{
		{row[0], "foo.go"},
		{row[1], "Go"},
		{row[2], "3"},
		{row[3], "7"},
		{row[4], "42"},
	}
	for _, c := range checks {
		if c.col != c.want {
			t.Errorf("column value: got %q, want %q", c.col, c.want)
		}
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
	} {
		if err := exec.Command("git", append([]string{"-C", dir}, args...)...).Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
}

func TestGitTrackedFilesOnlyReturnsTrackedFiles(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	tracked := filepath.Join(dir, "main.go")
	untracked := filepath.Join(dir, "scratch.go")
	for _, f := range []string{tracked, untracked} {
		if err := os.WriteFile(f, []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := exec.Command("git", "-C", dir, "add", "main.go").Run(); err != nil {
		t.Fatal(err)
	}

	files, root, err := gitTrackedFiles(dir)
	if err != nil {
		t.Fatalf("gitTrackedFiles: %v", err)
	}

	if root != dir {
		t.Errorf("root: got %q, want %q", root, dir)
	}
	if len(files) != 1 || files[0] != tracked {
		t.Errorf("got %v, want [%s]", files, tracked)
	}
}

func TestGitTrackedFilesFromSubdirectoryUsesRepoRoot(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	nestedDir := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	tracked := filepath.Join(nestedDir, "main.go")
	if err := os.WriteFile(tracked, []byte("package pkg\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "add", "pkg/main.go").Run(); err != nil {
		t.Fatal(err)
	}

	files, root, err := gitTrackedFiles(nestedDir)
	if err != nil {
		t.Fatalf("gitTrackedFiles: %v", err)
	}

	if root != dir {
		t.Errorf("root: got %q, want %q", root, dir)
	}
	if len(files) != 1 || files[0] != tracked {
		t.Errorf("got %v, want [%s]", files, tracked)
	}
}

func TestGitTrackedFilesErrorsOnNonRepo(t *testing.T) {
	dir := t.TempDir()
	_, _, err := gitTrackedFiles(dir)
	if err == nil {
		t.Error("expected error for non-git directory, got nil")
	}
}

func TestClocIntegration(t *testing.T) {
	dir := t.TempDir()

	goFile := "package main\n\n// Doc comment\nfunc main() {\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goFile), 0644); err != nil {
		t.Fatal(err)
	}

	processor := gocloc.NewProcessor(gocloc.NewDefinedLanguages(), gocloc.NewClocOptions())
	result, err := processor.Analyze([]string{dir})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	rows := clocLanguageRows(result)

	if len(rows) < 3 {
		t.Fatalf("expected at least header + Go + TOTAL, got %d rows", len(rows))
	}
	if rows[0][0] != "Language" {
		t.Errorf("missing header row, got %v", rows[0])
	}

	goIdx := -1
	for i, row := range rows {
		if row[0] == "Go" {
			goIdx = i
			break
		}
	}
	if goIdx == -1 {
		t.Fatal("Go language not found in output")
	}

	files, _ := strconv.Atoi(rows[goIdx][1])
	code, _ := strconv.Atoi(rows[goIdx][4])
	if files != 1 {
		t.Errorf("Go files: got %d, want 1", files)
	}
	if code < 1 {
		t.Errorf("Go code lines: got %d, want > 0", code)
	}
}

func TestClocIntegrationExclude(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"main.go":     "package main\n\nfunc main() {}\n",
		"types.pb.go": "package main\n\ntype Foo struct{}\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	processor := gocloc.NewProcessor(gocloc.NewDefinedLanguages(), gocloc.NewClocOptions())
	result, err := processor.Analyze([]string{dir})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	relativizeResult(result, dir)
	applyClocExcludes(result, []string{"*.pb.go"})

	if _, ok := result.Files["types.pb.go"]; ok {
		t.Error("types.pb.go should have been excluded")
	}
	if _, ok := result.Files["main.go"]; !ok {
		t.Error("main.go should still be present")
	}
	if result.Total.Total != 1 {
		t.Errorf("total file count: got %d, want 1", result.Total.Total)
	}
}

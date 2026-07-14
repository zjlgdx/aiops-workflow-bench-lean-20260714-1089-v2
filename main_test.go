package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAddPersistsTodosAndListPrintsThemInIDOrder(t *testing.T) {
	todo := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.db")

	stdout, stderr, err := runTodo(t, todo, database, "add", "  Buy milk  ")
	if err != nil {
		t.Fatalf("add first todo: %v\nstderr: %s", err, stderr)
	}
	if stdout != "added 1\n" {
		t.Fatalf("add first todo stdout = %q, want %q", stdout, "added 1\n")
	}
	if stderr != "" {
		t.Fatalf("add first todo stderr = %q, want empty", stderr)
	}

	stdout, stderr, err = runTodo(t, todo, database, "add", "Walk dog")
	if err != nil {
		t.Fatalf("add second todo: %v\nstderr: %s", err, stderr)
	}
	if stdout != "added 2\n" {
		t.Fatalf("add second todo stdout = %q, want %q", stdout, "added 2\n")
	}
	if stderr != "" {
		t.Fatalf("add second todo stderr = %q, want empty", stderr)
	}

	stdout, stderr, err = runTodo(t, todo, database, "list")
	if err != nil {
		t.Fatalf("list todos: %v\nstderr: %s", err, stderr)
	}
	if stdout != "1\tactive\tBuy milk\n2\tactive\tWalk dog\n" {
		t.Fatalf("list stdout = %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("list stderr = %q, want empty", stderr)
	}
}

func TestAddRejectsEmptyTitleWithoutChangingDatabase(t *testing.T) {
	todo := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.db")

	if _, stderr, err := runTodo(t, todo, database, "add", "Keep me"); err != nil {
		t.Fatalf("seed todo: %v\nstderr: %s", err, stderr)
	}

	stdout, stderr, err := runTodo(t, todo, database, "add", " \t ")
	if err == nil {
		t.Fatal("empty title succeeded, want non-zero exit")
	}
	if stdout != "" {
		t.Fatalf("empty title stdout = %q, want empty", stdout)
	}
	if stderr != "title must not be empty\n" {
		t.Fatalf("empty title stderr = %q, want %q", stderr, "title must not be empty\n")
	}

	stdout, stderr, err = runTodo(t, todo, database, "list")
	if err != nil {
		t.Fatalf("list after rejected add: %v\nstderr: %s", err, stderr)
	}
	if stdout != "1\tactive\tKeep me\n" {
		t.Fatalf("database changed after rejected add; list stdout = %q", stdout)
	}
}

func TestDoneCompletesTodoAndIsIdempotent(t *testing.T) {
	todo := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.db")

	if _, stderr, err := runTodo(t, todo, database, "add", "Buy milk"); err != nil {
		t.Fatalf("seed todo: %v\nstderr: %s", err, stderr)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		stdout, stderr, err := runTodo(t, todo, database, "done", "1")
		if err != nil {
			t.Fatalf("complete todo (attempt %d): %v\nstderr: %s", attempt, err, stderr)
		}
		if stdout != "completed 1\n" {
			t.Fatalf("complete todo (attempt %d) stdout = %q, want %q", attempt, stdout, "completed 1\n")
		}
		if stderr != "" {
			t.Fatalf("complete todo (attempt %d) stderr = %q, want empty", attempt, stderr)
		}
	}

	stdout, stderr, err := runTodo(t, todo, database, "list")
	if err != nil {
		t.Fatalf("list completed todo: %v\nstderr: %s", err, stderr)
	}
	if stdout != "1\tdone\tBuy milk\n" {
		t.Fatalf("list completed todo stdout = %q, want %q", stdout, "1\tdone\tBuy milk\n")
	}
}

func TestDoneRejectsInvalidIDsWithoutChangingDatabase(t *testing.T) {
	todo := buildTodo(t)

	tests := []struct {
		name string
		args []string
	}{
		{name: "missing ID", args: []string{"done"}},
		{name: "malformed ID", args: []string{"done", "not-a-number"}},
		{name: "unknown ID", args: []string{"done", "99"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := filepath.Join(t.TempDir(), "todos.db")
			if _, stderr, err := runTodo(t, todo, database, "add", "Keep active"); err != nil {
				t.Fatalf("seed todo: %v\nstderr: %s", err, stderr)
			}
			before, err := os.ReadFile(database)
			if err != nil {
				t.Fatalf("read database before done: %v", err)
			}

			stdout, stderr, err := runTodo(t, todo, database, test.args...)
			if err == nil {
				t.Fatalf("%s succeeded, want non-zero exit", test.name)
			}
			if stdout != "" {
				t.Fatalf("%s stdout = %q, want empty", test.name, stdout)
			}
			if stderr == "" {
				t.Fatalf("%s stderr is empty, want useful message", test.name)
			}
			after, err := os.ReadFile(database)
			if err != nil {
				t.Fatalf("read database after done: %v", err)
			}
			if !bytes.Equal(after, before) {
				t.Fatalf("%s changed database: before %q, after %q", test.name, before, after)
			}
		})
	}
}

func TestListFiltersByStatusAndRejectsUnsupportedStatus(t *testing.T) {
	todo := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.db")

	for _, title := range []string{"First", "Second", "Third"} {
		if _, stderr, err := runTodo(t, todo, database, "add", title); err != nil {
			t.Fatalf("add %q: %v\nstderr: %s", title, err, stderr)
		}
	}
	if _, stderr, err := runTodo(t, todo, database, "done", "2"); err != nil {
		t.Fatalf("complete second todo: %v\nstderr: %s", err, stderr)
	}

	stdout, stderr, err := runTodo(t, todo, database, "list", "--status", "active")
	if err != nil {
		t.Fatalf("list active todos: %v\nstderr: %s", err, stderr)
	}
	if stdout != "1\tactive\tFirst\n3\tactive\tThird\n" {
		t.Fatalf("list active stdout = %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("list active stderr = %q, want empty", stderr)
	}

	stdout, stderr, err = runTodo(t, todo, database, "list", "--status", "done")
	if err != nil {
		t.Fatalf("list done todos: %v\nstderr: %s", err, stderr)
	}
	if stdout != "2\tdone\tSecond\n" {
		t.Fatalf("list done stdout = %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("list done stderr = %q, want empty", stderr)
	}

	before, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("read database before unsupported status: %v", err)
	}
	stdout, stderr, err = runTodo(t, todo, database, "list", "--status", "blocked")
	if err == nil {
		t.Fatal("unsupported status succeeded, want non-zero exit")
	}
	if stdout != "" {
		t.Fatalf("unsupported status stdout = %q, want empty", stdout)
	}
	if stderr == "" {
		t.Fatal("unsupported status stderr is empty, want useful message")
	}
	after, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("read database after unsupported status: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("unsupported status changed database: before %q, after %q", before, after)
	}
}

func TestVersionStillPrintsSeedVersion(t *testing.T) {
	todo := buildTodo(t)

	stdout, stderr, err := runTodo(t, todo, filepath.Join(t.TempDir(), "todos.db"), "version")
	if err != nil {
		t.Fatalf("version: %v\nstderr: %s", err, stderr)
	}
	if stdout != "todo-bench seed\n" {
		t.Fatalf("version stdout = %q, want %q", stdout, "todo-bench seed\n")
	}
	if stderr != "" {
		t.Fatalf("version stderr = %q, want empty", stderr)
	}
}

func buildTodo(t *testing.T) string {
	t.Helper()

	name := "todo"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(t.TempDir(), name)
	cmd := exec.Command("go", "build", "-o", path, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build todo: %v\n%s", err, output)
	}
	return path
}

func runTodo(t *testing.T, todo, database string, args ...string) (string, string, error) {
	t.Helper()

	cmd := exec.Command(todo, args...)
	cmd.Env = append(withoutEnv(os.Environ(), "TODO_DB"), "TODO_DB="+database)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func withoutEnv(environment []string, name string) []string {
	prefix := name + "="
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		if len(entry) < len(prefix) || entry[:len(prefix)] != prefix {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

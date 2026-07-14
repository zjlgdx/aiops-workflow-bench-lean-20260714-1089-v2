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

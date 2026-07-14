package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type todo struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Title  string `json:"title"`
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 1 && args[0] == "version" {
		fmt.Println("todo-bench seed")
		return 0
	}

	switch {
	case len(args) == 2 && args[0] == "add":
		return addTodo(args[1])
	case len(args) == 1 && args[0] == "list":
		return listTodos()
	default:
		fmt.Fprintln(os.Stderr, "usage: todo <add|list|done>")
		return 2
	}
}

func addTodo(title string) int {
	title = strings.TrimSpace(title)
	if title == "" {
		fmt.Fprintln(os.Stderr, "title must not be empty")
		return 1
	}

	path, ok := databasePath()
	if !ok {
		return 1
	}
	todos, err := loadTodos(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	nextID := 1
	for _, item := range todos {
		if item.ID >= nextID {
			nextID = item.ID + 1
		}
	}
	todos = append(todos, todo{ID: nextID, Status: "active", Title: title})
	if err := saveTodos(path, todos); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("added %d\n", nextID)
	return 0
}

func listTodos() int {
	path, ok := databasePath()
	if !ok {
		return 1
	}
	todos, err := loadTodos(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	sort.Slice(todos, func(i, j int) bool {
		return todos[i].ID < todos[j].ID
	})
	for _, item := range todos {
		fmt.Printf("%d\t%s\t%s\n", item.ID, item.Status, item.Title)
	}
	return 0
}

func databasePath() (string, bool) {
	path := os.Getenv("TODO_DB")
	if path == "" {
		fmt.Fprintln(os.Stderr, "TODO_DB must be set")
		return "", false
	}
	return path, true
}

func loadTodos(path string) ([]todo, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var todos []todo
	if err := json.Unmarshal(data, &todos); err != nil {
		return nil, err
	}
	return todos, nil
}

func saveTodos(path string, todos []todo) error {
	data, err := json.Marshal(todos)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

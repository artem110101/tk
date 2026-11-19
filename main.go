package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const usage = `tk [--json] COMMAND
  inbox TEXT                 capture without filing
  project TITLE [key=value ...]
  add TITLE [key=value ...]  create a task; parent=ID nests it
  list [VIEW]                open items by default; no command also lists
  tags                       list each tag and its unfinished task count
  show ID                    full item, including notes (JSON)
  set ID key=value ...       edit or file an inbox item in place
  note ID TEXT               append context
  done ID                    shorthand for set ID state=done
Views: open, all, inbox, project, next, blocked, delegated, done, tag=NAME, or a parent ID.
Keys: title, kind (file inbox as task/project), parent, priority (1=high to 3=low),
      due, when (YYYY-MM-DD), state (open/next/blocked/delegated/done),
      tags (comma-separated names; tags= clears).
Use - for text/title from stdin on creation or note. TK_FILE selects the store.`

func apply(tx *sql.Tx, t *item, fields []string) error {
	originalKind, originalTitle := t.Kind, t.Title
	for _, field := range fields {
		k, v, ok := strings.Cut(field, "=")
		if !ok {
			return fmt.Errorf("expected key=value, got %q", field)
		}
		switch k {
		case "title":
			t.Title = v
		case "kind":
			if t.Kind != "inbox" || (v != "task" && v != "project") {
				return errors.New("kind can only file an inbox item as task or project")
			}
			t.Kind = v
		case "state":
			if !slices.Contains([]string{"open", "next", "blocked", "delegated", "done"}, v) {
				return fmt.Errorf("invalid state: %q", v)
			}
			t.State = v
		case "tags":
			tags, err := parseTags(v)
			if err != nil {
				return err
			}
			t.Tags = tags
		case "priority":
			if v != "" && v != "1" && v != "2" && v != "3" {
				return errors.New("priority must be 1, 2, 3, or empty")
			}
			t.Priority, _ = strconv.Atoi(v)
		case "due", "when":
			if _, err := time.Parse(time.DateOnly, v); v != "" && err != nil {
				return fmt.Errorf("%s must be YYYY-MM-DD or empty", k)
			}
			if k == "due" {
				t.Due = v
			} else {
				t.When = v
			}
		case "parent":
			t.Parent = 0
			if v != "" {
				id, err := strconv.Atoi(v)
				if err != nil || id < 1 {
					return fmt.Errorf("invalid parent ID: %q", v)
				}
				t.Parent = id
			}
		default:
			return fmt.Errorf("unknown key: %q", k)
		}
	}
	if strings.TrimSpace(t.Title) == "" || !utf8.ValidString(t.Title) {
		return errors.New("title must be nonblank UTF-8 text")
	}
	if t.Parent != 0 && t.Kind != "task" {
		return errors.New("only tasks may have a parent")
	}
	seen := map[int]bool{t.ID: true}
	for id := t.Parent; id != 0; {
		if seen[id] {
			return errors.New("parent would create a cycle")
		}
		seen[id] = true
		var kind string
		err := tx.QueryRow(`SELECT kind,coalesce(parent,0) FROM items WHERE id=?`, id).Scan(&kind, &id)
		if err == sql.ErrNoRows || kind == "inbox" {
			return errors.New("parent must be an existing task or project")
		}
		if err != nil {
			return err
		}
	}
	if originalKind == "inbox" && t.Kind != "inbox" && t.Title != originalTitle {
		t.Notes = append([]string{originalTitle}, t.Notes...)
	}
	return nil
}

func (t item) summary() string {
	line, _, _ := strings.Cut(t.Title, "\n")
	s := fmt.Sprintf("%d\t%s/%s\t%q", t.ID, t.Kind, t.State, line)
	if t.Parent != 0 {
		s += fmt.Sprintf(" parent=%d", t.Parent)
	}
	if t.Priority != 0 {
		s += fmt.Sprintf(" priority=%d", t.Priority)
	}
	if t.When != "" {
		s += " when=" + t.When
	}
	if t.Due != "" {
		s += " due=" + t.Due
	}
	if len(t.Tags) != 0 {
		s += " tags=" + strings.Join(t.Tags, ",")
	}
	return s
}

// Minimum and maximum positional arguments; -1 means no upper bound.
var arity = map[string][2]int{
	"list": {0, 1}, "tags": {0, 0}, "show": {1, 1}, "set": {2, -1}, "done": {1, 1},
	"note": {2, 2}, "inbox": {1, 1}, "add": {1, -1}, "project": {1, -1},
}

func run(args []string, in io.Reader, out io.Writer) error {
	machine := len(args) > 0 && args[0] == "--json"
	if machine {
		args = args[1:]
	}
	if len(args) == 0 {
		args = []string{"list"}
	}
	cmd, args := args[0], args[1:]
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		_, err := fmt.Fprintln(out, usage)
		return err
	}
	bounds, ok := arity[cmd]
	if !ok {
		return fmt.Errorf("unknown command %q; see tk help", cmd)
	}
	if len(args) < bounds[0] || (bounds[1] >= 0 && len(args) > bounds[1]) {
		return fmt.Errorf("invalid arguments for %s; see tk help", cmd)
	}
	p, err := path()
	if err != nil {
		return err
	}
	var text string
	if cmd == "add" || cmd == "project" || cmd == "inbox" || cmd == "note" {
		pos := 0
		if cmd == "note" {
			pos = 1
		}
		text = args[pos]
		if text == "-" {
			b, err := io.ReadAll(in)
			if err != nil {
				return err
			}
			text = string(b)
		}
		if strings.TrimSpace(text) == "" || !utf8.ValidString(text) {
			return errors.New("expected nonblank UTF-8 text")
		}
	}
	write := cmd != "list" && cmd != "show" && cmd != "tags"
	db, err := openStore(p, write)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := schema(tx, write); err != nil {
		return err
	}
	var t item
	var items []item
	var counts []tagCount
	switch cmd {
	case "tags":
		counts, err = tagCounts(tx)
	case "list":
		view := "open"
		if len(args) > 0 {
			view = args[0]
		}
		items, err = list(tx, view)
	case "add", "project", "inbox":
		kind := cmd
		if cmd == "add" {
			kind = "task"
		}
		t = item{Kind: kind, State: "open", Title: text}
		err = apply(tx, &t, args[1:])
	default:
		t, err = get(tx, args[0])
		if err != nil {
			return err
		}
		switch cmd {
		case "note":
			t.Notes = append(t.Notes, text)
		case "done":
			t.State = "done"
		case "set":
			err = apply(tx, &t, args[1:])
		}
	}
	if err != nil {
		return err
	}
	if write {
		if err := save(tx, &t); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	if cmd == "tags" {
		if machine {
			return enc.Encode(counts)
		}
		for _, c := range counts {
			if _, err := fmt.Fprintf(out, "%s\t%d\n", c.Tag, c.Count); err != nil {
				return err
			}
		}
		return nil
	}
	if cmd == "list" {
		if machine {
			return enc.Encode(items)
		}
		for _, it := range items {
			if _, err := fmt.Fprintln(out, it.summary()); err != nil {
				return err
			}
		}
		return nil
	}
	if machine || cmd == "show" {
		return enc.Encode(t)
	}
	_, err = fmt.Fprintln(out, t.ID)
	return err
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "tk:", err)
		os.Exit(1)
	}
}

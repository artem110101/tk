package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Exercise the same argv, stdin, stdout, and on-disk file as the CLI.
func setup(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "nested", "work #%25.db")
	t.Setenv("TK_FILE", p)
	return p
}

func call(t *testing.T, input string, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	if err := run(args, strings.NewReader(input), &out); err != nil {
		t.Fatalf("%q: %v", args, err)
	}
	return out.String()
}

func fileBytes(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestWorkflow(t *testing.T) {
	p := setup(t)
	if got := call(t, "", "--json"); got != "[]\n" {
		t.Fatal(got)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("reading an empty store should not create it")
	}
	blob := "Follow up with Zoë\nhttps://example.org/a?x=y\n"
	if got := call(t, blob, "inbox", "-"); got != "1\n" {
		t.Fatal(got)
	}
	call(t, "", "project", "Release")
	call(t, "", "add", "Checklist", "parent=2")
	call(t, "", "set", "1", "title=Schedule review", "parent=3", "kind=task", "state=next", "due=2028-02-29", "when=2028-02-28", "priority=1")
	note := "line 1\n\n  line 2\n"
	call(t, note, "note", "1", "-")
	got := call(t, "", "--json", "note", "1", "final note")
	if got != call(t, "", "show", "1") {
		t.Fatal("mutation JSON should be the persisted item")
	}
	want := item{ID: 1, Kind: "task", Title: "Schedule review", State: "next", Parent: 3, Priority: 1, Due: "2028-02-29", When: "2028-02-28", Notes: []string{blob, note, "final note"}}
	var actual item
	if err := json.Unmarshal([]byte(got), &actual); err != nil || !reflect.DeepEqual(want, actual) {
		t.Fatalf("unexpected item: %s", got)
	}
	for _, view := range []string{"3", "next"} {
		if got := call(t, "", "list", view); !strings.HasPrefix(got, "1\ttask/next") || strings.Count(got, "\n") != 1 {
			t.Fatal(got)
		}
	}
	call(t, "", "done", "3")
	if got := call(t, "", "list", "done"); !strings.HasPrefix(got, "3\ttask/done") || strings.Count(call(t, ""), "\n") != 2 {
		t.Fatal("completion should not cascade")
	}
	call(t, "", "set", "1", "parent=", "priority=", "when=", "due=", "state=delegated")
	got = call(t, "", "--json", "list", "delegated")
	if strings.Contains(got, `"parent":`) || strings.Contains(got, `"priority":`) || strings.Contains(got, `"due":`) || strings.Contains(got, `"when":`) || !strings.Contains(got, `"id":1`) {
		t.Fatal(got)
	}
}

func TestMoveTaskBranch(t *testing.T) {
	setup(t)
	call(t, "", "project", "Office move")
	call(t, "", "add", "Move the network")
	call(t, "", "add", "Pack the router", "parent=2")
	call(t, "", "add", "Label the cables", "parent=3")
	call(t, "", "add", "Submit expenses")

	// Only the top task's parent changes; descendant IDs and links stay put.
	check := func(branchParent int) {
		t.Helper()
		var items []item
		if err := json.Unmarshal([]byte(call(t, "", "--json", "list", "all")), &items); err != nil {
			t.Fatal(err)
		}
		parents := map[int]int{}
		for _, it := range items {
			parents[it.ID] = it.Parent
		}
		if !reflect.DeepEqual(parents, map[int]int{1: 0, 2: branchParent, 3: 2, 4: 3, 5: 0}) {
			t.Fatalf("unexpected outline: %v", parents)
		}
	}
	check(0)
	call(t, "", "set", "2", "parent=1")
	check(1)
	if err := run([]string{"set", "2", "parent=4"}, nil, io.Discard); err == nil {
		t.Fatal("accepted moving a branch under its own grandchild")
	}
	check(1)
	call(t, "", "set", "2", "parent=")
	check(0)
}

func TestRejectedCommandsDoNotPersist(t *testing.T) {
	setup(t)
	call(t, "", "add", "root")
	call(t, "", "add", "child", "parent=1")
	call(t, "", "inbox", "blob")
	before := call(t, "", "--json", "list", "all")
	for _, args := range [][]string{
		{"set", "1", "title=partial", "parent=2"}, {"set", "2", "parent=2"}, {"set", "1", "parent=3"},
		{"add", "orphan", "parent=42"}, {"project", "nested", "parent=1"}, {"set", "1", "kind=project"},
		{"set", "1", "due=2027-02-29"}, {"set", "1", "when=now"}, {"set", "1", "state=later"},
		{"add", "bad", "priority=4"}, {"add", "bad", "other=a"}, {"add", "bad", "oops"}, {"done", "0"},
		{"note", "1"}, {"note", "1", " "}, {"note", "1", "a", "b"}, {"set", "1"}, {"set", "1", "title="},
		{"set", "1", "title=partial", "tags=a,,b"}, {"add", "bad", "tags=two words"}, {"list", "tag="}, {"tags", "extra"},
		{"inbox"}, {"inbox", "-"}, {"inbox", "a", "b"}, {"list", "unknown"}, {"rm", "1"},
	} {
		var out bytes.Buffer
		if err := run(args, strings.NewReader(""), &out); err == nil || out.Len() != 0 {
			t.Fatalf("unexpected success/output: %q", args)
		}
		if call(t, "", "--json", "list", "all") != before {
			t.Fatalf("%q changed store on failure", args)
		}
	}
}

func TestRefuseUnrelatedStores(t *testing.T) {
	p := setup(t)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	for _, bad := range []string{"not a database", `{"version":1,"items":[]}`} {
		if err := os.WriteFile(p, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		if err := run([]string{"inbox", "new"}, nil, &out); err == nil || string(fileBytes(t, p)) != bad {
			t.Fatal("did not reject/preserve invalid store")
		}
	}
	for _, ddl := range []string{`CREATE TABLE other (id INTEGER)`, `PRAGMA application_id=0x544b; PRAGMA user_version=999`} {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		db, err := openStore(p, true)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(ddl)
		db.Close()
		if err != nil {
			t.Fatal(err)
		}
		before := fileBytes(t, p)
		if err := run([]string{"inbox", "new"}, nil, &out); err == nil || !bytes.Equal(before, fileBytes(t, p)) {
			t.Fatal("did not reject/preserve foreign database")
		}
	}
}

func TestRolledBackFirstWrite(t *testing.T) {
	setup(t)
	if err := run([]string{"add", "oops", "parent=42"}, nil, io.Discard); err == nil {
		t.Fatal("missing parent accepted")
	}
	if got := call(t, "", "--json"); got != "[]\n" {
		t.Fatal(got)
	}
	if got := call(t, "", "inbox", "retry"); got != "1\n" {
		t.Fatal(got)
	}
}

func TestConcurrentNotes(t *testing.T) {
	p := setup(t)
	call(t, "", "add", "shared task")
	if !bytes.HasPrefix(fileBytes(t, p), []byte("SQLite format 3\x00")) {
		t.Fatal("not a SQLite database")
	}
	const writers = 8
	results := make(chan error, writers)
	for i := range writers {
		go func() {
			results <- run([]string{"note", "1", fmt.Sprint(i)}, nil, io.Discard)
		}()
	}
	for range writers {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
	var got item
	if err := json.Unmarshal([]byte(call(t, "", "show", "1")), &got); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, n := range got.Notes {
		seen[n] = true
	}
	if len(seen) != writers || len(got.Notes) != writers {
		t.Fatalf("lost or duplicated a note: %v", got.Notes)
	}
}

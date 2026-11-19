package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type item struct {
	ID       int      `json:"id"`
	Kind     string   `json:"kind"`
	Title    string   `json:"title"`
	State    string   `json:"state"`
	Parent   int      `json:"parent,omitempty"`
	Priority int      `json:"priority,omitempty"`
	Due      string   `json:"due,omitempty"`
	When     string   `json:"when,omitempty"`
	Notes    []string `json:"notes,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

func path() (string, error) {
	if p := os.Getenv("TK_FILE"); p != "" {
		return filepath.Abs(p)
	}
	dir, err := os.UserConfigDir()
	return filepath.Join(dir, "tk", "tasks.db"), err
}

func openStore(p string, write bool) (*sql.DB, error) {
	u := url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(filepath.ToSlash(p), "/"), RawQuery: "mode=ro&_busy_timeout=3000"}
	if write {
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return nil, err
		}
		// Create privately on Unix; don't rely on SQLite's default file mode.
		f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
		u.RawQuery = "mode=rw&_busy_timeout=3000&_foreign_keys=on&_txlock=immediate"
	} else if _, err := os.Stat(p); os.IsNotExist(err) {
		// An absent store reads as empty without touching the filesystem.
		return sql.Open("sqlite3", ":memory:")
	}
	return sql.Open("sqlite3", u.String())
}

// Called in the command's transaction. Never adopt a populated foreign database.
func schema(tx *sql.Tx, write bool) error {
	var app, version int
	if err := tx.QueryRow(`SELECT application_id, user_version FROM pragma_application_id, pragma_user_version`).Scan(&app, &version); err != nil {
		return err
	}
	const tagColumn = `tags TEXT NOT NULL DEFAULT '[]' CHECK(json_type(tags)='array')`
	if app == 0x544b {
		switch version {
		case 2:
			return nil
		case 1:
			// Read legacy stores without changing the file. Migrate on the first write.
			query := `CREATE TEMP VIEW items AS SELECT *, '[]' AS tags FROM main.items`
			if write {
				query = `ALTER TABLE items ADD COLUMN ` + tagColumn + `; PRAGMA user_version=2`
			}
			_, err := tx.Exec(query)
			return err
		}
	}
	var tables int
	if err := tx.QueryRow(`SELECT count(*) FROM sqlite_schema`).Scan(&tables); err != nil {
		return err
	}
	if app != 0 || version != 0 || tables != 0 {
		return errors.New("not a tk database or unsupported schema version")
	}
	// A rolled-back first write can leave an empty file. Read it as empty
	// without initializing on disk or requiring write permission.
	create, stamp := "CREATE TEMP", ""
	if write {
		create, stamp = "CREATE", "; PRAGMA application_id=0x544b; PRAGMA user_version=2"
	}
	_, err := tx.Exec(create + ` TABLE items (
		id INTEGER PRIMARY KEY, kind TEXT NOT NULL, title TEXT NOT NULL, state TEXT NOT NULL,
		parent INTEGER REFERENCES items(id), priority INTEGER NOT NULL, due TEXT NOT NULL,
		planned TEXT NOT NULL, notes TEXT NOT NULL CHECK(json_type(notes)='array'),
		` + tagColumn + `, CHECK (parent IS NULL OR kind='task')
	)` + stamp)
	return err
}

const selectItems = `SELECT id,kind,title,state,coalesce(parent,0),priority,due,planned,notes,tags FROM items`

func scan(row interface{ Scan(...any) error }) (item, error) {
	var t item
	var notes, tags string
	err := row.Scan(&t.ID, &t.Kind, &t.Title, &t.State, &t.Parent, &t.Priority, &t.Due, &t.When, &notes, &tags)
	if err == nil {
		err = json.Unmarshal([]byte(notes), &t.Notes)
	}
	if err == nil {
		err = json.Unmarshal([]byte(tags), &t.Tags)
	}
	return t, err
}

func get(tx *sql.Tx, raw string) (item, error) {
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 {
		return item{}, fmt.Errorf("invalid ID: %q", raw)
	}
	t, err := scan(tx.QueryRow(selectItems+` WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return t, fmt.Errorf("unknown ID: %d", id)
	}
	return t, err
}

func stringArray(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ss) // A slice of strings always encodes.
	return string(b)
}

func save(tx *sql.Tx, t *item) error {
	const columns = `kind,title,state,parent,priority,due,planned,notes,tags`
	const values = `(?,?,?,nullif(?,0),?,?,?,?,?)`
	args := []any{t.Kind, t.Title, t.State, t.Parent, t.Priority, t.Due, t.When, stringArray(t.Notes), stringArray(t.Tags)}
	if t.ID == 0 {
		return tx.QueryRow(`INSERT INTO items (`+columns+`) VALUES `+values+` RETURNING id`, args...).Scan(&t.ID)
	}
	_, err := tx.Exec(`UPDATE items SET (`+columns+`) = `+values+` WHERE id=?`, append(args, t.ID)...)
	return err
}

func list(tx *sql.Tx, view string) ([]item, error) {
	where := `state!='done'`
	var args []any
	switch view {
	case "open":
	case "all":
		where = "1"
	case "inbox", "project":
		where += ` AND kind=?`
		args = append(args, view)
	case "next", "delegated", "blocked", "done":
		where = `state=?`
		args = append(args, view)
	default:
		if raw, ok := strings.CutPrefix(view, "tag="); ok {
			tag, err := tagName(raw)
			if err != nil {
				return nil, err
			}
			where += ` AND kind='task' AND EXISTS (SELECT 1 FROM json_each(items.tags) WHERE value=?)`
			args = append(args, tag)
			break
		}
		p, err := get(tx, view)
		if err != nil {
			return nil, err
		}
		where += ` AND parent=?`
		args = append(args, p.ID)
	}
	rows, err := tx.Query(selectItems+` WHERE `+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []item{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestTags(t *testing.T) {
	p := setup(t)
	if got := call(t, "", "--json", "tags"); got != "[]\n" {
		t.Fatal(got)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("tags created a store on read")
	}
	call(t, "", "project", "Release", "tags=team,ops")
	call(t, "", "inbox", "blob")
	call(t, "", "set", "2", "tags=ops")
	call(t, "", "add", "Review", "tags= OPS,finance,ops ", "parent=1", "state=next")
	call(t, "", "add", "Unscheduled", "tags=ops")
	call(t, "", "add", "Waiting", "tags=ops", "state=blocked")
	call(t, "", "add", "Assigned", "tags=ops", "state=delegated")
	call(t, "", "add", "Finished", "tags=ops,old", "state=done")
	call(t, "", "add", "Untagged subtask", "parent=3")
	if got := call(t, "", "tags"); got != "finance\t1\nold\t0\nops\t4\nteam\t0\n" {
		t.Fatal(got)
	}
	var tasks []item
	if err := json.Unmarshal([]byte(call(t, "", "--json", "list", "tag=Ops")), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 4 || !reflect.DeepEqual(tasks[0].Tags, []string{"finance", "ops"}) {
		t.Fatalf("wrong tasks or tags: %+v", tasks)
	}
	for i, it := range tasks {
		if it.ID != 3+i {
			t.Fatalf("unexpected tag match: %d", it.ID)
		}
	}
	call(t, "", "set", "3", "tags=ops") // finance disappears when no item uses it
	call(t, "", "set", "4", "tags=")
	call(t, "", "done", "5")
	call(t, "", "set", "7", "state=open")
	if got := call(t, "", "--json", "tags"); got != "[{\"tag\":\"old\",\"count\":1},{\"tag\":\"ops\",\"count\":3},{\"tag\":\"team\",\"count\":0}]\n" {
		t.Fatal(got)
	}
}

func TestTagsOnLegacyDatabase(t *testing.T) {
	p := setup(t)
	db, err := openStore(p, true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE items(id INTEGER PRIMARY KEY,kind TEXT,title TEXT,state TEXT,parent INTEGER,priority INTEGER,due TEXT,planned TEXT,notes TEXT);
		INSERT INTO items VALUES (1,'task','Existing','open',NULL,1,'','','["context"]');
		PRAGMA application_id=0x544b; PRAGMA user_version=1;`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	before := fileBytes(t, p)
	old := call(t, "", "show", "1")
	if call(t, "", "--json", "tags") != "[]\n" || !bytes.Equal(before, fileBytes(t, p)) {
		t.Fatal("legacy reads should see no tags and not modify the file")
	}
	call(t, "", "set", "1", "tags=ops")
	if got := call(t, "", "tags"); got != "ops\t1\n" {
		t.Fatal(got)
	}
	call(t, "", "set", "1", "tags=")
	if call(t, "", "show", "1") != old {
		t.Fatal("existing item data changed during migration")
	}
}

package main

import (
	"database/sql"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var validTag = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func tagName(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if !validTag.MatchString(name) {
		return "", fmt.Errorf("invalid tag %q: use letters a-z, digits, underscores or hyphens; start with a letter or digit", name)
	}
	return name, nil
}

func parseTags(value string) ([]string, error) {
	if value == "" {
		return nil, nil
	}
	tags := strings.Split(value, ",")
	for i, s := range tags {
		name, err := tagName(s)
		if err != nil {
			return nil, err
		}
		tags[i] = name
	}
	slices.Sort(tags)
	return slices.Compact(tags), nil
}

type tagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func tagCounts(tx *sql.Tx) ([]tagCount, error) {
	rows, err := tx.Query(`SELECT tag.value, count(*) FILTER (WHERE kind='task' AND state!='done')
		FROM items, json_each(items.tags) AS tag GROUP BY tag.value ORDER BY tag.value`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := []tagCount{}
	for rows.Next() {
		var c tagCount
		if err := rows.Scan(&c.Tag, &c.Count); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}

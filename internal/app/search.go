package app

import (
	"database/sql"
	"strconv"
	"time"
)

// FTS payload columns are not ordinary indexes. Keep an indexed identity map so
// updating one streamed message never scans every archived document.
func searchDelete(tx *sql.Tx, column, value string) error {
	if column != "entity" && column != "session" && column != "memory" {
		return fail("invalid search reference")
	}
	_, err := tx.Exec("DELETE FROM search WHERE rowid IN (SELECT rowid FROM search_refs WHERE "+column+"=?)", value)
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM search_refs WHERE "+column+"=?", value)
	return err
}

func searchInsert(tx *sql.Tx, entity, stamp string, values []any) error {
	parsed, err := time.Parse(time.RFC3339Nano, stamp)
	if err != nil {
		parsed = time.Unix(0, 0)
	}
	seconds := parsed.Unix()
	// Stable second-level chronological order; ties use a deterministic identity
	// suffix, with collision probing. All documents remain pageable after rebuild.
	if seconds > 9000000000 {
		seconds = 9000000000
	}
	if seconds < -9000000000 {
		seconds = -9000000000
	}
	suffix, _ := strconv.ParseUint(hash([]byte(entity))[:15], 16, 64)
	id := seconds*1000000000 + int64(suffix%1000000000)
	for {
		var existing string
		err = tx.QueryRow("SELECT entity FROM search_refs WHERE rowid=?", id).Scan(&existing)
		if err == sql.ErrNoRows {
			break
		}
		if err != nil {
			return err
		}
		id++
	}
	_, err = tx.Exec("INSERT INTO search_refs VALUES(?,?,?,?)", entity, id, values[2], values[3])
	if err != nil {
		return err
	}
	args := append([]any{id}, values...)
	_, err = tx.Exec("INSERT INTO search(rowid,tokens,text,session,memory,message,title,agent,role,time) VALUES(?,?,?,?,?,?,?,?,?,?)", args...)
	return err
}

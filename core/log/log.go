package log

import (
	"database/sql"
)

type TudoLog struct {
	ID        uint32
	TableName string
	RowID     uint32
	CreatedAt string
}

func New(db *sql.DB, table string, rowID uint32) error {
	if _, err := db.Exec("INSERT INTO action_log (id, table_name, row_id, created_at) VALUES (NULL, ?, ?, datetime())", "capture", rowID); err != nil {
		return err
	}
	return nil
}

func Get(db *sql.DB) (TudoLog, bool, error) {
	row := db.QueryRow("SELECT id, table_name, row_id, created_at FROM action_log ORDER BY id DESC LIMIT 1")
	err := row.Err()
	if err != nil {
		return TudoLog{}, false, err
	}

	var a TudoLog
	row.Scan(&a.ID, &a.TableName, &a.RowID, &a.CreatedAt)

	if a.ID <= 0 || a.TableName == "" || a.RowID <= 0 {
		return TudoLog{}, false, nil
	}
	return a, true, nil
}

func Undo(db *sql.DB, table string, id, rowID uint32) error {
	if _, err := db.Exec("UPDATE "+table+" SET done = 0 WHERE id = ?", rowID); err != nil {
		return err
	}
	if _, err := db.Exec("DELETE FROM action_log WHERE id = ?", id); err != nil {
		return err
	}
	return nil
}

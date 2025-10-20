package capture

import (
	"database/sql"
	"errors"
)

type TudoCapture struct {
	ID        uint32
	Content   string
	Done      bool
	CreatedAt string
}

func New(db *sql.DB, captureTxt string) error {
	if _, err := db.Exec("INSERT INTO capture (id, content, done, created_at) VALUES (NULL, ?, 0, date());", captureTxt); err != nil {
		return err
	}
	return nil
}

func IDExists(db *sql.DB, id uint32) (bool, error) {
	row := db.QueryRow("SELECT id FROM capture WHERE id = ?", id)
	err := row.Err()
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func GetActive(db *sql.DB) ([]TudoCapture, error) {
	rows, err := db.Query("SELECT id, content, done, created_at FROM capture WHERE done == 0")
	if err != nil {
		return []TudoCapture{}, err
	}
	defer rows.Close()

	var captureList []TudoCapture
	for rows.Next() {
		var c TudoCapture
		if err := rows.Scan(&c.ID, &c.Content, &c.Done, &c.CreatedAt); err != nil {
			return []TudoCapture{}, err
		}
		captureList = append(captureList, c)
	}

	return captureList, err
}

func Done(db *sql.DB, id uint32) error {
	if _, err := db.Exec("UPDATE capture SET done = 1 WHERE id = ?", id); err != nil {
		return err
	}
	return nil
}

func Count(db *sql.DB) (int, error) {
	row := db.QueryRow("SELECT COUNT(*) FROM capture WHERE done = 0")
	err := row.Err()
	if err != nil {
		return 0, err
	}
	var cnt int
	row.Scan(&cnt)

	return cnt, nil
}

func Clean(db *sql.DB) error {
	if _, err := db.Exec("UPDATE capture SET done = 1 WHERE done = 0"); err != nil {
		return err
	}
	return nil
}

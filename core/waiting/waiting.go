package waiting

import (
	"database/sql"
	"errors"
	"time"
)

type TudoWaiting struct {
	ID         uint32
	Content    string
	Done       bool
	CreatedAt  string
	FinishedAt *string
}

func New(db *sql.DB, content string) error {
	if _, err := db.Exec("INSERT INTO waiting (id, content, done, created_at, finished_at) VALUES (NULL, ?, 0, date(), NULL)", content); err != nil {
		return err
	}
	return nil
}

func IDExists(db *sql.DB, id uint32) (bool, error) {
	row := db.QueryRow("SELECT id, content, done, created_at FROM waiting WHERE id = ? AND done = 0", id)

	var wID uint32
	if err := row.Scan(&wID); errors.Is(err, sql.ErrNoRows) {
		return false, err
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func ContentExists(db *sql.DB, content string) (bool, uint32, error) {
	row := db.QueryRow("SELECT id FROM waiting WHERE content = ? AND done = 0", content)
	var id uint32
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0, nil
	} else if err != nil {
		return false, 0, err
	}

	return true, id, nil
}

func GetActive(db *sql.DB) ([]TudoWaiting, error) {
	rows, err := db.Query("SELECT id, content, done, created_at FROM waiting WHERE done = 0")
	if err != nil {
		return []TudoWaiting{}, err
	}
	defer rows.Close()

	var waitList []TudoWaiting
	for rows.Next() {
		var w TudoWaiting
		if err := rows.Scan(&w.ID, &w.Content, &w.Done, &w.CreatedAt); err != nil {
			return []TudoWaiting{}, err
		}
		waitList = append(waitList, w)
	}

	return waitList, nil
}

func Done(db *sql.DB, id uint32) error {
	if _, err := db.Exec("UPDATE waiting SET done = 1, finished_at = date() WHERE id = ?", id); err != nil {
		return err
	}
	return nil
}

func Review(db *sql.DB, thresh time.Time) (map[time.Time][]TudoWaiting, error) {
	rows, err := db.Query("SELECT id, content, done, created_at, finished_at FROM waiting WHERE done = 1")
	if err != nil {
		return map[time.Time][]TudoWaiting{}, err
	}
	defer rows.Close()

	finishedByDate := make(map[time.Time][]TudoWaiting)
	for rows.Next() {
		var w TudoWaiting
		if err := rows.Scan(&w.ID, &w.Content, &w.Done, &w.CreatedAt, &w.FinishedAt); err != nil {
			return map[time.Time][]TudoWaiting{}, err
		}

		if w.FinishedAt != nil {
			finishTime, err := time.ParseInLocation("2006-01-02", *w.FinishedAt, time.Now().Location())
			if err != nil {
				return map[time.Time][]TudoWaiting{}, err
			}

			if thresh.Before(finishTime) {
				finishedByDate[finishTime] = append(finishedByDate[finishTime], w)
			}
		}
	}

	return finishedByDate, nil
}

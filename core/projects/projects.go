package projects

import (
	"database/sql"
	"errors"
	"time"
)

type TudoProject struct {
	ID         uint32
	Content    string
	Done       bool
	CreatedAt  string
	FinishedAt *string
}

func New(db *sql.DB, content string) error {
	if _, err := db.Exec("INSERT INTO projects (id, content, done, created_at) VALUES (NULL, ?, 0, date())", content); err != nil {
		return err
	}
	return nil
}

func ContentExists(db *sql.DB, content string) (bool, uint32, error) {
	row := db.QueryRow("SELECT id FROM projects WHERE content = ? AND done = 0", content)
	var id uint32
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0, nil
	} else if err != nil {
		return false, 0, err
	}

	return true, id, nil
}

func Get(db *sql.DB, id uint32) (TudoProject, error) {
	row := db.QueryRow("SELECT id, content, done, created_at, finished_at FROM projects WHERE id = ?", id)

	var p TudoProject
	err := row.Scan(&p.ID, &p.Content, &p.Done, &p.CreatedAt, &p.FinishedAt)
	if err != nil {
		return TudoProject{}, err
	}

	return p, nil
}

func GetActive(db *sql.DB) ([]TudoProject, error) {
	rows, err := db.Query("SELECT id, content, done, created_at FROM projects WHERE done = 0")
	if err != nil {
		return []TudoProject{}, err
	}
	defer rows.Close()

	var projects []TudoProject
	for rows.Next() {
		var p TudoProject
		if err := rows.Scan(&p.ID, &p.Content, &p.Done, &p.CreatedAt); err != nil {
			return []TudoProject{}, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func Done(db *sql.DB, id uint32) error {
	if _, err := db.Exec("UPDATE projects SET done = 1, finished_at = date() WHERE id = ? AND done = 0", id); err != nil {
		return err
	}
	return nil
}

func Review(db *sql.DB, thresh time.Time) ([]TudoProject, error) {
	rows, err := db.Query("SELECT id, content, finished_at FROM projects WHERE finished_at IS NOT NULL AND done = 1")
	if err != nil {
		return []TudoProject{}, err
	}
	defer rows.Close()

	var projects []TudoProject
	for rows.Next() {
		var p TudoProject
		if err = rows.Scan(&p.ID, &p.Content, &p.FinishedAt); err != nil {
			return []TudoProject{}, err
		}
		finishTime, err := time.Parse("2006-01-02", *p.FinishedAt)
		if err != nil {
			return []TudoProject{}, err
		}

		if thresh.Before(finishTime) {
			projects = append(projects, p)
		}
	}

	return projects, nil
}

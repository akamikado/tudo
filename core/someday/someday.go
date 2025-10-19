package someday

import (
	"database/sql"
	"errors"
)

type TudoSomeday struct {
	ID        uint32
	Content   string
	Done      bool
	CreatedAt string
}

func New(db *sql.DB, content string) error {
	if _, err := db.Exec("INSERT INTO someday (id, content, done, created_at) VALUES (NULL, ?, 0, date())", content); err != nil {
		return err
	}
	return nil
}

func ContentExists(db *sql.DB, content string) (bool, uint32, error) {
	row := db.QueryRow("SELECT id FROM someday WHERE content = ? AND done = 0", content)
	var task TudoSomeday
	err := row.Scan(&task.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0, nil
	} else if err != nil {
		return false, 0, err
	}

	return true, task.ID, nil
}

func IDExists(db *sql.DB, id uint32) (bool, error) {
	row := db.QueryRow("SELECT id FROM someday WHERE id = ? AND done = 0", id)
	var task TudoSomeday
	err := row.Scan(&task.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func Get(db *sql.DB, id uint32) (TudoSomeday, error) {
	row := db.QueryRow("SELECT id, content, created_at, done FROM someday WHERE id = ? AND done = 0", id)
	var task TudoSomeday
	err := row.Scan(&task.ID, &task.Content, &task.CreatedAt, &task.Done)
	if err != nil {
		return TudoSomeday{}, err
	}
	return task, nil
}

func GetActive(db *sql.DB) ([]TudoSomeday, error) {
	rows, err := db.Query("SELECT id, content, done, created_at FROM someday WHERE done = 0")
	if err != nil {
		return []TudoSomeday{}, err
	}
	defer rows.Close()

	var tasks []TudoSomeday
	for rows.Next() {
		var s TudoSomeday
		if err := rows.Scan(&s.ID, &s.Content, &s.Done, &s.CreatedAt); err != nil {
			return []TudoSomeday{}, err
		}
		tasks = append(tasks, s)
	}

	return tasks, nil
}

func Read(db *sql.DB) ([]TudoSomeday, error) {
	rows, err := db.Query("SELECT id, tasks.content, done, tasks.created_at FROM someday WHERE done = 0 AND content LIKE '%read%'")
	if err != nil {
		return []TudoSomeday{}, err
	}
	defer rows.Close()

	var tasks []TudoSomeday
	for rows.Next() {
		var task TudoSomeday
		if err := rows.Scan(&task.ID, &task.Content, &task.Done, &task.CreatedAt); err != nil {
			return []TudoSomeday{}, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func Done(db *sql.DB, id uint32) error {
	if _, err := db.Exec("UPDATE someday SET done = 1 WHERE id = ?", id); err != nil {
		return err
	}
	return nil
}

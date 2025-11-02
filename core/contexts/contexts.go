package contexts

import (
	"database/sql"
	"errors"
)

type TudoContext struct {
	ID      uint32
	Content string
}

func New(db *sql.DB, content string) error {
	if _, err := db.Exec("INSERT INTO contexts (id, content) VALUES (NULL, ?)", content); err != nil {
		return err
	}
	return nil
}

func ContentExists(db *sql.DB, content string) (bool, uint32, error) {
	row := db.QueryRow("SELECT id FROM contexts WHERE content = ?", content)

	var id uint32
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0, nil
	} else if err != nil {
		return false, 0, err
	}

	return true, id, nil
}

func Get(db *sql.DB, id uint32) (TudoContext, error) {
	row := db.QueryRow("SELECT id, content FROM contexts WHERE id = ?", id)
	var c TudoContext
	err := row.Scan(&c.ID, &c.Content)
	if err != nil {
		return TudoContext{}, err
	}

	return c, nil
}

func GetAll(db *sql.DB) ([]TudoContext, error) {
	rows, err := db.Query("SELECT id, content FROM contexts")
	if err != nil {
		return []TudoContext{}, err
	}
	defer rows.Close()

	var contexts []TudoContext
	for rows.Next() {
		var c TudoContext
		if err := rows.Scan(&c.ID, &c.Content); err != nil {
			return []TudoContext{}, err
		}
		contexts = append(contexts, c)
	}
	return contexts, nil
}

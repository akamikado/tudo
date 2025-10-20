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
	err := row.Err()
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0, nil
	} else if err != nil {
		return false, 0, err
	}

	var id uint32
	row.Scan(&id)

	return true, id, nil
}

func Get(db *sql.DB, id uint32) (TudoContext, error) {
	row := db.QueryRow("SELECT id, content FROM contexts WHERE id = ?", id)
	err := row.Err()
	if err != nil {
		return TudoContext{}, err
	}

	var c TudoContext
	row.Scan(&c.ID, &c.Content)

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

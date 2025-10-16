package database

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

const createTables string = `
CREATE TABLE IF NOT EXISTS capture (
  id INTEGER NOT NULL PRIMARY KEY,
  content TEXT NOT NULL,
  done INTEGER NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS next_actions (
  id INTEGER NOT NULL PRIMARY KEY,
  content TEXT NOT NULL,
  done INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  finished_at TEXT
);

CREATE TABLE IF NOT EXISTS tasks (
  id INTEGER NOT NULL PRIMARY KEY,
  content TEXT NOT NULL,
  project_id INTEGER NOT NULL,
  due TEXT NOT NULL,
  done INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  finished_at TEXT
);

CREATE TABLE IF NOT EXISTS projects (
  id INTEGER NOT NULL PRIMARY KEY,
  content TEXT NOT NULL,
  done INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  finished_at TEXT
);

CREATE TABLE IF NOT EXISTS someday (
  id INTEGER NOT NULL PRIMARY KEY,
  content TEXT NOT NULL,
  done INTEGER NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS waiting (
  id INTEGER NOT NULL PRIMARY KEY,
  content TEXT NOT NULL,
  done INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  finished_at TEXT
);
`

func Setup(dbFile string) error {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec(createTables); err != nil {
		db.Close()
		return err
	}

	return nil
}

func Connect(dbFile string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, err
	}
	return db, nil
}

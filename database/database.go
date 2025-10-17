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

CREATE TABLE IF NOT EXISTS tasks (
  id INTEGER NOT NULL PRIMARY KEY,
  content TEXT NOT NULL,
  project_id INTEGER,
  context TEXT,
  due TEXT ,
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

CREATE TABLE IF NOT EXISTS contexts (
  id INTEGER NOT NULL PRIMARY KEY,
  content TEXT NOT NULL
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

CREATE TABLE IF NOT EXISTS action_log (
  id INTEGER NOT NULL PRIMARY KEY,
  table_name TEXT NOT NULL,
  row_id INTEGER NOT NULL,
  created_at TEXT NOT NULL
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

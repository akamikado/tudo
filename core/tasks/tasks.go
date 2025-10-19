package tasks

import (
	"database/sql"
	"errors"
	"time"
)

type TudoTask struct {
	ID         uint32
	Content    string
	ProjectID  *uint32
	Context    *string
	Due        *string
	Done       bool
	CreatedAt  string
	FinishedAt *string
}

func New(db *sql.DB, content string, projectID *uint32, context *string, due *string) error {
	if _, err := db.Exec("INSERT INTO tasks (id, content, project_id, context, due, done, created_at, finished_at) VALUES (NULL, ?, ?, ?, ?, 0, date(), NULL)", content, projectID, context, due); err != nil {
		return err
	}
	return nil
}

func IDExists(db *sql.DB, id uint32) (bool, error) {
	row := db.QueryRow("SELECT content FROM tasks WHERE id = ?", id)
	err := row.Err()
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func Get(db *sql.DB, id uint32) (TudoTask, error) {
	row := db.QueryRow("SELECT id, content, project_id, context, due, done, created_at, finished_at FROM tasks WHERE id = ?", id)
	err := row.Err()
	if err != nil {
		return TudoTask{}, err
	}
	var task TudoTask
	row.Scan(&task.ID, &task.Content, &task.ProjectID, &task.Context, &task.Due, &task.Done, &task.CreatedAt, &task.FinishedAt)

	return task, nil
}

func GetActiveNextActions(db *sql.DB) ([]TudoTask, error) {
	rows, err := db.Query("SELECT id, content, context FROM tasks WHERE done == 0 AND project_id IS NULL GROUP BY context")
	if err != nil {
		return []TudoTask{}, err
	}
	defer rows.Close()

	var nextActions []TudoTask
	for rows.Next() {
		var action TudoTask
		if err := rows.Scan(&action.ID, &action.Content, &action.Context); err != nil {
			return []TudoTask{}, err
		}
		nextActions = append(nextActions, action)
	}

	return nextActions, nil
}

func GetTodayCalenderTasks(db *sql.DB) ([]TudoTask, error) {
	rows, err := db.Query("SELECT id, content, project_id, context, due, done, created_at, finished_at FROM tasks WHERE done = 0 AND due IS NOT NULL")
	if err != nil {
		return []TudoTask{}, err
	}

	yyyy, mm, dd := time.Now().Date()
	todayDate := time.Date(yyyy, mm, dd, 0, 0, 0, 0, time.Now().Location())
	var tasks []TudoTask
	for rows.Next() {
		var task TudoTask
		if err := rows.Scan(&task.ID, &task.Content, &task.ProjectID, &task.Context, &task.Due, &task.Done, &task.CreatedAt, &task.FinishedAt); err != nil {
			return []TudoTask{}, err
		}
		if task.Due != nil {
			due, err := time.ParseInLocation("2006-01-02", *task.Due, time.Now().Location())
			if err != nil {
				return []TudoTask{}, err
			}
			if todayDate.Equal(due) {
				tasks = append(tasks, task)
			}
		}
	}
	return tasks, nil
}

func GetAllCalenderTasks(db *sql.DB) ([]TudoTask, error) {
	rows, err := db.Query("SELECT id, content, project_id, context, due, done, created_at, finished_at FROM tasks WHERE done = 0 AND due IS NOT NULL")
	if err != nil {
		return []TudoTask{}, err
	}

	var tasks []TudoTask
	for rows.Next() {
		var task TudoTask
		if err := rows.Scan(&task.ID, &task.Content, &task.ProjectID, &task.Context, &task.Due, &task.Done, &task.CreatedAt, &task.FinishedAt); err != nil {
			return []TudoTask{}, err
		}
		if task.Due != nil {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func GetTodayProjectCalendarTasks(db *sql.DB, projectID uint32) ([]TudoTask, error) {
	rows, err := db.Query("SELECT id, content, project_id, context, due, done, created_at, finished_at FROM tasks WHERE done = 0 AND due IS NOT NULL AND project_id = ?", projectID)
	if err != nil {
		return []TudoTask{}, err
	}

	yyyy, mm, dd := time.Now().Date()
	todayDate := time.Date(yyyy, mm, dd, 0, 0, 0, 0, time.Now().Location())
	var tasks []TudoTask
	for rows.Next() {
		var task TudoTask
		if err := rows.Scan(&task.ID, &task.Content, &task.ProjectID, &task.Context, &task.Due, &task.Done, &task.CreatedAt, &task.FinishedAt); err != nil {
			return []TudoTask{}, err
		}
		if task.Due != nil {
			due, err := time.ParseInLocation("2006-01-02", *task.Due, time.Now().Location())
			if err != nil {
				return []TudoTask{}, err
			}
			if todayDate.Equal(due) {
				tasks = append(tasks, task)
			}
		}
	}
	return tasks, nil
}

func GetAllProjectCalendarTasks(db *sql.DB, projectID uint32) ([]TudoTask, error) {
	rows, err := db.Query("SELECT id, content, project_id, context, due, done, created_at, finished_at FROM tasks WHERE done = 0 AND due IS NOT NULL AND project_id = ?", projectID)
	if err != nil {
		return []TudoTask{}, err
	}

	var tasks []TudoTask
	for rows.Next() {
		var task TudoTask
		if err := rows.Scan(&task.ID, &task.Content, &task.ProjectID, &task.Context, &task.Due, &task.Done, &task.CreatedAt, &task.FinishedAt); err != nil {
			return []TudoTask{}, err
		}
		if task.Due != nil {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func GetActiveProjectTasks(db *sql.DB, projectID uint32) ([]TudoTask, error) {
	rows, err := db.Query("SELECT id, content, project_id, context, due, done, created_at, finished_at FROM tasks WHERE done = 0 AND due IS NULL AND project_id = ?", projectID)
	if err != nil {
		return []TudoTask{}, err
	}
	defer rows.Close()

	var tasks []TudoTask
	for rows.Next() {
		var t TudoTask
		if err := rows.Scan(&t.ID, &t.Content, &t.ProjectID, &t.Context, &t.Due, &t.Done, &t.CreatedAt, &t.FinishedAt); err != nil {
			return []TudoTask{}, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func Read(db *sql.DB) ([]TudoTask, error) {
	rows, err := db.Query("SELECT id, content, project_id, context, due, done, created_at, finished_at FROM tasks WHERE done = 0 AND content LIKE '%read%'")
	if err != nil {
		return []TudoTask{}, err
	}
	defer rows.Close()

	var tasks []TudoTask
	for rows.Next() {
		var task TudoTask
		if err := rows.Scan(&task.ID, &task.Content, &task.ProjectID, &task.Context, &task.Due, &task.Done, &task.CreatedAt, &task.FinishedAt); err != nil {
			return []TudoTask{}, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func Done(db *sql.DB, id uint32) error {
	_, err := db.Exec("UPDATE tasks SET done = 1, finished_at = date() WHERE id = ?", id)
	if err != nil {
		return err
	}
	return nil
}

func CountCalendar(db *sql.DB) (int, error) {
	row := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE done = 0 AND due IS NOT NULL")
	err := row.Err()
	if err != nil {
		return 0, err
	}
	var cnt int
	row.Scan(&cnt)
	return cnt, nil
}

func CountNextActions(db *sql.DB) (int, error) {
	row := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE done = 0 AND project_id IS NULL AND due IS NULL")
	err := row.Err()
	if err != nil {
		return 0, err
	}
	var cnt int
	row.Scan(&cnt)
	return cnt, nil
}

func CountProjectTasks(db *sql.DB) (int, error) {
	row := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE done = 0 AND project_id IS NOT NULL AND due IS NULL")
	err := row.Err()
	if err != nil {
		return 0, err
	}
	var cnt int
	row.Scan(&cnt)
	return cnt, nil
}

func CleanCalendar(db *sql.DB) error {
	if _, err := db.Exec("UPDATE tasks SET done = 1, finished_at = date() WHERE done = 0 AND due IS NOT NULL"); err != nil {
		return err
	}
	return nil
}

func CleanNextActions(db *sql.DB) error {
	if _, err := db.Exec("UPDATE tasks SET done = 1, finished_at = date() WHERE done = 0 AND project_id IS NULL AND due IS NULL"); err != nil {
		return err
	}
	return nil
}

func CleanProjectTasks(db *sql.DB) error {
	if _, err := db.Exec("UPDATE tasks SET done = 1, finished_at = date() WHERE done = 0 AND project_id IS NOT NULL AND due IS NULL"); err != nil {
		return err
	}
	return nil
}

func Review(db *sql.DB, thresh time.Time) (map[time.Time][]TudoTask, error) {
	rows, err := db.Query("SELECT id, content, project_id, context, due, done, created_at, finished_at FROM tasks WHERE done = 1")
	if err != nil {
		return map[time.Time][]TudoTask{}, err
	}

	tasksFinishedSinceThreshold := make(map[time.Time][]TudoTask)
	for rows.Next() {
		var task TudoTask
		if err := rows.Scan(&task.ID, &task.Content, &task.ProjectID, &task.Context, &task.Due, &task.Done, &task.CreatedAt, &task.FinishedAt); err != nil {
			return map[time.Time][]TudoTask{}, err
		}
		if task.FinishedAt != nil {
			finishedTime, err := time.ParseInLocation("2006-01-02", *task.FinishedAt, time.Now().Location())
			if err != nil {
				return map[time.Time][]TudoTask{}, err
			}

			if thresh.Before(finishedTime) {
				tasksFinishedSinceThreshold[finishedTime] = append(tasksFinishedSinceThreshold[finishedTime], task)
			}
		}
	}

	return tasksFinishedSinceThreshold, nil
}

func PendingCalendar(db *sql.DB, thresh time.Time) ([]TudoTask, error) {
	rows, err := db.Query("SELECT id, content, project_id, context, due, done, created_at, finished_at FROM tasks WHERE done = 0 AND due IS NOT NULL")
	if err != nil {
		return []TudoTask{}, err
	}

	var pendingTasks []TudoTask
	for rows.Next() {
		var task TudoTask
		if err := rows.Scan(&task.ID, &task.Content, &task.ProjectID, &task.Context, &task.Due, &task.Done, &task.CreatedAt, &task.FinishedAt); err != nil {
			return []TudoTask{}, err
		}

		if task.Due != nil {
			dueTime, err := time.ParseInLocation("2006-01-02", *task.Due, time.Now().Location())
			if err != nil {
				return []TudoTask{}, err
			}

			if thresh.Before(dueTime) {
				pendingTasks = append(pendingTasks, task)
			}
		}
	}

	return pendingTasks, nil
}

package cmd

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"tudo/database"
)

var invalidCommand error = errors.New("Invalid command: ")

var invalidCommandFormat error = errors.New("Invalid command format")

func Todo(s string) {
	fmt.Println("TODO: " + s)
	os.Exit(0)
}

func nonFatalError(err error, args ...string) {
	fmt.Println("Error occurred!!")
	e := err.Error()
	for _, arg := range args {
		e += arg + " "
	}
	fmt.Println(e)
	os.Exit(0)
}

type tudoCapture struct {
	ID        uint32
	Content   string
	Done      bool
	CreatedAt string
}

func (c *tudoCapture) Format() string {
	return fmt.Sprint(c.ID) + "\n" + c.Content + "\n"
}

type tudoNextAction struct {
	ID        uint32
	Content   string
	Done      bool
	CreatedAt string
}

func (a *tudoNextAction) Format() string {
	return fmt.Sprint(a.ID) + "\n" + a.Content + "\n"
}

type tudoProjectAction struct {
	ID        uint32
	Content   string
	Project   uint32
	Due       string
	Done      bool
	CreatedAt string
}

func (a *tudoProjectAction) Format() string {
	return fmt.Sprint(a.ID) + "\n" + a.Content + "\n" + a.Due + "\n"
}

type tudoProject struct {
	ID        uint32
	Content   string
	Done      bool
	CreatedAt string
}

func projectExists(db *sql.DB, content string) (bool, uint32) {
	row := db.QueryRow("SELECT id FROM projects WHERE content = ? AND done = 0", content)
	var id uint32
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0
	} else if err != nil {
		nonFatalError(err)
	}

	return true, id
}

func (p *tudoProject) Format() string {
	return p.Content + "\n"
}

type tudoSomeday struct {
	ID        uint32
	Content   string
	Done      bool
	CreatedAt string
}

func somedayTaskExists(db *sql.DB, content string) (bool, uint32) {
	row := db.QueryRow("SELECT id FROM someday WHERE content = ? AND done = 0", content)
	var id uint32
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0
	} else if err != nil {
		nonFatalError(err)
	}

	return true, id
}

func (s *tudoSomeday) Format() string {
	return s.Content + "\n"
}

func ParseArgs(dbFile string) {
	db, err := database.Connect(dbFile)
	if err != nil {
		fmt.Println("Could not connect to database :" + err.Error())
		os.Exit(1)
	}
	defer db.Close()

	cmdArgs := os.Args
	if len(cmdArgs) == 1 {
		Todo("no arg")
	}
	switch cmdArgs[1] {
	case "help":
		fallthrough
	case "-h":
		fallthrough
	case "--help":
		Todo("help")

	case "new":
		if len(cmdArgs) == 2 {
			fmt.Println("Capture started (<Ctrl-c> on a new line to stop capture)")
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT)
			done := make(chan bool, 1)

			go func() {
				<-sigChan
				done <- true
			}()

			capturing := true
			captureTxt := ""
			scanner := bufio.NewScanner(os.Stdin)
			for capturing {
				select {
				case <-done:
					capturing = false
				default:
					if scanner.Scan() {
						captureTxt = captureTxt + scanner.Text() + "\n"
					} else {
						capturing = false
					}
				}
			}
			if _, err := db.Exec("INSERT INTO capture (id, content, done, created_at) VALUES (NULL, ?, 0, datetime());", captureTxt); err != nil {
				nonFatalError(err)
			}
		} else if len(cmdArgs) > 2 {
			switch cmdArgs[2] {
			case "next":
				fmt.Print("Please enter new next action: ")
				var task string
				fmt.Scan(&task)
				if _, err := db.Exec("INSERT INTO next_actions (id, content, done, created_at) VALUES (NULL, ?, 0, datetime())", task); err != nil {
					nonFatalError(err)
				}
			case "project":
				fmt.Print("Please enter new project name: ")
				var projectName string
				fmt.Scan(&projectName)
				if exists, _ := projectExists(db, projectName); exists {
					fmt.Println("Project `" + projectName + "` already exists")
					return
				}

				if _, err := db.Exec("INSERT INTO projects (id, content, done, created_at) VALUES (NULL, ?, 0, date())", projectName); err != nil {
					nonFatalError(err)
				}

				fmt.Println("Project `" + projectName + "` has been created")
			case "wait":
				Todo("wait")
			case "someday":
				fmt.Print("Please enter new project name: ")
				var futureTask string
				fmt.Scan(&futureTask)
				if exists, _ := somedayTaskExists(db, futureTask); exists {
					fmt.Println("Someday task `" + futureTask + "` already exists")
					return
				}

				if _, err := db.Exec("INSERT INTO someday (id, content, done, created_at) VALUES (NULL, ?, 0, date())", futureTask); err != nil {
					nonFatalError(err)
				}

				fmt.Println("Someday task `" + futureTask + "` has been created")
			default:
				projectName := ""
				for i := 2; i < len(cmdArgs); i++ {
					if i > 2 {
						projectName += " "
					}
					projectName += cmdArgs[i]
				}

				var projectID uint32
				exists, projectID := projectExists(db, projectName)
				if !exists {
					rows, err := db.Query("SELECT id FROM projects WHERE content = ?", projectName)
					if err != nil {
						nonFatalError(err)
					}
					if rows.Next() {
						fmt.Println("No active project `" + projectName + "` exists")
						return
					}
					nonFatalError(invalidCommand, projectName)
				}

				fmt.Print("Please enter new task for the project `" + projectName + "`: ")
				var task string
				fmt.Scanln(&task)
				fmt.Print("Due date (YYYY-MM-DD): ")
				var due string
				fmt.Scanln(&due)
				if _, err := db.Exec("INSERT INTO tasks (id, content, project_id, due, done, created_at) VALUES (NULL, ?, ?, ?, 0, datetime())", task, projectID, due); err != nil {
					nonFatalError(err)
				}
			}
		}

	case "done":
		if len(cmdArgs) < 4 {
			nonFatalError(invalidCommandFormat)
		}
		switch cmdArgs[2] {
		case "in":
			captureID, err := strconv.Atoi(cmdArgs[3])
			if err != nil {
				nonFatalError(invalidCommandFormat)
			}

			r, err := db.Exec("UPDATE capture SET done = 1 WHERE id = ?", captureID)
			if err != nil {
				nonFatalError(err)
			}

			rows, err := r.RowsAffected()
			if err != nil {
				nonFatalError(err)
			}

			if rows == 0 {
				fmt.Println("Capture item `" + cmdArgs[3] + "` does not exist")
			} else {
				fmt.Println("Marked capture item `" + cmdArgs[3] + "` as finished\n")
			}
		case "next":
			actionID, err := strconv.Atoi(cmdArgs[3])
			if err != nil {
				nonFatalError(invalidCommandFormat)
			}
			r, err := db.Exec("UPDATE next_actions SET done = 1 WHERE id = ?", actionID)
			if err != nil {
				nonFatalError(err)
			}

			rows, err := r.RowsAffected()
			if err != nil {
				nonFatalError(err)
			}

			if rows == 0 {
				fmt.Println("Next action " + cmdArgs[3] + " does not exist")
			} else {
				row := db.QueryRow("SELECT content FROM next_actions WHERE id = ?", actionID)
				var finishedTask string
				if err := row.Scan(&finishedTask); err != nil {
					nonFatalError(err)
				}
				fmt.Println("Finished task " + cmdArgs[3] + "\n" + finishedTask)
			}
		case "someday":
			if len(cmdArgs) < 3 {
				nonFatalError(invalidCommandFormat)
			}
			id, err := strconv.Atoi(cmdArgs[3])
			if err != nil {
				nonFatalError(invalidCommandFormat)
			}

			var futureTask tudoSomeday
			row := db.QueryRow("SELECT id, content, done, created_at FROM someday WHERE id = ? AND done = 0", id)
			err = row.Scan(&futureTask.ID, &futureTask.Content, &futureTask.Done, &futureTask.CreatedAt)
			if errors.Is(err, sql.ErrNoRows) {
				nonFatalError(errors.New("Someday task `" + cmdArgs[3] + "` does not exist"))
			}

			if _, err := db.Exec("UPDATE projects SET done = 1 WHERE id = ? AND done = 0", id); err != nil {
				nonFatalError(err)
			}

			fmt.Println("Marked someday task `" + cmdArgs[3] + "` as done")
			fmt.Println("Create someday task as project? (y/n)")
			var ans string
			fmt.Scanln(&ans)
			switch ans {
			case "n":
				return
			case "y":
				fmt.Println("Use someday task as project name? (y/n)")
				fmt.Scanln(&ans)
				switch ans {
				case "n":
					fmt.Print("Please enter new project name: ")
					var projectName string
					fmt.Scan(&projectName)
					if exists, _ := projectExists(db, projectName); exists {
						fmt.Println("Project `" + projectName + "` already exists")
						return
					}

					if _, err := db.Exec("INSERT INTO projects (id, content, done, created_at) VALUES (NULL, ?, 0, date())", projectName); err != nil {
						nonFatalError(err)
					}

					fmt.Println("Project `" + projectName + "` has been created")
				case "y":
					if exists, _ := projectExists(db, futureTask.Content); exists {
						fmt.Println("Project `" + futureTask.Content + "` already exists")
						return
					}

					if _, err := db.Exec("INSERT INTO projects (id, content, done, created_at) VALUES (NULL, ?, 0, datetime())", futureTask.Content); err != nil {
						nonFatalError(err)
					}
				default:
					nonFatalError(invalidCommand)
				}
			default:
				nonFatalError(invalidCommand)
			}
		default:
			projectName := ""
			for i := 2; i < len(cmdArgs)-1; i++ {
				if i > 2 {
					projectName += " "
				}
				projectName += cmdArgs[i]
			}
			if taskID, err := strconv.Atoi(cmdArgs[len(cmdArgs)-1]); err != nil {
				projectName += " " + cmdArgs[len(cmdArgs)-1]

				if exists, _ := projectExists(db, projectName); !exists {
					fmt.Println("No active project `" + projectName + "` exists")
					return
				}

				if _, err := db.Exec("UPDATE projects SET done = 1 WHERE content = ? AND done = 0", projectName); err != nil {
					nonFatalError(err)
				}
				fmt.Println("Finished project `" + projectName + "`\n")
			} else {
				exists, projectID := projectExists(db, projectName)
				if !exists {
					fmt.Println("No active project `" + projectName + "` exists")
					return
				}

				res, err := db.Exec("UPDATE tasks SET done = 1 WHERE id = ? AND project_id = ?", taskID, projectID)
				if err != nil {
					nonFatalError(err)
				}
				updatedRows, err := res.RowsAffected()
				if err != nil {
					nonFatalError(err)
				}
				if updatedRows == 0 {
					fmt.Println("Task `" + cmdArgs[len(cmdArgs)-1] + "` does not exist for project `" + projectName + "`")
					return
				}
				fmt.Println("Task `" + cmdArgs[len(cmdArgs)-1] + "` for project `" + projectName + "` has been marked complete")
			}
		}
	case "in":
		rows, err := db.Query("SELECT id, content FROM capture WHERE done == 0")
		if err != nil {
			nonFatalError(err)
		}
		defer rows.Close()

		var captureList []tudoCapture
		for rows.Next() {
			var c tudoCapture
			if err := rows.Scan(&c.ID, &c.Content); err != nil {
				nonFatalError(err)
			}
			captureList = append(captureList, c)
		}

		if len(captureList) == 0 {
			fmt.Println("No next actions for now")
		}

		for _, c := range captureList {
			fmt.Print(c.Format())
		}
	case "waiting":
		Todo("waiting")
	case "someday":
		rows, err := db.Query("SELECT id, content, done, created_at FROM someday WHERE done = 0")
		if err != nil {
			nonFatalError(err)
		}
		var futureTasks []tudoSomeday
		for rows.Next() {
			var s tudoSomeday
			if err := rows.Scan(&s.ID, &s.Content, &s.Done, &s.CreatedAt); err != nil {
				nonFatalError(err)
			}
			futureTasks = append(futureTasks, s)
		}
		if len(futureTasks) == 0 {
			fmt.Println("No tasks for someday")
		}
		for _, s := range futureTasks {
			fmt.Print("- " + s.Format())
		}
	case "read":
		Todo("read")
	case "next":
		rows, err := db.Query("SELECT id, content FROM next_actions WHERE done == 0")
		if err != nil {
			nonFatalError(err)
		}
		defer rows.Close()

		var nextActions []tudoNextAction
		for rows.Next() {
			var action tudoNextAction
			if err := rows.Scan(&action.ID, &action.Content); err != nil {
				nonFatalError(err)
			}
			nextActions = append(nextActions, action)
		}

		if len(nextActions) == 0 {
			fmt.Println("No next actions for now")
		}

		for _, a := range nextActions {
			fmt.Print(a.Format())
		}

	case "edit":
		Todo("edit")
	case "clean":
		Todo("clean")
	case "undo":
		Todo("undo")
	case "redo":
		Todo("redo")
	case "review":
		Todo("review")
	default:
		if len(cmdArgs) == 2 && cmdArgs[1] == "projects" {
			rows, err := db.Query("SELECT id, content, done, created_at FROM projects WHERE done = 0")
			if err != nil {
				nonFatalError(err)
			}
			var projects []tudoProject
			for rows.Next() {
				var p tudoProject
				if err := rows.Scan(&p.ID, &p.Content, &p.Done, &p.CreatedAt); err != nil {
					nonFatalError(err)
				}
				projects = append(projects, p)
			}
			if len(projects) == 0 {
				fmt.Println("No active projects")
			}
			for _, p := range projects {
				fmt.Print("- " + p.Format())
			}
		} else {
			projectName := ""
			for i := 1; i < len(cmdArgs); i++ {
				if i > 2 {
					projectName += " "
				}
				projectName += cmdArgs[i]
			}
			exists, projectID := projectExists(db, projectName)
			if !exists {
				fmt.Println("No active project `" + projectName + "` exists\n")
			}

			now := time.Now()
			todayDate := now.Format("2006-01-02")

			rows, err := db.Query("SELECT id, content, project_id, due, done, created_at FROM tasks WHERE project_id = ? AND done = 0 AND due = ?", projectID, todayDate)
			if err != nil {
				nonFatalError(err)
			}

			var tasks []tudoProjectAction
			for rows.Next() {
				var t tudoProjectAction
				if err := rows.Scan(&t.ID, &t.Content, &t.Project, &t.Due, &t.Done, &t.CreatedAt); err != nil {
					nonFatalError(err)
				}
				tasks = append(tasks, t)
			}
			if len(tasks) == 0 {
				fmt.Println("No tasks for project `" + projectName + "` today")
			}
			for _, t := range tasks {
				fmt.Print("- " + t.Format())
			}
		}
	}
}

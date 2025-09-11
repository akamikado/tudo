package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"

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
	Task      string
	Done      bool
	CreatedAt string
}

func (a *tudoNextAction) Format() string {
	return fmt.Sprint(a.ID) + "\n" + a.Task + "\n"
}

type tudoProjectAction struct {
	ID        uint32
	Task      string
	Project   uint32
	Due       string
	Done      bool
	CreatedAt string
}

func (a *tudoProjectAction) Format() string {
	return fmt.Sprint(a.ID) + "\n" + a.Task + "\n" + a.Due + "\n"
}

type tudoProject struct {
	ID        uint32
	Name      string
	Done      bool
	CreatedAt string
}

func (p *tudoProject) Format() string {
	return p.Name + "\n"
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
			Todo("new capture")
		} else if len(cmdArgs) > 2 {
			switch cmdArgs[2] {
			case "next":
				fmt.Print("Please enter new next action: ")
				var task string
				fmt.Scan(&task)
				if _, err := db.Exec("INSERT INTO next_actions (id, task, done, created_at) VALUES (NULL, ?, 0, datetime())", task); err != nil {
					nonFatalError(err)
				}
			case "project":
				fmt.Print("Please enter new project name: ")
				var projectName string
				fmt.Scan(&projectName)
				if row := db.QueryRow("SELECT id FROM projects WHERE name = ? AND done = 0", projectName); row != nil {
					fmt.Println("Project `" + projectName + "` already exists")
					return
				}

				if _, err := db.Exec("INSERT INTO projects (id, name, done, created_at) VALUES (NULL, ?, 0, date())", projectName); err != nil {
					nonFatalError(err)
				}

				fmt.Println("Project `" + projectName + "` has been created")
			default:
				projectName := ""
				for i := 2; i < len(cmdArgs); i++ {
					if i > 2 {
						projectName += " "
					}
					projectName += cmdArgs[i]
				}

				row := db.QueryRow("SELECT id FROM projects WHERE name = ? AND WHERE done = 0", projectName)
				if row == nil {
					rows, err := db.Query("SELECT id FROM projects WHERE name = ?", projectName)
					if err != nil {
						nonFatalError(err)
					}
					if rows.Next() {
						fmt.Println("No active project `" + projectName + "` exists")
						return
					}
					nonFatalError(invalidCommand, projectName)
				}
				var projectID uint32
				row.Scan(&projectID)

				fmt.Print("Please enter new task for the project `" + projectName + "`: ")
				var task string
				fmt.Scanln(&task)
				fmt.Print("Due date (YYYY-MM-DD): ")
				var due string
				fmt.Scanln(&due)
				if _, err := db.Exec("INSERT INTO tasks (id, task, project_id, due, done, created_at) VALUES (NULL, ?, ?, ?, 0, datetime())", task, projectID, due); err != nil {
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
			Todo("in done")
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
				row := db.QueryRow("SELECT task FROM next_actions WHERE id = ?", actionID)
				if row == nil {
					nonFatalError(err)
				}

				var finishedTask string
				if err := row.Scan(&finishedTask); err != nil {
					nonFatalError(err)
				}
				fmt.Println("Finished task " + cmdArgs[3] + "\n" + finishedTask)
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
				if row := db.QueryRow("SELECT id FROM projects WHERE name = ? AND done = 0", projectName); row == nil {
					fmt.Println("No active project `" + projectName + "` exists")
					return
				}
				if _, err := db.Exec("UPDATE projects SET done = 1 WHERE name = ? AND done = 0", projectName); err != nil {
					nonFatalError(err)
				}
			} else {
				row := db.QueryRow("SELECT id FROM projects WHERE name = ? AND done = 0", projectName)
				if row == nil {
					fmt.Println("No active project `" + projectName + "` exists")
					return
				}
				var projectID int
				row.Scan(&projectID)

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
		Todo("in")
	case "wait":
		Todo("wait")
	case "waiting":
		Todo("waiting")
	case "someday":
		Todo("someday")
	case "read":
		Todo("read")
	case "next":
		rows, err := db.Query("SELECT id, task FROM next_actions WHERE done == 0")
		if err != nil {
			nonFatalError(err)
		}
		defer rows.Close()

		var nextActions []tudoNextAction
		for rows.Next() {
			var action tudoNextAction
			if err := rows.Scan(&action.ID, &action.Task); err != nil {
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
	case "purge":
		Todo("purge")
	case "undo":
		Todo("undo")
	case "redo":
		Todo("redo")
	case "sql":
		Todo("sql")
	case "review":
		Todo("review")
	case "projects":
		Todo("List active projects")
	default:
		Todo("project")
	}
}

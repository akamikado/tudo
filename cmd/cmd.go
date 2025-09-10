package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"tudo/database"
)

var invalidCommand error = errors.New("Invalid command")

var invalidCommandFormat error = errors.New("Invalid command format")

func Todo(s string) {
	fmt.Println("TODO: " + s)
	os.Exit(0)
}

func nonFatalError(err error) {
	fmt.Println("Error occurred!!")
	fmt.Println(err)
	os.Exit(0)
}

type nextAction struct {
	ID        uint32
	Task      string
	Done      bool
	CreatedAt string
}

func (a *nextAction) Format() string {
	sb := fmt.Sprint(a.ID) + "\n" + a.Task + "\n"
	return sb
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
				fmt.Scanln(&task)
				if _, err := db.Exec("INSERT INTO next_actions (id, task, done, created_at) VALUES (NULL, ?, 0, datetime())", task); err != nil {
					nonFatalError(err)
				}
			case "project":
				fmt.Print("Please enter new project name: ")
				var project string
				fmt.Scan(&project)
				if _, err := db.Exec("INSERT INTO projects (id, project, done, created_at) VALUES (NULL, ?, 0, date())", project); err != nil {
					nonFatalError(err)
				}
			default:
				project := ""
				for i := 2; i < len(cmdArgs); i++ {
					if i > 2 {
						project += " "
					}
					project += cmdArgs[i]
				}

				if row := db.QueryRow("SELECT id FROM projects WHERE project = ?", project); row == nil {
					fmt.Println("Project `" + project + "` does not exist")
					return
				}

				fmt.Print("Please enter new task for the project `" + project + "`: ")
				var task string
				fmt.Scanln(&task)
				fmt.Print("Due date (YYYY-MM-DD): ")
				var due string
				fmt.Scanln(&due)
				if _, err := db.Exec("INSERT INTO tasks (id, task, project, due, done, created_at) VALUES (NULL, ?, ?, ?, 0, datetime())", task, project, due); err != nil {
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
			Todo("done project")
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

		var nextActions []nextAction
		for rows.Next() {
			var action nextAction
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
	default:
		Todo("project")
	}
}

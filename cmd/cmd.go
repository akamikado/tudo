package cmd

import (
	"fmt"
	"os"

	"tudo/database"
)

func Todo(s string) {
	fmt.Println("TODO: " + s)
	os.Exit(0)
}

func nonFatalError(err error) {
	fmt.Println("Error occurred!!")
	fmt.Println(err)
	os.Exit(0)
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
				Todo("next actions new")
			case "project":
				fmt.Print("Please enter new project name: ")
				var project string
				fmt.Scan(&project)
				if _, err := db.Exec("INSERT INTO projects (id, project, done, created_at) VALUES (NULL, ?, 0, date())", project); err != nil {
					nonFatalError(err)
				}
			default:
				Todo("new taks for project")
			}
		}

	case "done":
		Todo("done")
	case "in":
		Todo("in")
	case "waiting":
		Todo("waiting")
	case "someday":
		Todo("someday")
	case "read":
		Todo("read")
	case "next":
		Todo("next")
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
	default:
		Todo("project")
	}
}

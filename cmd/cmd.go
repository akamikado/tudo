package cmd

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	return fmt.Sprint("ID: ", c.ID, "\n", c.Content, "\n")
}

type tudoNextAction struct {
	ID         uint32
	Content    string
	Done       bool
	CreatedAt  string
	FinishedAt string
}

func (a *tudoNextAction) Format() string {
	return fmt.Sprint("ID: ", a.ID, "\n", a.Content, "\n")
}

type tudoProjectAction struct {
	ID         uint32
	Content    string
	Project    uint32
	Due        string
	Done       bool
	CreatedAt  string
	FinishedAt string
}

func (a *tudoProjectAction) Format() string {
	return fmt.Sprint("ID: ", a.ID, "\n", a.Content, "\nDue: ", a.Due, "\n")
}

type tudoProject struct {
	ID         uint32
	Content    string
	Done       bool
	CreatedAt  string
	FinishedAt string
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

type tudoSomedayAction struct {
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

func (s *tudoSomedayAction) Format() string {
	return fmt.Sprint("ID: ", s.ID, "\n", s.Content, "\n")
}

type tudoWaitingAction struct {
	ID         uint32
	Content    string
	Done       bool
	CreatedAt  string
	FinishedAt string
}

func waitingActionExists(db *sql.DB, content string) (bool, uint32) {
	row := db.QueryRow("SELECT id FROM waiting WHERE content = ? AND done = 0", content)
	var id uint32
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0
	} else if err != nil {
		nonFatalError(err)
	}

	return true, id
}

func (w *tudoWaitingAction) Format() string {
	return fmt.Sprint("ID: ", w.ID, "\n", w.Content, "\n")
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
		noTasks := true
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

		if len(nextActions) > 0 {
			noTasks = false
			fmt.Println("Next Actions")
		}
		for _, a := range nextActions {
			fmt.Print(a.Format())
		}

		rows, err = db.Query("SELECT id, content, done, created_at FROM projects WHERE done = 0")
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

		now := time.Now()
		todayDate := now.Format("2006-01-02")
		for _, project := range projects {
			rows, err := db.Query("SELECT id, content, project_id, due, done, created_at FROM tasks WHERE project_id = ? AND done = 0 AND due = ?", project.ID, todayDate)
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
			if len(tasks) > 0 {
				noTasks = false
				fmt.Print("\n" + project.Content + "\n")
			}
			for _, t := range tasks {
				fmt.Print("- " + t.Format())
			}
		}
		if noTasks {
			fmt.Println("No tasks for now")
		}
		return
	}
	switch cmdArgs[1] {
	case "-h":
		fallthrough
	case "--help":
		fmt.Println("For help, use `" + cmdArgs[0] + " help`")
	case "help":
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
			reader := bufio.NewReader(os.Stdin)
			for capturing {
				select {
				case <-done:
					capturing = false
				default:
					line, _ := reader.ReadString('\n')
					captureTxt = captureTxt + strings.TrimSpace(line) + "\n"
				}
			}
			if _, err := db.Exec("INSERT INTO capture (id, content, done, created_at) VALUES (NULL, ?, 0, datetime());", captureTxt); err != nil {
				nonFatalError(err)
			}
			fmt.Println("Created new capture")
		} else if len(cmdArgs) > 2 {
			switch cmdArgs[2] {
			case "next":
				fmt.Print("Please enter new next action: ")
				reader := bufio.NewReader(os.Stdin)
				task, _ := reader.ReadString('\n')
				if _, err := db.Exec("INSERT INTO next_actions (id, content, done, created_at, finished_at) VALUES (NULL, ?, 0, datetime(), NULL)", strings.TrimSpace(task)); err != nil {
					nonFatalError(err)
				}
				fmt.Println("Created new next action")
			case "project":
				fmt.Print("Please enter new project name: ")
				var projectName string
				fmt.Scan(&projectName)
				if exists, _ := projectExists(db, projectName); exists {
					fmt.Println("Project `" + projectName + "` already exists")
					return
				}

				if _, err := db.Exec("INSERT INTO projects (id, content, done, created_at) VALUES (NULL, ?, 0, datetime())", projectName); err != nil {
					nonFatalError(err)
				}

				fmt.Println("Project `" + projectName + "` has been created")
			case "wait":
				fmt.Print("Please enter new task to wait for: ")
				reader := bufio.NewReader(os.Stdin)
				waitAction, _ := reader.ReadString('\n')
				waitAction = strings.TrimSpace(waitAction)
				if exists, _ := waitingActionExists(db, waitAction); exists {
					fmt.Println("Waiting action `" + waitAction + "` already exists")
					return
				}
				if _, err := db.Exec("INSERT INTO waiting (id, content, done, created_at, finished_at) VALUES (NULL, ?, 0, datetime(), NULL)", waitAction); err != nil {
					nonFatalError(err)
				}
				fmt.Println("Created new wait action")
			case "someday":
				fmt.Print("Please enter new project name: ")
				reader := bufio.NewReader(os.Stdin)
				futureTask, _ := reader.ReadString('\n')
				futureTask = strings.TrimSpace(futureTask)
				if exists, _ := somedayTaskExists(db, futureTask); exists {
					fmt.Println("Someday action `" + futureTask + "` already exists")
					return
				}

				if _, err := db.Exec("INSERT INTO someday (id, content, done, created_at) VALUES (NULL, ?, 0, date())", futureTask); err != nil {
					nonFatalError(err)
				}

				fmt.Println("Someday action `" + futureTask + "` has been created")
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
				reader := bufio.NewReader(os.Stdin)
				task, _ := reader.ReadString('\n')
				task = strings.TrimSpace(task)
				fmt.Print("Due date (YYYY-MM-DD): ")
				dueStr, _ := reader.ReadString('\n')
				due, err := time.Parse("2006-01-02", strings.TrimSpace(dueStr))
				if err != nil {
					nonFatalError(err)
				}
				yyyy, mm, dd := time.Now().Date()
				if due.Before(time.Date(yyyy, mm, dd, 0, 0, 0, 0, time.Now().Location())) {
					nonFatalError(errors.New("due date has passed already"))
				}
				if _, err := db.Exec("INSERT INTO tasks (id, content, project_id, due, done, created_at, finished_at) VALUES (NULL, ?, ?, ?, 0, datetime(), NULL)", task, projectID, due.Format("2006-01-02")); err != nil {
					nonFatalError(err)
				}
				fmt.Println("New task created for `" + projectName + "`")
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
			r, err := db.Exec("UPDATE next_actions SET done = 1, finished_at = datetime() WHERE id = ?", actionID)
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

			var futureTask tudoSomedayAction
			row := db.QueryRow("SELECT id, content, done, created_at FROM someday WHERE id = ? AND done = 0", id)
			err = row.Scan(&futureTask.ID, &futureTask.Content, &futureTask.Done, &futureTask.CreatedAt)
			if errors.Is(err, sql.ErrNoRows) {
				nonFatalError(errors.New("Someday action `" + cmdArgs[3] + "` does not exist"))
			}

			if _, err := db.Exec("UPDATE someday SET done = 1 WHERE id = ?", id); err != nil {
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

					if _, err := db.Exec("INSERT INTO projects (id, content, done, created_at, finished_at) VALUES (NULL, ?, 0, datetime(), NULL)", projectName); err != nil {
						nonFatalError(err)
					}

					fmt.Println("Project `" + projectName + "` has been created")
				case "y":
					if exists, _ := projectExists(db, futureTask.Content); exists {
						fmt.Println("Project `" + futureTask.Content + "` already exists")
						return
					}

					if _, err := db.Exec("INSERT INTO projects (id, content, done, created_at, finished_at) VALUES (NULL, ?, 0, datetime(), NULL)", futureTask.Content); err != nil {
						nonFatalError(err)
					}
				default:
					nonFatalError(invalidCommand)
				}
			default:
				nonFatalError(invalidCommand)
			}
		case "waiting":
			if len(cmdArgs) < 3 {
				nonFatalError(invalidCommandFormat)
			}
			id, err := strconv.Atoi(cmdArgs[3])
			if err != nil {
				nonFatalError(invalidCommandFormat)
			}
			var waitingTask tudoWaitingAction
			row := db.QueryRow("SELECT id, content, done, created_at FROM waiting WHERE id = ? AND done = 0", id)
			err = row.Scan(&waitingTask.ID, &waitingTask.Content, &waitingTask.Done, &waitingTask.CreatedAt)
			if errors.Is(err, sql.ErrNoRows) {
				nonFatalError(errors.New("Waiting action `" + cmdArgs[3] + "` does not exist"))
			}

			if _, err := db.Exec("UPDATE waiting SET done = 1, finished_at = datetime() WHERE id = ?", id); err != nil {
				nonFatalError(err)
			}

			fmt.Println("Marked waiting task `" + cmdArgs[3] + "` as done")
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

				if _, err := db.Exec("UPDATE projects SET done = 1, finished_at = datetime() WHERE content = ? AND done = 0", projectName); err != nil {
					nonFatalError(err)
				}
				fmt.Println("Finished project `" + projectName + "`\n")
			} else {
				exists, projectID := projectExists(db, projectName)
				if !exists {
					fmt.Println("No active project `" + projectName + "` exists")
					return
				}

				res, err := db.Exec("UPDATE tasks SET done = 1, finished_at = datetime() WHERE id = ? AND project_id = ?", taskID, projectID)
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
		rows, err := db.Query("SELECT id, content, done, created_at FROM waiting WHERE done = 0")
		if err != nil {
			nonFatalError(err)
		}
		var waitList []tudoWaitingAction
		for rows.Next() {
			var w tudoWaitingAction
			if err := rows.Scan(&w.ID, &w.Content, &w.Done, &w.CreatedAt); err != nil {
				nonFatalError(err)
			}
			waitList = append(waitList, w)
		}
		if len(waitList) == 0 {
			fmt.Println("No tasks to wait for now")
		}
		for _, w := range waitList {
			fmt.Println("- " + w.Format())
		}
	case "someday":
		rows, err := db.Query("SELECT id, content, done, created_at FROM someday WHERE done = 0")
		if err != nil {
			nonFatalError(err)
		}
		var futureTasks []tudoSomedayAction
		for rows.Next() {
			var s tudoSomedayAction
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
		rows, err := db.Query("SELECT tasks.id, tasks.content, tasks.project_id, tasks.due, tasks.done, tasks.created_at, projects.id, projects.content due FROM tasks JOIN projects ON projects.id = tasks.project_id WHERE projects.done = 0 AND tasks.done = 0 AND tasks.content LIKE '%read %' GROUP BY tasks.project_id")
		if err != nil {
			nonFatalError(err)
		}
		tasks := make(map[tudoProject][]tudoProjectAction)
		for rows.Next() {
			var project tudoProject
			var task tudoProjectAction
			if err := rows.Scan(&task.ID, &task.Content, &task.Project, &task.Due, &task.Done, &task.CreatedAt, &project.ID, &project.Content); err != nil {
				nonFatalError(err)
			}
			tasks[project] = append(tasks[project], task)
		}
		if len(tasks) > 0 {
			for project, projectTasks := range tasks {
				fmt.Println(project.Content)
				for _, t := range projectTasks {
					fmt.Println("- " + t.Format())
				}
			}
		} else {
			rows, err = db.Query("SELECT id, content FROM someday WHERE done = 0 AND content LIKE '%read %'")
			if err != nil {
				nonFatalError(err)
			}
			var futureTasks []tudoSomedayAction
			for rows.Next() {
				var t tudoSomedayAction
				if err := rows.Scan(&t.ID, &t.Content); err != nil {
					nonFatalError(err)
				}
				futureTasks = append(futureTasks, t)
			}
			if len(futureTasks) == 0 {
				fmt.Println("Nothing to read for now")
			} else {
				for _, t := range futureTasks {
					fmt.Println("- " + t.Format())
				}
			}
		}
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
	case "all":
		Todo("all")
	case "edit":
		Todo("edit")
	case "clean":
		row := db.QueryRow("SELECT COUNT(*) FROM next_actions WHERE done = 0")
		if row.Err() != nil {
			nonFatalError(row.Err())
		}
		var cnt int
		row.Scan(&cnt)
		if cnt > 0 {
			fmt.Println(fmt.Sprint("You will be cleaning out `", cnt, "` next actions. Continue? (y/n)"))
			var ans string
			fmt.Scanln(&ans)
			switch ans {
			case "n":
				break
			case "y":
				if _, err := db.Exec("UPDATE next_actions SET done = 1 WHERE done = 0"); err != nil {
					nonFatalError(err)
				}
				fmt.Println(fmt.Sprint("Cleaned `", cnt, "` next actions"))
			default:
				nonFatalError(invalidCommand)
			}
		}
		row = db.QueryRow("SELECT COUNT(*) FROM tasks JOIN projects ON projects.id = tasks.project_id WHERE projects.done = 0 AND tasks.done = 0")
		row.Scan(&cnt)
		if cnt > 0 {
			fmt.Println(fmt.Sprint("You will be cleaning out `", cnt, "` tasks. Continue? (y/n)"))
			var ans string
			fmt.Scanln(&ans)
			switch ans {
			case "n":
				break
			case "y":
				if _, err := db.Exec("UPDATE tasks SET done = 1 WHERE done = 0 AND project_id IN (SELECT id FROM projects WHERE done = 0)"); err != nil {
					nonFatalError(err)
				}
				fmt.Println(fmt.Sprint("Cleaned `", cnt, "` tasks"))
			default:
				nonFatalError(invalidCommand)
			}
		}
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

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

func fatalError(err error, args ...string) {
	fmt.Println("Error occurred!!")
	e := err.Error()
	for _, arg := range args {
		e += arg + " "
	}
	fmt.Println(e)
	os.Exit(1)
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

type tudoTask struct {
	ID         uint32
	Content    string
	ProjectID  *uint32
	Context    *string
	Due        *string
	Done       bool
	CreatedAt  string
	FinishedAt *string
}

type tudoContext struct {
	ID      uint32
	Content string
}

type tudoProject struct {
	ID         uint32
	Content    string
	Done       bool
	CreatedAt  string
	FinishedAt *string
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
	return fmt.Sprint("ID: ", p.ID, "\n", p.Content, "\n")
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
	FinishedAt *string
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

type tudoLogAction struct {
	ID        uint32
	TableName string
	RowID     uint32
	CreatedAt string
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
		rows, err := db.Query("SELECT id, content, context FROM tasks WHERE done == 0 AND project_id IS NULL GROUP BY context")
		if err != nil {
			fatalError(err)
		}
		defer rows.Close()

		var nextActions []tudoTask
		for rows.Next() {
			var action tudoTask
			if err := rows.Scan(&action.ID, &action.Content, &action.Context); err != nil {
				fatalError(err)
			}
			nextActions = append(nextActions, action)
		}

		if len(nextActions) > 0 {
			noTasks = false
			fmt.Println("NEXT ACTIONS")
		}
		for _, a := range nextActions {
			fmt.Print(fmt.Sprint("- ID: ", a.ID, "\n", a.Content, "\n"))
			if a.Context != nil {
				fmt.Print(fmt.Sprint("Context: ", *a.Context, "\n"))
			}
		}

		rows, err = db.Query("SELECT id, content, done, created_at FROM projects WHERE done = 0")
		if err != nil {
			fatalError(err)
		}

		fmt.Println("\nPROJECTS")

		var projects []tudoProject
		for rows.Next() {
			var p tudoProject
			if err := rows.Scan(&p.ID, &p.Content, &p.Done, &p.CreatedAt); err != nil {
				fatalError(err)
			}
			projects = append(projects, p)
		}

		yyyy, mm, dd := time.Now().Date()
		todayDate := time.Date(yyyy, mm, dd, 0, 0, 0, 0, time.Now().Location())

		for _, project := range projects {
			rows, err := db.Query("SELECT id, content, project_id, context, due, done FROM tasks WHERE project_id = ? AND done = 0", project.ID)
			if err != nil {
				fatalError(err)
			}
			defer rows.Close()

			var calendarTasks []tudoTask
			var nonCalendarTasks []tudoTask
			for rows.Next() {
				var t tudoTask
				if err := rows.Scan(&t.ID, &t.Content, &t.ProjectID, &t.Context, &t.Due, &t.Done); err != nil {
					fatalError(err)
				}
				if t.Due != nil {
					due, err := time.ParseInLocation("2006-01-02", *t.Due, time.Now().Location())
					if err != nil {
						fatalError(err)
					}
					if todayDate.Equal(due) {
						calendarTasks = append(calendarTasks, t)
					}
				} else {
					nonCalendarTasks = append(nonCalendarTasks, t)
				}
			}
			if len(calendarTasks) > 0 || len(nonCalendarTasks) > 0 {
				noTasks = false
				fmt.Println(project.Content)
			}
			for _, t := range calendarTasks {
				fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
				if t.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
				}
				fmt.Print(fmt.Sprint("Due: ", *t.Due, "\n"))
			}
			for _, t := range nonCalendarTasks {
				fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
				if t.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
				}
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
			nonFatalError(invalidCommandFormat)
		}

		rows, err := db.Query("SELECT id, content FROM contexts")
		if err != nil {
			fatalError(err)
		}
		contextMap := make(map[uint32]string)
		for rows.Next() {
			var c tudoContext
			if err := rows.Scan(&c.ID, &c.Content); err != nil {
				fatalError(err)
			}
			contextMap[c.ID] = c.Content
		}

		reader := bufio.NewReader(os.Stdin)

		switch cmdArgs[2] {
		case "in":
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
			for capturing {
				select {
				case <-done:
					capturing = false
				default:
					line, _ := reader.ReadString('\n')
					captureTxt = captureTxt + line + "\n"
				}
			}
			if _, err := db.Exec("INSERT INTO capture (id, content, done, created_at) VALUES (NULL, ?, 0, date());", captureTxt); err != nil {
				fatalError(err)
			}
			fmt.Println("Created new capture")

		case "next":
			fmt.Print("Please enter new next action: ")
			task, _ := reader.ReadString('\n')
			task = strings.TrimSpace(task)

			fmt.Println("Select context (Press ENTER if no context): ")
			for k, v := range contextMap {
				fmt.Println(fmt.Sprint(k, ". ", v, "\n"))
			}

			number, _ := reader.ReadString('\n')
			number = strings.TrimSpace(number)

			if number == "" {
				if _, err := db.Exec("INSERT INTO tasks (id, content, project_id, context, due, done, created_at, finished_at) VALUES (NULL, ?, NULL, NULL, NULL, 0, date(), NULL)", task); err != nil {
					fatalError(err)
				}
			} else {
				contextID, err := strconv.Atoi(number)
				if err != nil {
					nonFatalError(invalidCommandFormat)
				}
				if _, err := db.Exec("INSERT INTO tasks (id, content, project_id, context, due, done, created_at, finished_at) VALUES (NULL, ?, NULL, ?, NULL, 0, date(), NULL)", task, contextMap[uint32(contextID)]); err != nil {
					fatalError(err)
				}
			}
			fmt.Println("Created new next action")

		case "project":
			fmt.Print("Please enter new project name: ")
			projectName, _ := reader.ReadString('\n')
			projectName = strings.TrimSpace(projectName)

			if exists, _ := projectExists(db, projectName); exists {
				fmt.Println("Project `" + projectName + "` already exists")
				return
			}

			if _, err := db.Exec("INSERT INTO projects (id, content, done, created_at) VALUES (NULL, ?, 0, date())", projectName); err != nil {
				fatalError(err)
			}

			fmt.Println("Project `" + projectName + "` has been created")

		case "context":
			fmt.Println("Currently available contexts: ")
			for k, v := range contextMap {
				fmt.Println(fmt.Sprint(k, ". ", v, "\n"))
			}
			fmt.Println("New context: ")
			context, _ := reader.ReadString('\n')
			context = strings.TrimSpace(context)

			for _, v := range contextMap {
				if v == context {
					nonFatalError(errors.New("Context `" + context + "` already exists"))
				}
			}

			if _, err := db.Exec("INSERT INTO contexts (id, content) VALUES (NULL, ?)", context); err != nil {
				fatalError(err)
			}
			fmt.Println("Created new context `" + context + "`")

		case "wait":
			fmt.Print("Please enter new task to wait for: ")
			waitAction, _ := reader.ReadString('\n')
			waitAction = strings.TrimSpace(waitAction)
			if exists, _ := waitingActionExists(db, waitAction); exists {
				fmt.Println("Waiting action `" + waitAction + "` already exists")
				return
			}
			if _, err := db.Exec("INSERT INTO waiting (id, content, done, created_at, finished_at) VALUES (NULL, ?, 0, date(), NULL)", waitAction); err != nil {
				fatalError(err)
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
				fatalError(err)
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
				nonFatalError(errors.New("No active project `" + projectName + "` exists"))
			}

			fmt.Print("Please enter new task for the project `" + projectName + "`: ")
			task, _ := reader.ReadString('\n')
			task = strings.TrimSpace(task)

			fmt.Print("Due date (YYYY-MM-DD) (Press ENTER if no due date): ")
			dueStr, _ := reader.ReadString('\n')
			dueStr = strings.TrimSpace(dueStr)

			if dueStr == "" {
				fmt.Println("Please enter context (Press ENTER for no context): ")
				for k, v := range contextMap {
					fmt.Println(fmt.Sprint(k, ". ", v))
				}
				number, _ := reader.ReadString('\n')
				number = strings.TrimSpace(number)

				if number == "" {
					if _, err := db.Exec("INSERT INTO tasks (id, content, project_id, context, due, done, created_at, finished_at) VALUES (NULL, ?, ?, NULL, NULL, 0, date(), NULL)", task, projectID); err != nil {
						fatalError(err)
					}
				} else {
					contextID, err := strconv.Atoi(number)
					if err != nil {
						nonFatalError(invalidCommandFormat)
					}

					if _, err := db.Exec("INSERT INTO tasks (id, content, project_id, context, due, done, created_at, finished_at) VALUES (NULL, ?, ?, ?, NULL, 0, date(), NULL)", task, projectID, contextMap[uint32(contextID)]); err != nil {
						fatalError(err)
					}
				}
			} else {
				due, err := time.Parse("2006-01-02", dueStr)
				if err != nil {
					nonFatalError(err)
				}
				yyyy, mm, dd := time.Now().Date()
				if due.Before(time.Date(yyyy, mm, dd, 0, 0, 0, 0, time.Now().Location())) {
					nonFatalError(errors.New("due date has passed already"))
				}

				fmt.Println("Please enter context (Press ENTER for no context): ")
				for k, v := range contextMap {
					fmt.Println(fmt.Sprint(k, ". ", v))
				}
				number, _ := reader.ReadString('\n')
				number = strings.TrimSpace(number)
				if number == "" {
					if _, err := db.Exec("INSERT INTO tasks (id, content, project_id, context, due, done, created_at, finished_at) VALUES (NULL, ?, ?, NULL, ?, 0, date(), NULL)", task, projectID, due.Format("2006-01-02")); err != nil {
						fatalError(err)
					}
				} else {
					contextID, err := strconv.Atoi(number)
					if err != nil {
						nonFatalError(invalidCommandFormat)
					}

					if _, err := db.Exec("INSERT INTO tasks (id, content, project_id, context, due, done, created_at, finished_at) VALUES (NULL, ?, ?, ?, ?, 0, date(), NULL)", task, projectID, contextMap[uint32(contextID)], due.Format("2006-01-02")); err != nil {
						fatalError(err)
					}
				}
			}
			fmt.Println("New task created for `" + projectName + "`")
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
				fatalError(err)
			}

			rows, err := r.RowsAffected()
			if err != nil {
				fatalError(err)
			}

			if rows == 0 {
				fmt.Println("Capture item `" + cmdArgs[3] + "` does not exist")
			} else {
				fmt.Println("Marked capture item `" + cmdArgs[3] + "` as finished\n")
				if _, err := db.Exec("INSERT INTO action_log (id, table_name, row_id, created_at) VALUES (NULL, ?, ?, datetime())", "capture", captureID); err != nil {
					fatalError(err)
				}
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
				fatalError(err)
			}

			fmt.Println("Marked someday task `" + cmdArgs[3] + "` as done")
			if _, err := db.Exec("INSERT INTO action_log (id, table_name, row_id, created_at) VALUES (NULL, ?, ?, datetime())", "someday", id); err != nil {
				fatalError(err)
			}

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

					if _, err := db.Exec("INSERT INTO projects (id, content, done, created_at, finished_at) VALUES (NULL, ?, 0, date(), NULL)", projectName); err != nil {
						fatalError(err)
					}

					fmt.Println("Project `" + projectName + "` has been created")
				case "y":
					if exists, _ := projectExists(db, futureTask.Content); exists {
						fmt.Println("Project `" + futureTask.Content + "` already exists")
						return
					}

					if _, err := db.Exec("INSERT INTO projects (id, content, done, created_at, finished_at) VALUES (NULL, ?, 0, date(), NULL)", futureTask.Content); err != nil {
						fatalError(err)
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
				fmt.Println("Waiting action `" + cmdArgs[3] + "` does not exist")
			}

			if _, err := db.Exec("UPDATE waiting SET done = 1, finished_at = date() WHERE id = ?", id); err != nil {
				fatalError(err)
			}

			fmt.Println("Marked waiting task `" + cmdArgs[3] + "` as done")

			if _, err := db.Exec("INSERT INTO action_log (id, table_name, row_id, created_at) VALUES (NULL, ?, ?, datetime())", "waiting", id); err != nil {
				fatalError(err)
			}

		case "task":
			taskID, err := strconv.Atoi(cmdArgs[3])
			if err != nil {
				nonFatalError(invalidCommandFormat)
			}

			row := db.QueryRow("SELECT content FROM tasks WHERE id = ?", taskID)
			err = row.Err()
			if errors.Is(err, sql.ErrNoRows) {
				fmt.Print(fmt.Sprint("Task ", cmdArgs[3], " does not exist\n"))
			} else if err != nil {
				fatalError(err)
			}

			var finishedTask string
			if err := row.Scan(&finishedTask); err != nil {
				fatalError(err)
			}

			_, err = db.Exec("UPDATE tasks SET done = 1, finished_at = date() WHERE id = ?", taskID)
			if err != nil {
				fatalError(err)
			}

			fmt.Println("Finished task `" + cmdArgs[3] + "`\n`" + finishedTask + "`")

			if _, err := db.Exec("INSERT INTO action_log (id, table_name, row_id, created_at) VALUES (NULL, ?, ?, datetime())", "tasks", taskID); err != nil {
				fatalError(err)
			}

		default:
			projectName := ""
			for i := 2; i < len(cmdArgs); i++ {
				if i > 2 {
					projectName += " "
				}
				projectName += cmdArgs[i]
			}
			exists, projectID := projectExists(db, projectName)
			if !exists {
				fmt.Println("No active project `" + projectName + "` exists")
				return
			}

			if _, err := db.Exec("UPDATE projects SET done = 1, finished_at = date() WHERE id = ? AND done = 0", projectID); err != nil {
				fatalError(err)
			}
			fmt.Println("Finished project `" + projectName + "`\n")

			if _, err := db.Exec("INSERT INTO action_log (id, table_name, id, created_at) VALUES (NULL, ?, ?, datetime())", "tasks", projectID); err != nil {
				fatalError(err)
			}
		}

	case "in":
		rows, err := db.Query("SELECT id, content FROM capture WHERE done == 0")
		if err != nil {
			fatalError(err)
		}
		defer rows.Close()

		var captureList []tudoCapture
		for rows.Next() {
			var c tudoCapture
			if err := rows.Scan(&c.ID, &c.Content); err != nil {
				fatalError(err)
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
			fatalError(err)
		}
		defer rows.Close()

		var waitList []tudoWaitingAction
		for rows.Next() {
			var w tudoWaitingAction
			if err := rows.Scan(&w.ID, &w.Content, &w.Done, &w.CreatedAt); err != nil {
				fatalError(err)
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
			fatalError(err)
		}
		defer rows.Close()

		var futureTasks []tudoSomedayAction
		for rows.Next() {
			var s tudoSomedayAction
			if err := rows.Scan(&s.ID, &s.Content, &s.Done, &s.CreatedAt); err != nil {
				fatalError(err)
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
		rows, err := db.Query("SELECT tasks.id, tasks.content, tasks.project_id, tasks.context, tasks.due, tasks.done, tasks.created_at, projects.id, projects.content due FROM tasks LEFT JOIN projects ON projects.id = tasks.project_id WHERE projects.done = 0 AND tasks.done = 0 AND tasks.content LIKE '%read%' GROUP BY tasks.due")
		if err != nil {
			fatalError(err)
		}
		defer rows.Close()

		projectTasks := make(map[tudoProject][]tudoTask)
		var nextActions []tudoTask
		for rows.Next() {
			var project tudoProject
			var task tudoTask
			if err := rows.Scan(&task.ID, &task.Content, &task.ProjectID, &task.Context, &task.Due, &task.Done, &task.CreatedAt, &project.ID, &project.Content); err != nil {
				fatalError(err)
			}
			if task.ProjectID != nil {
				projectTasks[project] = append(projectTasks[project], task)
			} else {
				nextActions = append(nextActions, task)
			}
		}

		if len(nextActions) > 0 {
			for _, t := range nextActions {
				fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
				if t.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
				}
			}
		}

		if len(projectTasks) > 0 {
			for project, projectTasks := range projectTasks {
				fmt.Println(project.Content)
				for _, t := range projectTasks {
					fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
					if t.Context != nil {
						fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
					}
					if t.Due != nil {
						fmt.Print(fmt.Sprint("Due: ", *t.Due, "\n"))
					}
				}
			}
		} else if len(nextActions) == 0 && len(projectTasks) == 0 {
			rows, err = db.Query("SELECT id, content FROM someday WHERE done = 0 AND content LIKE '%read %'")
			if err != nil {
				fatalError(err)
			}
			var futureTasks []tudoSomedayAction
			for rows.Next() {
				var t tudoSomedayAction
				if err := rows.Scan(&t.ID, &t.Content); err != nil {
					fatalError(err)
				}
				futureTasks = append(futureTasks, t)
			}
			if len(futureTasks) == 0 {
				fmt.Println("Nothing to read for now")
			} else {
				fmt.Println("SOMEDAY")
				for _, t := range futureTasks {
					fmt.Println("- " + t.Format())
				}
			}
		}

	case "next":
		rows, err := db.Query("SELECT id, content, context FROM tasks WHERE done == 0 AND project_id IS NULL")
		if err != nil {
			fatalError(err)
		}
		defer rows.Close()

		var nextActions []tudoTask
		for rows.Next() {
			var action tudoTask
			if err := rows.Scan(&action.ID, &action.Content, &action.Context); err != nil {
				fatalError(err)
			}
			nextActions = append(nextActions, action)
		}

		if len(nextActions) == 0 {
			fmt.Println("No next actions for now")
		}

		for _, a := range nextActions {
			fmt.Print(fmt.Sprint("- ID: ", a.ID, "\n", a.Content, "\n"))
			if a.Context != nil {
				fmt.Print(fmt.Sprint("Context: ", *a.Context, "\n"))
			}
		}

	case "all":
		if len(cmdArgs) == 2 {
			noTasks := true
			rows, err := db.Query("SELECT id, content, context FROM tasks WHERE done == 0 AND project_id IS NULL GROUP BY context")
			if err != nil {
				fatalError(err)
			}
			defer rows.Close()

			var nextActions []tudoTask
			for rows.Next() {
				var action tudoTask
				if err := rows.Scan(&action.ID, &action.Content, &action.Context); err != nil {
					fatalError(err)
				}
				nextActions = append(nextActions, action)
			}

			if len(nextActions) > 0 {
				noTasks = false
				fmt.Println("NEXT ACTIONS")
			}
			for _, a := range nextActions {
				fmt.Print(fmt.Sprint("- ID: ", a.ID, "\n", a.Content, "\n"))
				if a.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *a.Context, "\n"))
				}
			}

			rows, err = db.Query("SELECT id, content, done, created_at FROM projects WHERE done = 0")
			if err != nil {
				fatalError(err)
			}

			fmt.Println("\nPROJECTS")

			var projects []tudoProject
			for rows.Next() {
				var p tudoProject
				if err := rows.Scan(&p.ID, &p.Content, &p.Done, &p.CreatedAt); err != nil {
					fatalError(err)
				}
				projects = append(projects, p)
			}

			for _, project := range projects {
				rows, err := db.Query("SELECT id, content, project_id, context, due, done FROM tasks WHERE project_id = ? AND done = 0", project.ID)
				if err != nil {
					fatalError(err)
				}
				defer rows.Close()

				var calendarTasks []tudoTask
				var nonCalendarTasks []tudoTask
				for rows.Next() {
					var t tudoTask
					if err := rows.Scan(&t.ID, &t.Content, &t.ProjectID, &t.Context, &t.Due, &t.Done); err != nil {
						fatalError(err)
					}
					if t.Due != nil {
						calendarTasks = append(calendarTasks, t)
					} else {
						nonCalendarTasks = append(nonCalendarTasks, t)
					}
				}
				if len(calendarTasks) > 0 || len(nonCalendarTasks) > 0 {
					noTasks = false
					fmt.Println(project.Content)
				}
				for _, t := range calendarTasks {
					fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
					if t.Context != nil {
						fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
					}
					fmt.Print(fmt.Sprint("Due: ", *t.Due, "\n"))
				}
				for _, t := range nonCalendarTasks {
					fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
					if t.Context != nil {
						fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
					}
				}
			}

			fmt.Println("\nWAITING")

			rows, err = db.Query("SELECT id, content, done FROM waiting WHERE done = 0")
			if err != nil {
				fatalError(err)
			}

			var waitingTasks []tudoWaitingAction
			for rows.Next() {
				var w tudoWaitingAction
				if err := rows.Scan(&w.ID, &w.Content, &w.Done); err != nil {
					fatalError(err)
				}
				waitingTasks = append(waitingTasks, w)
			}

			if len(waitingTasks) > 0 {
				noTasks = false
			}
			for _, w := range waitingTasks {
				fmt.Print(fmt.Sprint("- ID: ", w.ID, "\n", w.Content, "\n"))
			}

			if noTasks {
				fmt.Println("No tasks for now")
			}
		} else {
			projectName := ""
			for i := 2; i < len(cmdArgs); i++ {
				if i > 2 {
					projectName += " "
				}
				projectName += cmdArgs[i]
			}
			exists, projectID := projectExists(db, projectName)
			if !exists {
				fmt.Println("No active project `" + projectName + "` exists\n")
			}

			rows, err := db.Query("SELECT id, content, project_id, context, due FROM tasks WHERE project_id = ? AND done = 0", projectID)
			if err != nil {
				fatalError(err)
			}
			for rows.Next() {
				var t tudoTask
				if err := rows.Scan(&t.ID, &t.Content, &t.ProjectID, &t.Context, &t.Due); err != nil {
					fatalError(err)
				}
				fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
				if t.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
				}
				if t.Due != nil {
					fmt.Print(fmt.Sprint("Due: ", *t.Due, "\n"))
				}
			}
		}

	case "edit":
		if len(cmdArgs) < 3 {
			nonFatalError(invalidCommandFormat)
		}

		switch cmdArgs[2] {
		case "in":
			Todo("edit in")

		case "task":
			Todo("edit task")

		case "waiting":
			Todo("edit waiting")

		case "someday":
			Todo("edit someday")

		case "project":
			Todo("edit project")
		}

	case "clean":
		row := db.QueryRow("SELECT COUNT(*) FROM capture WHERE done = 0")
		if row.Err() != nil {
			fatalError(row.Err())
		}
		var cnt int
		row.Scan(&cnt)
		if cnt > 0 {
			fmt.Println(fmt.Sprint("You will be cleaning out `", cnt, "` items from in list. Continue? (y/n)"))
			var ans string
			fmt.Scanln(&ans)
			switch ans {
			case "n":
				break
			case "y":
				if _, err := db.Exec("UPDATE capture SET done = 1 WHERE done = 0"); err != nil {
					fatalError(err)
				}
				fmt.Println(fmt.Sprint("Cleaned `", cnt, "` items from in list"))
			default:
				nonFatalError(invalidCommand)
			}
		}

		row = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE done = 0 AND project_id IS NULL")
		if row.Err() != nil {
			fatalError(row.Err())
		}
		row.Scan(&cnt)
		if cnt > 0 {
			fmt.Println(fmt.Sprint("You will be cleaning out `", cnt, "` next actions. Continue? (y/n)"))
			var ans string
			fmt.Scanln(&ans)
			switch ans {
			case "n":
				break
			case "y":
				if _, err := db.Exec("UPDATE tasks SET done = 1, finished_at = date() WHERE done = 0 AND project_id IS NULL"); err != nil {
					fatalError(err)
				}
				fmt.Println(fmt.Sprint("Cleaned `", cnt, "` next actions"))
			default:
				nonFatalError(invalidCommand)
			}
		}

		row = db.QueryRow("SELECT COUNT(*) FROM tasks JOIN projects ON projects.id = tasks.project_id WHERE projects.done = 0 AND tasks.done = 0")
		row.Scan(&cnt)
		if cnt > 0 {
			fmt.Println(fmt.Sprint("You will be cleaning out `", cnt, "` project tasks. Continue? (y/n)"))
			var ans string
			fmt.Scanln(&ans)
			switch ans {
			case "n":
				break
			case "y":
				if _, err := db.Exec("UPDATE tasks SET done = 1, finished_at = date() WHERE done = 0 AND project_id IN (SELECT id FROM projects WHERE done = 0)"); err != nil {
					fatalError(err)
				}
				fmt.Println(fmt.Sprint("Cleaned `", cnt, "` project tasks"))
			default:
				nonFatalError(invalidCommand)
			}
		}

	case "undo":
		row := db.QueryRow("SELECT id, table_name, row_id, created_at FROM action_log ORDER BY id DESC LIMIT 1")
		err := row.Err()
		if err != nil {
			fatalError(err)
		}

		var a tudoLogAction
		row.Scan(&a.ID, &a.TableName, &a.RowID, &a.CreatedAt)
		if a.ID <= 0 || a.TableName == "" || a.RowID <= 0 {
			nonFatalError(errors.New("No actions to undo"))
		}

		fmt.Print(fmt.Sprint("You will be marking `", a.RowID, "` in `", a.TableName, "` as undone. Continue? (y/n)\n"))
		var ans string
		fmt.Scanln(&ans)
		switch ans {
		case "n":
		case "y":
			if _, err := db.Exec("UPDATE "+a.TableName+" SET done = 0 WHERE id = ?", a.RowID); err != nil {
				fatalError(err)
			}
			if _, err := db.Exec("DELETE FROM action_log WHERE id = ?", a.ID); err != nil {
				fatalError(err)
			}
			fmt.Print(fmt.Sprint("Marked `", a.RowID, "` in `", a.TableName, "` as undone\n"))
		default:
			nonFatalError(invalidCommand)
		}

	case "review":
		thresh := time.Now().AddDate(0, 0, -7)

		fmt.Println("FINISHED PROJECTS")

		rows, err := db.Query("SELECT id, content, finished_at FROM projects WHERE finished_at IS NOT NULL AND done = 1")
		if err != nil {
			fatalError(err)
		}
		defer rows.Close()

		for rows.Next() {
			var p tudoProject
			if err = rows.Scan(&p.ID, &p.Content, &p.FinishedAt); err != nil {
				fatalError(err)
			}
			finishTime, err := time.Parse("2006-01-02", *p.FinishedAt)
			if err != nil {
				nonFatalError(err)
			}
			if thresh.Before(finishTime) {
				fmt.Println("- " + p.Content)
			}
		}

		fmt.Println("\nONGOING PROJECTS")

		rows, err = db.Query("SELECT id, content FROM projects WHERE done = 0")
		if err != nil {
			fatalError(err)
		}
		for rows.Next() {
			var p tudoProject
			if err = rows.Scan(&p.ID, &p.Content); err != nil {
				fatalError(err)
			}
			fmt.Print("- " + p.Format())
		}

		pendingTasks := make(map[tudoProject][]tudoTask)
		finishedTasks := make(map[tudoProject][]tudoTask)
		rows, err = db.Query("SELECT tasks.id, tasks.content, tasks.context, tasks.due, tasks.done, tasks.finished_at, projects.id, projects.content FROM tasks JOIN projects ON projects.id = tasks.project_id WHERE projects.done = 0 GROUP BY projects.id")
		if err != nil {
			fatalError(err)
		}

		for rows.Next() {
			var p tudoProject
			var t tudoTask
			if err = rows.Scan(&t.ID, &t.Content, &t.Due, &t.Done, &t.FinishedAt, &p.ID, &p.Content); err != nil {
				fatalError(err)
			}
			dueTime, err := time.Parse("2006-01-02", *t.Due)
			if err != nil {
				nonFatalError(err)
			}
			if thresh.Before(dueTime) {
				if t.Done {
					finishedTasks[p] = append(finishedTasks[p], t)
				} else {
					pendingTasks[p] = append(pendingTasks[p], t)
				}
			}
		}

		var pendingNextAction []tudoTask
		var finishedNextAction []tudoTask
		rows, err = db.Query("SELECT id, content, context, done, finished_at FROM tasks WHERE project_id IS NULL")
		if err != nil {
			fatalError(err)
		}
		for rows.Next() {
			var n tudoTask
			if err = rows.Scan(&n.ID, &n.Content, &n.Done, &n.FinishedAt); err != nil {
				fatalError(err)
			}

			if n.Done {
				finishedTime, err := time.Parse("2006-01-02", *n.FinishedAt)
				if err != nil {
					nonFatalError(err)
				}
				if thresh.Before(finishedTime) {
					finishedNextAction = append(finishedNextAction, n)
				}
			} else {
				pendingNextAction = append(pendingNextAction, n)
			}
		}

		var pendingWaitingAction []tudoWaitingAction
		var finishedWaitingAction []tudoWaitingAction
		rows, err = db.Query("SELECT id, content, done, finished_at FROM waiting")
		if err != nil {
			fatalError(err)
		}
		for rows.Next() {
			var n tudoWaitingAction
			if err = rows.Scan(&n.ID, &n.Content, &n.Done, &n.FinishedAt); err != nil {
				fatalError(err)
			}

			if n.Done {
				finishedTime, err := time.Parse("2006-01-02", *n.FinishedAt)
				if err != nil {
					nonFatalError(err)
				}
				if thresh.Before(finishedTime) {
					finishedWaitingAction = append(finishedWaitingAction, n)
				}
			} else {
				pendingWaitingAction = append(pendingWaitingAction, n)
			}
		}

		fmt.Println("\nPENDING TASKS")
		fmt.Println("Next Actions")
		for _, n := range pendingNextAction {
			fmt.Println("- " + n.Content)
		}

		fmt.Println("Waiting")
		for _, w := range pendingWaitingAction {
			fmt.Println("- " + w.Content)
		}

		fmt.Println("Projects")
		for project, tasks := range pendingTasks {
			fmt.Println(project.Content)
			for _, t := range tasks {
				fmt.Println("- " + t.Content)
			}
		}

		fmt.Println("\nCOMPLETED TASKS")
		fmt.Println("Next Actions")
		for _, n := range finishedNextAction {
			fmt.Println("- " + n.Content)
		}

		fmt.Println("Waiting")
		for _, w := range finishedWaitingAction {
			fmt.Println("- " + w.Content)
		}

		fmt.Println("Projects")
		for project, tasks := range finishedTasks {
			fmt.Println(project.Content)
			for _, t := range tasks {
				fmt.Println("- " + t.Content)
			}
		}

	default:
		if len(cmdArgs) == 2 {
			switch cmdArgs[1] {
			case "projects":
				rows, err := db.Query("SELECT id, content, done, created_at FROM projects WHERE done = 0")
				if err != nil {
					fatalError(err)
				}
				defer rows.Close()

				var projects []tudoProject
				for rows.Next() {
					var p tudoProject
					if err := rows.Scan(&p.ID, &p.Content, &p.Done, &p.CreatedAt); err != nil {
						fatalError(err)
					}
					projects = append(projects, p)
				}
				if len(projects) == 0 {
					fmt.Println("No active projects")
				}
				for _, p := range projects {
					fmt.Print("- " + p.Format())
				}
			case "contexts":
				rows, err := db.Query("SELECT id, content FROM contexts")
				if err != nil {
					fatalError(err)
				}
				for rows.Next() {
					var c tudoContext
					if err := rows.Scan(&c.ID, &c.Content); err != nil {
						fatalError(err)
					}
					fmt.Print(fmt.Sprint("- ID: ", c.ID, "\n", c.Content, "\n"))
				}
			default:
				nonFatalError(invalidCommand)
			}
		} else {
			projectName := ""
			for i := 1; i < len(cmdArgs); i++ {
				if i > 1 {
					projectName += " "
				}
				projectName += cmdArgs[i]
			}
			exists, projectID := projectExists(db, projectName)
			if !exists {
				fmt.Println("No active project `" + projectName + "` exists\n")
			}

			yyyy, mm, dd := time.Now().Date()
			todayDate := time.Date(yyyy, mm, dd, 0, 0, 0, 0, time.Now().Location())

			rows, err := db.Query("SELECT id, content, project_id, context, due, done, created_at FROM tasks WHERE project_id = ? AND done = 0", projectID)
			if err != nil {
				fatalError(err)
			}
			defer rows.Close()

			var calendarTasks []tudoTask
			var nonCalendarTasks []tudoTask
			for rows.Next() {
				var t tudoTask
				if err := rows.Scan(&t.ID, &t.Content, &t.ProjectID, &t.Context, &t.Due, &t.Done, &t.CreatedAt); err != nil {
					fatalError(err)
				}
				if t.Due != nil {
					due, err := time.ParseInLocation("2006-01-02", *t.Due, time.Now().Location())
					if err != nil {
						fatalError(err)
					}
					if todayDate.Equal(due) {
						calendarTasks = append(calendarTasks, t)
					}
				} else {
					nonCalendarTasks = append(nonCalendarTasks, t)
				}
			}
			if len(calendarTasks) == 0 && len(nonCalendarTasks) == 0 {
				fmt.Println("No tasks for project `" + projectName + "` today")
			}
			for _, t := range calendarTasks {
				fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
				if t.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
				}
				fmt.Print(fmt.Sprint("Due: ", *t.Due, "\n"))
			}
			for _, t := range nonCalendarTasks {
				fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
				if t.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
				}
			}
		}
	}
}

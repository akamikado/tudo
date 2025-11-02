package plaintext

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"tudo/core/capture"
	"tudo/core/contexts"
	"tudo/core/log"
	"tudo/core/projects"
	"tudo/core/someday"
	"tudo/core/tasks"
	"tudo/core/waiting"
	"tudo/database"
)

var invalidCommand error = errors.New("Invalid command: ")

var invalidCommandFormat error = errors.New("Invalid command format")

func ParseArgs(dbFile string, args []string) {
	db, err := database.Connect(dbFile)
	if err != nil {
		fmt.Println("Could not connect to database :" + err.Error())
		os.Exit(1)
	}
	defer db.Close()

	if len(args) == 0 {
		noTasks := true

		calendarTasks, err := tasks.GetTodayCalenderTasks(db)
		if err != nil {
			fatalError(err)
		}
		if len(calendarTasks) > 0 {
			noTasks = false
		}
		fmt.Println("CALENDAR")
		for _, t := range calendarTasks {
			fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
			if t.ProjectID != nil {
				project, err := projects.Get(db, *t.ProjectID)
				if err != nil {
					fatalError(err)
				}
				fmt.Print(fmt.Sprint("Project: ", project.Content, "\n"))
			}
			if t.Context != nil {
				fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
			}
		}

		nextActions, err := tasks.GetActiveNextActions(db)
		if err != nil {
			fatalError(err)
		}
		if len(nextActions) > 0 {
			noTasks = false
		}

		fmt.Println("\nNEXT ACTIONS")
		for _, a := range nextActions {
			fmt.Print(fmt.Sprint("- ID: ", a.ID, "\n", a.Content, "\n"))
			if a.Context != nil {
				fmt.Print(fmt.Sprint("Context: ", *a.Context, "\n"))
			}
		}

		projectList, err := projects.GetActive(db)
		if err != nil {
			fatalError(err)
		}

		fmt.Println("\nPROJECTS")
		for _, project := range projectList {
			tasks, err := tasks.GetActiveProjectTasks(db, project.ID)
			if err != nil {
				fatalError(err)
			}
			if len(tasks) > 0 {
				noTasks = false
				fmt.Println(project.Content)
			}
			for _, t := range tasks {
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

	switch args[0] {
	case "-h":
		fallthrough
	case "--help":
		fmt.Println("For help, use `tudo help`")
	case "help":
		fmt.Print(`tudo - personal command-line task manager

Usage:
    tudo [command] [arguments]

Commands:
    help                      Show this help message
    new <type>                Create a new item
        in                    Start a capture session
        next                  Create a next action
        project               Create a new project
        context               Create a new context
        wait                  Create a new waiting-for task
        someday               Create a new someday/maybe item
        <project name>        Add a new task under the given project

    done <type> <id|name>     Mark an item as done
        in <id>               Mark a capture item as done
        task <id>             Mark a task as done
        waiting <id>          Mark a waiting-for task as done
        someday <id>          Mark a someday item as done
        <project name>        Mark a project as completed

    in                        List active capture (in) items
    waiting                   List active waiting-for items
    someday                   List active someday/maybe items
    next                      List active next actions
    all                       Show all active and calendar tasks
    all <project>             Show all tasks under a specific project

    read                      Review tasks marked for reading (or someday if none)
    review                    Review weekly progress and missed calendar tasks
    edit <type> <id>          Edit an existing item (still in development)
    clean                     Remove completed items from all lists
    undo                      Undo the last completed action
`)

	case "new":
		if len(args) == 1 {
			nonFatalError(invalidCommandFormat)
		}

		contextMap := make(map[uint32]*string)
		contextList, err := contexts.GetAll(db)
		if err != nil {
			fatalError(err)
		}
		for _, c := range contextList {
			contextMap[c.ID] = &c.Content
		}

		reader := bufio.NewReader(os.Stdin)

		switch args[1] {
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
					captureTxt = captureTxt + line
				}
			}
			capture.New(db, captureTxt)
			fmt.Println("Created new capture")

		case "next":
			fmt.Print("Please enter new next action: ")
			task, _ := reader.ReadString('\n')
			task = strings.TrimSpace(task)

			fmt.Print("Due date (YYYY-MM-DD) (Press ENTER if no due date): ")
			dueStr, _ := reader.ReadString('\n')
			dueStr = strings.TrimSpace(dueStr)

			if dueStr != "" {
				due, err := time.Parse("2006-01-02", dueStr)
				if err != nil {
					nonFatalError(err)
				}
				yyyy, mm, dd := time.Now().Date()
				if due.Before(time.Date(yyyy, mm, dd, 0, 0, 0, 0, time.Now().Location())) {
					nonFatalError(errors.New("due date has passed already"))
				}

			}

			fmt.Println("Select context (Press ENTER if no context): ")
			for k, v := range contextMap {
				fmt.Println(fmt.Sprint(k, ". ", v, "\n"))
			}
			number, _ := reader.ReadString('\n')
			number = strings.TrimSpace(number)

			if dueStr == "" {
				if number == "" {
					if err := tasks.New(db, task, nil, nil, nil); err != nil {
						fatalError(err)
					}
				} else {
					contextID, err := strconv.Atoi(number)
					if err != nil {
						nonFatalError(invalidCommandFormat)
					}
					if err := tasks.New(db, task, nil, contextMap[uint32(contextID)], nil); err != nil {
						fatalError(err)
					}
				}
			} else {
				if number == "" {
					if err := tasks.New(db, task, nil, nil, &dueStr); err != nil {
						fatalError(err)
					}
				} else {
					contextID, err := strconv.Atoi(number)
					if err != nil {
						nonFatalError(invalidCommandFormat)
					}
					if err := tasks.New(db, task, nil, contextMap[uint32(contextID)], &dueStr); err != nil {
						fatalError(err)
					}
				}
			}

			fmt.Println("Created new next action")

		case "project":
			fmt.Print("Please enter new project name: ")
			projectName, _ := reader.ReadString('\n')
			projectName = strings.TrimSpace(projectName)

			exists, _, err := projects.ContentExists(db, projectName)
			if err != nil {
				fatalError(err)
			}
			if exists {
				fmt.Println("Project `" + projectName + "` already exists")
				return
			}

			if err := projects.New(db, projectName); err != nil {
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

			exists, _, err := contexts.ContentExists(db, context)
			if err != nil {
				fatalError(err)
			}
			if exists {
				fmt.Println("Context `" + context + "` already exists")
				return
			}

			if err := contexts.New(db, context); err != nil {
				fatalError(err)
			}
			fmt.Println("Created new context `" + context + "`")

		case "wait":
			fmt.Print("Please enter new task to wait for: ")
			waitAction, _ := reader.ReadString('\n')
			waitAction = strings.TrimSpace(waitAction)
			exists, _, err := waiting.ContentExists(db, waitAction)
			if err != nil {
				fatalError(err)
			}
			if exists {
				fmt.Println("Waiting action `" + waitAction + "` already exists")
				return
			}
			waiting.New(db, waitAction)
			fmt.Println("Created new wait action")

		case "someday":
			fmt.Print("Please enter new project name: ")

			reader := bufio.NewReader(os.Stdin)

			futureTask, _ := reader.ReadString('\n')
			futureTask = strings.TrimSpace(futureTask)

			exists, _, err := someday.ContentExists(db, futureTask)
			if err != nil {
				fatalError(err)
			}
			if exists {
				fmt.Println("Someday action `" + futureTask + "` already exists")
				return
			}

			if err := someday.New(db, futureTask); err != nil {
				fatalError(err)
			}

			fmt.Println("Someday action `" + futureTask + "` has been created")

		default:
			projectName := ""
			for i := 1; i < len(args); i++ {
				if i > 1 {
					projectName += " "
				}
				projectName += args[i]
			}

			var projectID uint32
			exists, projectID, err := projects.ContentExists(db, projectName)
			if err != nil {
				fatalError(err)
			}
			if !exists {
				nonFatalError(errors.New("No active project `" + projectName + "` exists"))
			}

			fmt.Print("Please enter new task for the project `" + projectName + "`: ")
			task, _ := reader.ReadString('\n')
			task = strings.TrimSpace(task)

			fmt.Print("Due date (YYYY-MM-DD) (Press ENTER if no due date): ")
			dueStr, _ := reader.ReadString('\n')
			dueStr = strings.TrimSpace(dueStr)

			if dueStr != "" {
				due, err := time.Parse("2006-01-02", dueStr)
				if err != nil {
					nonFatalError(err)
				}
				yyyy, mm, dd := time.Now().Date()
				if due.Before(time.Date(yyyy, mm, dd, 0, 0, 0, 0, time.Now().Location())) {
					nonFatalError(errors.New("due date has passed already"))
				}
			}

			fmt.Println("Please enter context (Press ENTER for no context): ")
			for k, v := range contextMap {
				fmt.Println(fmt.Sprint(k, ". ", v))
			}
			number, _ := reader.ReadString('\n')
			number = strings.TrimSpace(number)

			if dueStr == "" {
				if number == "" {
					if err := tasks.New(db, task, &projectID, nil, nil); err != nil {
						fatalError(err)
					}
				} else {
					contextID, err := strconv.Atoi(number)
					if err != nil {
						nonFatalError(invalidCommandFormat)
					}

					if err := tasks.New(db, task, &projectID, contextMap[uint32(contextID)], nil); err != nil {
						fatalError(err)
					}
				}
			} else {
				if number == "" {
					if err := tasks.New(db, task, &projectID, nil, &dueStr); err != nil {
						fatalError(err)
					}
				} else {
					contextID, err := strconv.Atoi(number)
					if err != nil {
						nonFatalError(invalidCommandFormat)
					}

					if err := tasks.New(db, task, &projectID, contextMap[uint32(contextID)], &dueStr); err != nil {
						fatalError(err)
					}
				}
			}

			fmt.Println("New task created for `" + projectName + "`")
		}

	case "done":
		if len(args) < 3 {
			nonFatalError(invalidCommandFormat)
		}
		switch args[1] {
		case "in":
			captureID, err := strconv.Atoi(args[2])
			if err != nil {
				nonFatalError(invalidCommandFormat)
			}

			exists, err := capture.IDExists(db, uint32(captureID))
			if err != nil {
				fatalError(err)
			}
			if !exists {
				fmt.Println("Capture item `" + args[2] + "` does not exist")
				return
			}

			if err := capture.Done(db, uint32(captureID)); err != nil {
				fatalError(err)
			}

			fmt.Println("Marked capture item `" + args[2] + "` as finished\n")

			if err := log.New(db, "capture", uint32(captureID)); err != nil {
				fatalError(err)
			}

		case "someday":
			if len(args) < 2 {
				nonFatalError(invalidCommandFormat)
			}
			id, err := strconv.Atoi(args[2])
			if err != nil {
				nonFatalError(invalidCommandFormat)
			}

			exists, err := someday.IDExists(db, uint32(id))
			if err != nil {
				fatalError(err)
			}
			if !exists {
				fmt.Println("Someday action `" + args[2] + "` does not exist")
				return
			}

			if err := someday.Done(db, uint32(id)); err != nil {
				fatalError(err)
			}

			fmt.Println("Marked someday task `" + args[2] + "` as done")

			if err := log.New(db, "someday", uint32(id)); err != nil {
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
					exists, _, err := projects.ContentExists(db, projectName)
					if err != nil {
						fatalError(err)
					}
					if exists {
						fmt.Println("Project `" + projectName + "` already exists")
						return
					}

					if err := projects.New(db, projectName); err != nil {
						fatalError(err)
					}

					fmt.Println("Project `" + projectName + "` has been created")
				case "y":
					task, err := someday.Get(db, uint32(id))
					if err != nil {
						fatalError(err)
					}

					exists, _, err := projects.ContentExists(db, task.Content)
					if err != nil {
						fatalError(err)
					}
					if exists {
						fmt.Println("Project `" + task.Content + "` already exists")
						return
					}

					if err := projects.New(db, task.Content); err != nil {
						fatalError(err)
					}

					fmt.Println("Project `" + task.Content + "` has been created")
				default:
					nonFatalError(invalidCommand, ans)
				}
			default:
				nonFatalError(invalidCommand, ans)
			}

		case "waiting":
			if len(args) < 2 {
				nonFatalError(invalidCommandFormat)
			}
			id, err := strconv.Atoi(args[2])
			if err != nil {
				nonFatalError(invalidCommandFormat)
			}
			exists, err := waiting.IDExists(db, uint32(id))
			if err != nil {
				fatalError(err)
			}
			if !exists {
				fmt.Println("Waiting action `" + args[2] + "` does not exist")
			}

			if err := waiting.Done(db, uint32(id)); err != nil {
				fatalError(err)
			}

			fmt.Println("Marked waiting task `" + args[2] + "` as done")

			if err := log.New(db, "waiting", uint32(id)); err != nil {
				fatalError(err)
			}

		case "task":
			taskID, err := strconv.Atoi(args[2])
			if err != nil {
				nonFatalError(invalidCommandFormat)
			}

			exists, err := tasks.IDExists(db, uint32(taskID))
			if err != nil {
				fatalError(err)
			}
			if !exists {
				fmt.Print(fmt.Sprint("Task ", args[2], " does not exist\n"))
			}

			if err := tasks.Done(db, uint32(taskID)); err != nil {
				fatalError(err)
			}

			task, err := tasks.Get(db, uint32(taskID))
			if err != nil {
				fatalError(err)
			}

			fmt.Println("Finished task `" + args[2] + "`\n`" + task.Content + "`")

			if err := log.New(db, "tasks", uint32(taskID)); err != nil {
				fatalError(err)
			}

		default:
			projectName := ""
			for i := 1; i < len(args); i++ {
				if i > 1 {
					projectName += " "
				}
				projectName += args[i]
			}
			exists, projectID, err := projects.ContentExists(db, projectName)
			if err != nil {
				fatalError(err)
			}
			if !exists {
				fmt.Println("No active project `" + projectName + "` exists")
				return
			}

			if err := projects.Done(db, projectID); err != nil {
				fatalError(err)
			}

			fmt.Println("Finished project `" + projectName + "`\n")

			if err := log.New(db, "projects", projectID); err != nil {
				fatalError(err)
			}
		}

	case "in":
		captureList, err := capture.GetActive(db)
		if err != nil {
			fatalError(err)
		}

		if len(captureList) == 0 {
			fmt.Println("No next actions for now")
		}

		for _, c := range captureList {
			fmt.Print(fmt.Sprint("- ID: ", c.ID, "\n", c.Content, "\n"))
		}

	case "waiting":
		waitList, err := waiting.GetActive(db)
		if err != nil {
			fatalError(err)
		}
		if len(waitList) == 0 {
			fmt.Println("No tasks to wait for now")
		}
		for _, w := range waitList {
			fmt.Println(fmt.Sprint("- ID: ", w.ID, "\n", w.Content))
		}

	case "someday":
		futureTasks, err := someday.GetActive(db)
		if err != nil {
			fatalError(err)
		}
		if len(futureTasks) == 0 {
			fmt.Println("No tasks for someday")
		}
		for _, s := range futureTasks {
			fmt.Print(fmt.Sprint("- ID: ", s.ID, "\n", s.Content, "\n"))
		}

	case "read":
		taskList, err := tasks.Read(db)
		if err != nil {
			fatalError(err)
		}

		if len(taskList) > 0 {
			for _, t := range taskList {
				fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
				if t.ProjectID != nil {
					project, err := projects.Get(db, *t.ProjectID)
					if err != nil {
						fatalError(err)
					}

					fmt.Print(fmt.Sprint("Project: ", project.Content, "\n"))
				}
				if t.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
				}
				if t.Due != nil {
					fmt.Print(fmt.Sprint("Due: ", *t.Due, "\n"))
				}
			}
		} else {
			somedayTasks, err := someday.Read(db)
			if err != nil {
				fatalError(err)
			}
			if len(somedayTasks) == 0 {
				fmt.Println("Nothing to read for now")
			} else {
				for _, t := range somedayTasks {
					fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
				}
			}
		}

	case "next":
		tasks, err := tasks.GetActiveNextActions(db)
		if err != nil {
			fatalError(err)
		}

		if len(tasks) == 0 {
			fmt.Println("No next actions for now")
		}

		for _, a := range tasks {
			fmt.Print(fmt.Sprint("- ID: ", a.ID, "\n", a.Content, "\n"))
			if a.Context != nil {
				fmt.Print(fmt.Sprint("Context: ", *a.Context, "\n"))
			}
		}

	case "all":
		if len(args) == 1 {
			noTasks := true

			calendarTasks, err := tasks.GetAllCalenderTasks(db)
			if err != nil {
				fatalError(err)
			}
			if len(calendarTasks) > 0 {
				noTasks = false
			}
			fmt.Println("CALENDAR")
			for _, t := range calendarTasks {
				fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
				if t.ProjectID != nil {
					project, err := projects.Get(db, *t.ProjectID)
					if err != nil {
						fatalError(err)
					}
					fmt.Print(fmt.Sprint("Project: ", project.Content, "\n"))
				}
				if t.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
				}
				if t.Due != nil {
					fmt.Print(fmt.Sprint("Due: ", *t.Due, "\n"))
				}
			}

			nextActions, err := tasks.GetActiveNextActions(db)
			if err != nil {
				fatalError(err)
			}

			if len(nextActions) > 0 {
				noTasks = false
				fmt.Println("\nNEXT ACTIONS")
			}
			for _, a := range nextActions {
				fmt.Print(fmt.Sprint("- ID: ", a.ID, "\n", a.Content, "\n"))
				if a.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *a.Context, "\n"))
				}
			}

			fmt.Println("\nPROJECTS")

			projectList, err := projects.GetActive(db)
			if err != nil {
				fatalError(err)
			}

			for _, project := range projectList {
				projectTasks, err := tasks.GetActiveProjectTasks(db, project.ID)
				if err != nil {
					fatalError(err)
				}

				if len(projectTasks) > 0 {
					noTasks = false
					fmt.Println(project.Content)
				}
				for _, t := range projectTasks {
					fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
					if t.Context != nil {
						fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
					}
				}
			}

			fmt.Println("\nWAITING")

			waitingTasks, err := waiting.GetActive(db)
			if err != nil {
				fatalError(err)
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
			for i := 1; i < len(args); i++ {
				if i > 1 {
					projectName += " "
				}
				projectName += args[i]
			}
			exists, projectID, err := projects.ContentExists(db, projectName)
			if err != nil {
				fatalError(err)
			}
			if !exists {
				fmt.Println("No active project `" + projectName + "` exists\n")
			}

			calendarTasks, err := tasks.GetAllProjectCalendarTasks(db, projectID)
			if err != nil {
				fatalError(err)
			}
			nonCalendarTasks, err := tasks.GetActiveProjectTasks(db, projectID)
			if err != nil {
				fatalError(err)
			}

			for _, t := range calendarTasks {
				fmt.Print(fmt.Sprint("- ID: ", t.ID, "\n", t.Content, "\n"))
				if t.Context != nil {
					fmt.Print(fmt.Sprint("Context: ", *t.Context, "\n"))
				}
				if t.Due != nil {
					fmt.Print(fmt.Sprint("Due: ", *t.Due, "\n"))
				}
			}
			for _, t := range nonCalendarTasks {
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
		if len(args) < 2 {
			nonFatalError(invalidCommandFormat)
		}

		switch args[1] {
		case "in":
			todo()

		case "task":
			todo()

		case "waiting":
			todo()

		case "someday":
			todo()

		case "project":
			todo()
		}

	case "clean":
		cnt, err := capture.Count(db)
		if err != nil {
			fatalError(err)
		}

		if cnt > 0 {
			fmt.Println(fmt.Sprint("You will be cleaning out `", cnt, "` items from in list. Continue? (y/n)"))
			var ans string
			fmt.Scanln(&ans)
			switch ans {
			case "n":
				break

			case "y":
				if err := capture.Clean(db); err != nil {
					fatalError(err)
				}

				fmt.Println(fmt.Sprint("Cleaned `", cnt, "` items from in list"))

			default:
				nonFatalError(invalidCommand, ans)
			}
		}

		cnt, err = tasks.CountCalendar(db)
		if err != nil {
			fatalError(err)
		}

		if cnt > 0 {
			fmt.Println(fmt.Sprint("You will be cleaning out `", cnt, "` calendar tasks. Continue? (y/n)"))
			var ans string
			fmt.Scanln(&ans)
			switch ans {
			case "n":
				break

			case "y":
				if err := tasks.CleanCalendar(db); err != nil {
					fatalError(err)
				}

				fmt.Println(fmt.Sprint("Cleaned `", cnt, "` tasks from calendar"))

			default:
				nonFatalError(invalidCommand, ans)
			}
		}

		cnt, err = tasks.CountNextActions(db)
		if err != nil {
			fatalError(err)
		}

		if cnt > 0 {
			fmt.Println(fmt.Sprint("You will be cleaning out `", cnt, "` next actions. Continue? (y/n)"))
			var ans string
			fmt.Scanln(&ans)
			switch ans {
			case "n":
				break

			case "y":
				if err := tasks.CleanNextActions(db); err != nil {
					fatalError(err)
				}

				fmt.Println(fmt.Sprint("Cleaned `", cnt, "` next actions"))

			default:
				nonFatalError(invalidCommand, ans)
			}
		}

		cnt, err = tasks.CountProjectTasks(db)
		if err != nil {
			fatalError(err)
		}

		if cnt > 0 {
			fmt.Println(fmt.Sprint("You will be cleaning out `", cnt, "` project tasks. Continue? (y/n)"))
			var ans string
			fmt.Scanln(&ans)
			switch ans {
			case "n":
				break

			case "y":
				if err := tasks.CleanProjectTasks(db); err != nil {
					fatalError(err)
				}

				fmt.Println(fmt.Sprint("Cleaned `", cnt, "` project tasks"))

			default:
				nonFatalError(invalidCommand, ans)
			}
		}

	case "undo":
		a, exists, err := log.Get(db)
		if err != nil {
			fatalError(err)
		}
		if !exists {
			fmt.Println("No actions to undo")
		}

		fmt.Print(fmt.Sprint("You will be marking `", a.RowID, "` in `", a.TableName, "` as undone. Continue? (y/n)\n"))
		var ans string
		fmt.Scanln(&ans)
		switch ans {
		case "n":
		case "y":
			if err := log.Undo(db, a.TableName, a.ID, a.RowID); err != nil {
				fatalError(err)
			}
			fmt.Print(fmt.Sprint("Marked `", a.RowID, "` in `", a.TableName, "` as undone\n"))
		default:
			nonFatalError(invalidCommand, ans)
		}

	case "review":
		thresh := time.Now().AddDate(0, 0, -7)

		fmt.Println("FINISHED PROJECTS")
		finishedProjects, err := projects.Review(db, thresh)
		if err != nil {
			fatalError(err)
		}

		for _, p := range finishedProjects {
			fmt.Print(fmt.Sprint("- ", p.Content, "\n"))
		}

		fmt.Println("\nACTIVE PROJECTS")

		activeProjects, err := projects.GetActive(db)
		if err != nil {
			fatalError(err)
		}
		for _, p := range activeProjects {
			fmt.Print(fmt.Sprint("- ", p.Content, "\n"))
		}

		fmt.Println("\nMISSED CALENDAR TASKS")
		pendingCalendarTasks, err := tasks.PendingCalendar(db, thresh)
		if err != nil {
			fatalError(err)
		}

		for _, t := range pendingCalendarTasks {
			fmt.Print(fmt.Sprint("- ", t.Content, "\nDue: ", *t.Due, "\n"))
			if t.ProjectID != nil {
				p, err := projects.Get(db, *t.ProjectID)
				if err != nil {
					fatalError(err)
				}

				fmt.Print(fmt.Sprint("Project: ", p.Content, "\n"))
			}
		}

		fmt.Println("\nFINISHED TASKS")

		finishedTasks, err := tasks.Review(db, thresh)
		if err != nil {
			fatalError(err)
		}

		finishedWaitingTasks, err := waiting.Review(db, thresh)
		if err != nil {
			fatalError(err)
		}

		for i := 0; i < 7; i = i + 1 {
			day := thresh.AddDate(0, 0, i)
			tasks := finishedTasks[day]
			waitingTasks := finishedWaitingTasks[day]
			if len(tasks) > 0 || len(waitingTasks) > 0 {
				fmt.Print(fmt.Sprint("Date: ", day.Format("2006-01-02"), "\n"))
			}
			for _, t := range tasks {
				fmt.Print(fmt.Sprint("- ", t.Content, "\n"))
				if t.Due != nil {
					fmt.Print(fmt.Sprint("Due: ", *t.Due, "\n"))
				}
				if t.ProjectID != nil {
					p, err := projects.Get(db, *t.ProjectID)
					if err != nil {
						fatalError(err)
					}

					fmt.Print(fmt.Sprint("Project: ", p.Content, "\n"))
				}
			}

			for _, w := range waitingTasks {
				fmt.Print(fmt.Sprint("- ", w.Content, "\n"))
			}
		}

	default:
		if len(args) == 1 && args[0] == "projects" {
			projects, err := projects.GetActive(db)
			if err != nil {
				fatalError(err)
			}
			if len(projects) == 0 {
				fmt.Println("No active projects")
			}
			for _, p := range projects {
				fmt.Print(fmt.Sprint("- ID: ", p.ID, "\n", p.Content, "\n"))
			}
		} else if len(args) == 1 && args[0] == "contexts" {
			contextList, err := contexts.GetAll(db)
			if err != nil {
				fatalError(err)
			}
			for _, c := range contextList {
				fmt.Print(fmt.Sprint("- ID: ", c.ID, "\n", c.Content, "\n"))
			}
		} else {
			projectName := ""
			for i := range args {
				if i > 0 {
					projectName += " "
				}
				projectName += args[i]
			}
			exists, projectID, err := projects.ContentExists(db, projectName)
			if err != nil {
				fatalError(err)
			}
			if !exists {
				fmt.Println("No active project `" + projectName + "` exists\n")
			}

			calendarTasks, err := tasks.GetTodayProjectCalendarTasks(db, projectID)
			if err != nil {
				fatalError(err)
			}

			nonCalendarTasks, err := tasks.GetActiveProjectTasks(db, projectID)
			if err != nil {
				fatalError(err)
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

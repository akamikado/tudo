package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"tudo/cmd"
	"tudo/database"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Could not access user file system")
		os.Exit(1)
	}

	tudoDir := homeDir + "/.tudo"
	_, err = os.Stat(tudoDir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(tudoDir, os.ModeDir); err != nil {
			fmt.Println("Could not create .tudo directory")
			os.Exit(1)
		}
		fmt.Println("Created directory `" + tudoDir + "`")
	} else if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := os.Chmod(tudoDir, 0755); err != nil {
		fmt.Println("Could not change permissions of .tudo directory")
		os.Exit(1)
	}

	dbFile := tudoDir + "/tudo.db"
	_, err = os.Stat(dbFile)
	if errors.Is(err, fs.ErrNotExist) {
		err = database.Setup(dbFile)
		if err != nil {
			fmt.Println("Could not setup database :" + err.Error())
			os.Exit(1)
		}
		fmt.Println("Finished setting up sqlite db file `" + dbFile + "`")
	} else if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	cmd.ParseArgs(dbFile)
}

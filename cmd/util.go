package cmd

import (
	"fmt"
	"os"
)

func todo(s string) {
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

package log

import (
	"fmt"
	"os"

	. "github.com/logrusorgru/aurora"
)

// CheckIfError should be used to naively panics if an error is not nil.
func CheckIfError(err error) {
	if err == nil {
		return
	}

	Error(err.Error())
}

// CheckIfErrorMessage should be used to naively panics if an error is not nil.
func CheckIfErrorMessage(err error, message string) {
	if err == nil {
		return
	}

	Error(message, err.Error())
}

// Error should be used to describe error messages. It will finish program execution
func Error(format string, args ...interface{}) {
	log(Bold(Red(fmt.Sprintf(format, args...))))
	os.Exit(1)
}

// Info should be used to describe the example commands that are about to run.
func Info(format string, args ...interface{}) {
	log(Bold(Blue(fmt.Sprintf(format, args...))))
}

// Log should be used to regular messages
func Log(format string, args ...interface{}) {
	log(White(fmt.Sprintf(format, args...)))
}

// Success should be used to describe success messages.
func Success(format string, args ...interface{}) {
	log(Bold(Green(fmt.Sprintf(format, args...))))
}

// Warn should be used to display a warning
func Warn(format string, args ...interface{}) {
	log(Bold(Yellow(fmt.Sprintf(format, args...))))
}

func log(args ...interface{}) {
	fmt.Printf("%s\n", args)
}

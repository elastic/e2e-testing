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
	os.Exit(1)
}

// CheckIfErrorMessage should be used to naively panics if an error is not nil.
func CheckIfErrorMessage(err error, message string) {
	if err == nil {
		return
	}

	Error(message, err.Error())
	os.Exit(1)
}

// Error should be used to describe error messages.
func Error(format string, args ...interface{}) {
	fmt.Printf("%s\n", Bold(Red(fmt.Sprintf(format, args...))))
}

// Info should be used to describe the example commands that are about to run.
func Info(format string, args ...interface{}) {
	fmt.Printf("%s\n", Bold(Blue(fmt.Sprintf(format, args...))))
}

// Success should be used to describe success messages.
func Success(format string, args ...interface{}) {
	fmt.Printf("%s\n", Bold(Green(fmt.Sprintf(format, args...))))
}

// Warn should be used to display a warning
func Warn(format string, args ...interface{}) {
	fmt.Printf("%s\n", Bold(Yellow(fmt.Sprintf(format, args...))))
}

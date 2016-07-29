package helpers

import "fmt"

type ErrorSet map[string]int

func (e ErrorSet) Error() string {
	message := "The following errors occurred:\n"
	for key, val := range e {
		message += fmt.Sprintf("  %s : %d\n", key, val)
	}
	return message
}

func (e ErrorSet) Add(err error) {
	e[err.Error()] = e[err.Error()] + 1
}

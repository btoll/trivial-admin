package main

import "fmt"

type DBError struct {
	Code    int
	Message string
}

func (e *DBError) Error() string {
	return fmt.Sprintf("code %d: err %s", e.Code, e.Message)
}

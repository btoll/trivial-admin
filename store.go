package main

type Store interface {
	Get(string) error
	Write(string, string) error
}

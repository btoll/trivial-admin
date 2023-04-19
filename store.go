package main

type Store interface {
	Create() error
	Read() error
	Update(string) error
	Delete() error
}

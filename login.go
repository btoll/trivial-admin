package main

import (
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (l *Login) Create() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(l.Password), 8)
	// Go creates prepared statements for you under the covers.  A simple db.Query(sql, param1, param2),
	// for example, works by preparing the sql, then executing it with the parameters and finally
	// closing the statement.
	// http://go-database-sql.org/prepared.html#avoiding-prepared-statements
	_, err = db.Query("INSERT INTO users VALUES (DEFAULT, $1, $2)", l.Username, string(hashedPassword))
	if err != nil {
		// https://pkg.go.dev/github.com/lib/pq?utm_source=godoc#hdr-Errors
		// https://www.postgresql.org/docs/current/errcodes-appendix.html
		if err, ok := err.(*pq.Error); ok {
			var duplicateKey pq.ErrorCode = "23505"
			//			fmt.Println("err.Code", err.Code)
			if duplicateKey == err.Code {
				return &DBError{
					Code:    400,
					Message: "Username already exists",
				}
			} else {
				// If the `users` table doesn't exist, the `err.Code` will be "42P01".
				return &DBError{
					Code:    502,
					Message: "Bad Gateway",
				}
			}
		}
	}
	return nil
}

func (l *Login) Read() error {
	result := db.QueryRow("SELECT username, password FROM users WHERE username=$1", l.Username)
	storedLogin := &Login{}
	err := result.Scan(&storedLogin.Username, &storedLogin.Password)
	if err != nil {
		return &DBError{
			Code:    403,
			Message: "Unauthorized",
		}
	}
	err = bcrypt.CompareHashAndPassword([]byte(storedLogin.Password), []byte(l.Password))
	if err != nil {
		return &DBError{
			Code:    403,
			Message: "Unauthorized",
		}
	}
	return nil
}

func (l *Login) Update(v string) error {
	return nil
}

func (l *Login) Delete() error {
	return nil
}

package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/ggetzie/badwords_be/internal/data"
)

func main() {
	var db data.DBConfig
	var email string
	var newPassword string
	var displayName string
	var fullName string

	flag.StringVar(&db.DSN, "db-dsn", "", "Postgresql DSN")
	flag.StringVar(&email, "email", "", "User email")
	flag.StringVar(&newPassword, "password", "", "password")
	flag.StringVar(&displayName, "display-name", "", "Display name")
	flag.StringVar(&fullName, "full-name", "", "Full name")
	flag.Parse()

	db.MaxOpenConns = 25
	db.MinConns = 4
	db.MaxIdleTime = 15 * time.Minute

	dbPool, err := data.OpenDB(db)
	if err != nil {
		panic(err)
	}
	defer dbPool.Close()

	models := data.NewModels(dbPool)

	user := &data.User{
		Email:       email,
		FullName:    fullName,
		DisplayName: displayName,
		Activated:   true,
	}
	err = user.Password.Set(newPassword)
	if err != nil {
		panic(err)
	}

	err = models.Users.Insert(user)
	if err != nil {
		panic(err)
	}

	err = models.Permissions.AddForUser(user.ID, data.StandardPermissions...)
	if err != nil {
		panic(err)
	}

	fmt.Println("user added successfully")
}

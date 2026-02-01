package main

import (
	"flag"
	"log"
	"time"

	"github.com/ggetzie/badwords_be/internal/data"
	"github.com/ggetzie/badwords_be/internal/validator"
)

func main() {
	var db data.DBConfig
	var email string
	var newPassword string

	flag.StringVar(&db.DSN, "db-dsn", "", "Postgresql DSN")
	flag.StringVar(&email, "email", "", "User email")
	flag.StringVar(&newPassword, "new-password", "", "New password")
	flag.Parse()

	db.MaxOpenConns = 25
	db.MinConns = 4
	db.MaxIdleTime = 15 * time.Minute

	if email == "" || newPassword == "" {
		log.Fatal("Email and new password are required")
	}

	dbPool, err := data.OpenDB(db)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer dbPool.Close()

	models := data.NewModels(dbPool)

	user, err := models.Users.GetByEmail(email)
	if err != nil {
		log.Fatalf("Error fetching user: %v", err)
	}

	v := validator.New()
	if data.ValidatePasswordPlaintext(v, newPassword); !v.Valid() {
		log.Fatalf("Invalid password: %v", v.Errors)
	}

	err = user.Password.Set(newPassword)
	if err != nil {
		log.Fatalf("Error setting new password: %v", err)
	}

	err = models.Users.Update(user)
	if err != nil {
		log.Fatalf("Error updating user password: %v", err)
	}

	log.Printf("Password for user %s has been updated successfully.", email)
}

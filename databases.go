package main

import (
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDatabase() *gorm.DB {
	if _, err := os.Stat("session.db"); os.IsNotExist(err) {
		file, err := os.Create("session.db")
		if err != nil {
			log.Fatalf("failed to create session.db: %v", err)
		}
		file.Close()
		log.Println("session.db created successfully.")
	}

	db, err := gorm.Open(sqlite.Open("session.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	err = db.AutoMigrate(&Notified{})
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	return db
}

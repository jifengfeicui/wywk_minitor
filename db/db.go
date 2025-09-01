package db

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	. "wywk/models" // Import models from the models package
)

func InitDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("wywk.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 全局不打印 SQL
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	db = db.Debug() // Enable GORM debug mode

	// Auto-migrate the schema
	log.Println("Auto-migrating database schema...")
	err = db.AutoMigrate(&Shop{}, &Room{}, &Snapshot{}, &RoomSnapshot{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}
	log.Println("Database migration completed.")

	return db
}

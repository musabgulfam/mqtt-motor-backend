// database.go - Handles database connection and setup

package database // Declares the package name

import ( // Import required packages
	"go-mqtt-backend/models" // User model

	"gorm.io/driver/sqlite" // SQLite driver for GORM
	"gorm.io/gorm"          // GORM ORM
)

var DB *gorm.DB // Global variable to hold the database connection (pointer to gorm.DB)

func Connect(dbPath string) error { // Connect opens the database and runs migrations
	var err error                                            // Declare error variable
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{}) // Open SQLite DB
	if err != nil {                                          // If error, return it
		return err
	}
	return DB.AutoMigrate(&models.User{}, &models.DeviceActivation{}) // Auto-migrate the User model (create table if needed)
}

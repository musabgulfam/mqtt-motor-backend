// database.go - Handles database connection and setup

package database // Declares the package name

import ( // Import required packages
	"go-mqtt-backend/config" // Project config
	"go-mqtt-backend/models" // User model

	"golang.org/x/crypto/bcrypt" // Password hashing
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
	
	// Auto-migrate the User model (create table if needed)
	if err := DB.AutoMigrate(&models.User{}); err != nil {
		return err
	}
	
	// Create default admin user if configured
	return createDefaultAdmin()
}

// createDefaultAdmin - Creates a default admin user if configured and none exists
// This uses environment variables for security instead of hardcoded credentials
func createDefaultAdmin() error {
	cfg := config.Load() // Load configuration
	
	// Only create admin if explicitly configured
	if !cfg.CreateAdmin {
		return nil
	}
	
	// Check if any admin user exists
	var count int64
	DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	
	if count == 0 {
		// Create default admin user using config values
		hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		
		adminUser := models.User{
			Email:    cfg.AdminEmail,
			Password: string(hash),
			Role:     "admin",
		}
		
		if err := DB.Create(&adminUser).Error; err != nil {
			return err
		}
	}
	
	return nil
}

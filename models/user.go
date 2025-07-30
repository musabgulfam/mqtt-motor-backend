// user.go - Defines the User model for the database

package models // Declares the package name

type User struct { // User struct represents a user in the database
	ID       uint   `gorm:"primaryKey"`      // Unique user ID (primary key)
	Email    string `gorm:"unique;not null"` // User's email (must be unique, cannot be null)
	Password string `gorm:"not null"`        // Hashed password (cannot be null)
	Role     string `gorm:"default:'user'"`  // User role (user/admin)
}

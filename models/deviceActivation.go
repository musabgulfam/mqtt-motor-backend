package models

import "time"

type DeviceActivation struct {
	ID        uint          `gorm:"primaryKey"`                                                       // Unique ID
	UserID    uint          `gorm:"not null"`                                                         // Foreign key to users table
	User      User          `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"` // Foreign key constraint
	RequestAt time.Time     // When request was made
	Duration  time.Duration // For how long the device was active
}

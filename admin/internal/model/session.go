package model

import (
	"time"

	"gorm.io/gorm"
)

type Session struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64          `gorm:"not null;index" json:"user_id"`
	Token     string         `gorm:"size:512;uniqueIndex;not null" json:"token"`
	IP        string         `gorm:"size:50" json:"ip"`
	UserAgent string         `gorm:"size:512" json:"user_agent,omitempty"`
	ExpiresAt time.Time      `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func ListSessionsByUser(db *gorm.DB, userID int64) ([]Session, error) {
	var sessions []Session
	err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func CreateSession(db *gorm.DB, session *Session) error {
	return db.Create(session).Error
}

func DeleteSession(db *gorm.DB, id int64, userID int64) error {
	result := db.Where("id = ? AND user_id = ?", id, userID).Delete(&Session{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func DeleteAllUserSessions(db *gorm.DB, userID int64) error {
	return db.Where("user_id = ?", userID).Delete(&Session{}).Error
}

func CleanupExpiredSessions(db *gorm.DB) error {
	return db.Where("expires_at < ?", time.Now()).Delete(&Session{}).Error
}
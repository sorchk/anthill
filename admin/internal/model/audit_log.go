package model

import (
	"time"

	"gorm.io/gorm"
)

type AuditLog struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64          `gorm:"not null;index" json:"user_id"`
	Username  string         `gorm:"size:255;not null" json:"username"`
	Action    string         `gorm:"size:50;not null" json:"action"`
	Resource  string         `gorm:"size:255;not null" json:"resource"`
	Method    string         `gorm:"size:10;not null" json:"method"`
	Path      string         `gorm:"size:512;not null" json:"path"`
	IP        string         `gorm:"size:50" json:"ip"`
	Status    int            `json:"status"`
	Details   string         `gorm:"type:text" json:"details,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type AuditLogListResult struct {
	Data      []AuditLog `json:"data"`
	Total     int64      `json:"total"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
}

func ListAuditLogs(db *gorm.DB, page, pageSize int) (*AuditLogListResult, error) {
	var logs []AuditLog
	var total int64

	db.Model(&AuditLog{}).Count(&total)

	offset := (page - 1) * pageSize
	err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error
	if err != nil {
		return nil, err
	}

	return &AuditLogListResult{
		Data:     logs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func GetAuditLogByID(db *gorm.DB, id int64) (*AuditLog, error) {
	var log AuditLog
	err := db.First(&log, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, err
	}
	return &log, err
}

func CreateAuditLog(db *gorm.DB, log *AuditLog) error {
	return db.Create(log).Error
}
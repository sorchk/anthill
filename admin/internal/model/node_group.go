package model

import (
	"time"

	"gorm.io/gorm"
)

type NodeGroup struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"uniqueIndex;size:255;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func ListNodeGroups(db *gorm.DB) ([]NodeGroup, error) {
	var groups []NodeGroup
	err := db.Order("id").Find(&groups).Error
	return groups, err
}

func CreateNodeGroup(db *gorm.DB, group *NodeGroup) error {
	return db.Create(group).Error
}

func DeleteNodeGroup(db *gorm.DB, id int64) error {
	result := db.Delete(&NodeGroup{}, id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}
package model

import (
	"time"

	"gorm.io/gorm"
)

type DeployTask struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeID      int64          `json:"node_id,omitempty"`
	SSHHost     string         `gorm:"size:255;not null" json:"ssh_host"`
	SSHPort     int            `gorm:"default:22" json:"ssh_port"`
	SSHUser     string         `gorm:"size:255;not null" json:"ssh_user"`
	SSHKey      string         `gorm:"type:text" json:"ssh_key,omitempty"`
	Status      string         `gorm:"size:50;default:pending" json:"status"`
	Log         string         `gorm:"type:text" json:"log,omitempty"`
	CreatedBy   int64          `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type DeployRequest struct {
	SSHHost          string   `json:"ssh_host" binding:"required"`
	SSHPort          int      `json:"ssh_port"`
	SSHUser          string   `json:"ssh_user" binding:"required"`
	SSHPassword      string   `json:"ssh_password,omitempty"`
	SSHKey           string   `json:"ssh_key,omitempty"`
	NodeVersion      string   `json:"node_version" binding:"required"`
	PreinstallPlugins []string `json:"preinstall_plugins,omitempty"`
}

func ListDeployTasks(db *gorm.DB, limit int) ([]DeployTask, error) {
	var tasks []DeployTask
	query := db.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&tasks).Error
	return tasks, err
}

func GetDeployTaskByID(db *gorm.DB, id int64) (*DeployTask, error) {
	var task DeployTask
	err := db.First(&task, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, err
	}
	return &task, err
}

func CreateDeployTask(db *gorm.DB, task *DeployTask) error {
	return db.Create(task).Error
}

func AppendDeployTaskLog(db *gorm.DB, id int64, msg string) error {
	return db.Model(&DeployTask{}).Where("id = ?", id).Update("log", gorm.Expr("log || ? || char(10)", msg)).Error
}

func UpdateDeployTaskStatus(db *gorm.DB, id int64, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if status == "completed" || status == "failed" {
		updates["completed_at"] = time.Now()
	}
	return db.Model(&DeployTask{}).Where("id = ?", id).Updates(updates).Error
}
package model

import (
	"time"

	"gorm.io/gorm"
)

type Deployment struct {
	ID         int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeID     int64          `gorm:"not null;index" json:"node_id"`
	PluginID   int64          `gorm:"not null;index" json:"plugin_id"`
	Version    string         `gorm:"size:50;not null" json:"version"`
	Status     string         `gorm:"size:50;default:pending" json:"status"`
	Result     string         `gorm:"type:text" json:"result,omitempty"`
	DeployedBy int64          `gorm:"not null" json:"deployed_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	Node   *Node   `gorm:"foreignKey:NodeID" json:"node,omitempty"`
	Plugin *Plugin `gorm:"foreignKey:PluginID" json:"plugin,omitempty"`
}

type DeploymentRequest struct {
	NodeIDs  []int64 `json:"node_ids" binding:"required"`
	PluginID int64   `json:"plugin_id" binding:"required"`
}

type DeploymentWithNames struct {
	Deployment
	NodeName   string `json:"node_name"`
	PluginName string `json:"plugin_name"`
}

func ListDeployments(db *gorm.DB, nodeID, pluginID string, limit int) ([]DeploymentWithNames, error) {
	query := db.Model(&Deployment{}).
		Select("deployments.*, nodes.name as node_name, plugins.name as plugin_name").
		Joins("LEFT JOIN nodes ON deployments.node_id = nodes.id").
		Joins("LEFT JOIN plugins ON deployments.plugin_id = plugins.id").
		Order("deployments.created_at DESC")

	if nodeID != "" {
		query = query.Where("deployments.node_id = ?", nodeID)
	}
	if pluginID != "" {
		query = query.Where("deployments.plugin_id = ?", pluginID)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	var deployments []DeploymentWithNames
	err := query.Scan(&deployments).Error
	return deployments, err
}

func GetDeploymentByID(db *gorm.DB, id int64) (*Deployment, error) {
	var deployment Deployment
	err := db.First(&deployment, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, err
	}
	return &deployment, err
}

func CreateDeployment(db *gorm.DB, deployment *Deployment) error {
	return db.Create(deployment).Error
}

func UpdateDeploymentStatus(db *gorm.DB, id int64, status, result string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if result != "" {
		updates["result"] = result
	}
	if status == "completed" || status == "failed" {
		updates["updated_at"] = time.Now()
	}
	return db.Model(&Deployment{}).Where("id = ?", id).Updates(updates).Error
}

func CancelDeployment(db *gorm.DB, id int64) error {
	result := db.Model(&Deployment{}).Where("id = ? AND status = ?", id, "pending").Updates(map[string]interface{}{
		"status":     "cancelled",
		"updated_at": time.Now(),
	})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}
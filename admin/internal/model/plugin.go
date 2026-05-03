package model

import "time"

type Plugin struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
	PluginType  string    `json:"plugin_type"`
	Checksum    string    `json:"checksum"`
	UploadedBy  int64     `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
}
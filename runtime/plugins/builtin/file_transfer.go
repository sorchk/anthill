package builtin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileTransferPlugin struct {
	uploadDir string
	chunkSize int64
}

func NewFileTransferPlugin(uploadDir string) *FileTransferPlugin {
	os.MkdirAll(uploadDir, 0755)
	return &FileTransferPlugin{
		uploadDir: uploadDir,
		chunkSize: 1024 * 1024,
	}
}

type FileInfo struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Mode      string    `json:"mode"`
	Modified  time.Time `json:"modified"`
	IsDir     bool      `json:"is_dir"`
}

type TransferProgress struct {
	SessionID    string `json:"session_id"`
	TotalChunks  int    `json:"total_chunks"`
	Completed    int    `json:"completed"`
	BytesTotal   int64  `json:"bytes_total"`
	BytesDone    int64  `json:"bytes_done"`
	Speed        string `json:"speed"`
	ETA          string `json:"eta"`
}

type UploadRequest struct {
	RemotePath string `json:"remote_path"`
	FileName   string `json:"file_name"`
}

func (p *FileTransferPlugin) Invoke(method string, args json.RawMessage) (json.RawMessage, error) {
	switch method {
	case "ls":
		return p.handleLs(args)
	case "mkdir":
		return p.handleMkdir(args)
	case "rm":
		return p.handleRm(args)
	case "upload":
		return p.handleUpload(args)
	case "download":
		return p.handleDownload(args)
	case "rename":
		return p.handleRename(args)
	case "stat":
		return p.handleStat(args)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func (p *FileTransferPlugin) handleLs(args json.RawMessage) (json.RawMessage, error) {
	var req struct{ Path string }
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	if req.Path == "" {
		req.Path = "."
	}

	entries, err := os.ReadDir(req.Path)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		info, _ := entry.Info()
		files = append(files, FileInfo{
			Name:     entry.Name(),
			IsDir:    entry.IsDir(),
			Size:     0,
			Modified: time.Now(),
		})
		if info != nil {
			files[len(files)-1].Size = info.Size()
			files[len(files)-1].Modified = info.ModTime()
		}
	}

	return json.Marshal(files)
}

func (p *FileTransferPlugin) handleMkdir(args json.RawMessage) (json.RawMessage, error) {
	var req struct{ Path string }
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(req.Path, 0755); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]bool{"success": true})
}

func (p *FileTransferPlugin) handleRm(args json.RawMessage) (json.RawMessage, error) {
	var req struct{ Path string }
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	if err := os.RemoveAll(req.Path); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]bool{"success": true})
}

func (p *FileTransferPlugin) handleUpload(args json.RawMessage) (json.RawMessage, error) {
	var req UploadRequest
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	destPath := filepath.Join(p.uploadDir, req.RemotePath)

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]interface{}{
		"success":    true,
		"path":       destPath,
		"session_id": fmt.Sprintf("upload-%d", time.Now().Unix()),
	})
}

func (p *FileTransferPlugin) handleDownload(args json.RawMessage) (json.RawMessage, error) {
	var req struct{ Path string }
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	info, err := os.Stat(req.Path)
	if err != nil {
		return nil, err
	}

	return json.Marshal(map[string]interface{}{
		"path":          req.Path,
		"size":          info.Size(),
		"last_modified": info.ModTime(),
	})
}

func (p *FileTransferPlugin) handleRename(args json.RawMessage) (json.RawMessage, error) {
	var req struct{ OldPath, NewPath string }
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	if err := os.Rename(req.OldPath, req.NewPath); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]bool{"success": true})
}

func (p *FileTransferPlugin) handleStat(args json.RawMessage) (json.RawMessage, error) {
	var req struct{ Path string }
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	info, err := os.Stat(req.Path)
	if err != nil {
		return nil, err
	}

	return json.Marshal(FileInfo{
		Name:     info.Name(),
		Size:     info.Size(),
		Mode:     info.Mode().String(),
		Modified: info.ModTime(),
		IsDir:    info.IsDir(),
	})
}
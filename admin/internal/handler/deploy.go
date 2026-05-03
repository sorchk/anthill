package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"

	"anthill/admin/internal/model"
)

type DeployHandler struct {
	DB         *sql.DB
	binaryPath string
}

func NewDeployHandler(db *sql.DB) *DeployHandler {
	os.MkdirAll("./data/binaries", 0755)
	return &DeployHandler{
		DB:         db,
		binaryPath: "./data/binaries",
	}
}

func (h *DeployHandler) CreateTask(c *gin.Context) {
	var req model.DeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SSHPort == 0 {
		req.SSHPort = 22
	}

	result, err := h.DB.Exec(`
		INSERT INTO deploy_tasks (ssh_host, ssh_port, ssh_user, status, created_at)
		VALUES (?, ?, ?, 'pending', CURRENT_TIMESTAMP)
	`, req.SSHHost, req.SSHPort, req.SSHUser)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	taskID, _ := result.LastInsertId()

	go h.executeDeploy(int64(taskID), &req)

	c.JSON(http.StatusCreated, gin.H{"task_id": taskID})
}

func (h *DeployHandler) GetTask(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var task model.DeployTask
	err := h.DB.QueryRow(`
		SELECT id, ssh_host, ssh_port, ssh_user, status, log, created_at, completed_at
		FROM deploy_tasks WHERE id = ?
	`, id).Scan(&task.ID, &task.SSHHost, &task.SSHPort, &task.SSHUser,
		&task.Status, &task.Log, &task.CreatedAt, &task.CompletedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *DeployHandler) ListTasks(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT id, ssh_host, ssh_port, ssh_user, status, created_at
		FROM deploy_tasks ORDER BY created_at DESC LIMIT 50
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var tasks []model.DeployTask
	for rows.Next() {
		var t model.DeployTask
		rows.Scan(&t.ID, &t.SSHHost, &t.SSHPort, &t.SSHUser, &t.Status, &t.CreatedAt)
		tasks = append(tasks, t)
	}

	c.JSON(http.StatusOK, tasks)
}

func (h *DeployHandler) executeDeploy(taskID int64, req *model.DeployRequest) {
	h.appendLog(taskID, "Starting deployment...")

	config := &ssh.ClientConfig{
		User: req.SSHUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout: 30 * time.Second,
	}

	if req.SSHKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(req.SSHKey))
		if err != nil {
			h.appendLog(taskID, fmt.Sprintf("Failed to parse SSH key: %v", err))
			h.updateStatus(taskID, "failed")
			return
		}
		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	} else if req.SSHPassword != "" {
		config.Auth = append(config.Auth, ssh.Password(req.SSHPassword))
	}

	addr := fmt.Sprintf("%s:%d", req.SSHHost, req.SSHPort)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		h.appendLog(taskID, fmt.Sprintf("Failed to connect: %v", err))
		h.updateStatus(taskID, "failed")
		return
	}
	defer client.Close()

	h.appendLog(taskID, "Connected via SSH")

	binName := fmt.Sprintf("anthill-runtime-%s", req.NodeVersion)
	remotePath := "/tmp/" + binName

	h.appendLog(taskID, "Uploading binary...")

	localBin := filepath.Join(h.binaryPath, binName)
	if _, err := os.Stat(localBin); os.IsNotExist(err) {
		localBin = "./anthill-runtime"
	}

	err = h.uploadFile(client, localBin, remotePath)
	if err != nil {
		h.appendLog(taskID, fmt.Sprintf("Upload failed: %v", err))
		h.updateStatus(taskID, "failed")
		return
	}

	h.appendLog(taskID, "Binary uploaded")

	session, err := client.NewSession()
	if err != nil {
		h.appendLog(taskID, fmt.Sprintf("Failed to create session: %v", err))
		h.updateStatus(taskID, "failed")
		return
	}
	defer session.Close()

	cmd := fmt.Sprintf("chmod +x %s && mv %s /usr/local/bin/anthill-runtime", remotePath, remotePath)
	if err := session.Run(cmd); err != nil {
		h.appendLog(taskID, fmt.Sprintf("Failed to install: %v", err))
		h.updateStatus(taskID, "failed")
		return
	}

	h.appendLog(taskID, "Binary installed to /usr/local/bin/anthill-runtime")

	if len(req.PreinstallPlugins) > 0 {
		h.appendLog(taskID, "Preinstalling plugins...")
		for _, plugin := range req.PreinstallPlugins {
			h.appendLog(taskID, fmt.Sprintf("  - %s", plugin))
		}
	}

	h.appendLog(taskID, "Deployment complete!")
	h.updateStatus(taskID, "completed")
}

func (h *DeployHandler) uploadFile(client *ssh.Client, localPath, remotePath string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()

		cmd := exec.Command("tar", "czf", "-", "-C", filepath.Dir(localPath), filepath.Base(localPath))
		cmd.Stdout = w
		cmd.Run()
	}()

	return session.Run(fmt.Sprintf("tar xzf - -C %s", filepath.Dir(remotePath)))
}

func (h *DeployHandler) appendLog(taskID int64, msg string) {
	h.DB.Exec("UPDATE deploy_tasks SET log = log || ? || char(10) WHERE id = ?", msg, taskID)
}

func (h *DeployHandler) updateStatus(taskID int64, status string) {
	if status == "completed" || status == "failed" {
		h.DB.Exec("UPDATE deploy_tasks SET status = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?", status, taskID)
	} else {
		h.DB.Exec("UPDATE deploy_tasks SET status = ? WHERE id = ?", status, taskID)
	}
}
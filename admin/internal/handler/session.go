package handler

import (
    "database/sql"
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"

    "anthill/admin/internal/model"
)

type SessionHandler struct {
    DB        *sql.DB
    JWTSecret []byte
}

func NewSessionHandler(db *sql.DB, jwtSecret []byte) *SessionHandler {
    return &SessionHandler{DB: db, JWTSecret: jwtSecret}
}

func (h *SessionHandler) List(c *gin.Context) {
    userID, _ := c.Get("user_id")

    rows, err := h.DB.Query(`
        SELECT id, user_id, ip, user_agent, expires_at, created_at
        FROM sessions
        WHERE user_id = ?
        ORDER BY created_at DESC
    `, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    var sessions []model.Session
    for rows.Next() {
        var s model.Session
        var ua sql.NullString

        if err := rows.Scan(&s.ID, &s.UserID, &s.IP, &ua, &s.ExpiresAt, &s.CreatedAt); err != nil {
            continue
        }
        if ua.Valid {
            s.UserAgent = ua.String
        }
        sessions = append(sessions, s)
    }

    c.JSON(http.StatusOK, sessions)
}

func (h *SessionHandler) Revoke(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    userID, _ := c.Get("user_id")

    result, err := h.DB.Exec(
        "DELETE FROM sessions WHERE id = ? AND user_id = ?",
        id, userID,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    affected, _ := result.RowsAffected()
    if affected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Session revoked"})
}

func (h *SessionHandler) RevokeAll(c *gin.Context) {
    userID, _ := c.Get("user_id")

    _, err := h.DB.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "All sessions revoked"})
}

func (h *SessionHandler) CleanupExpired() error {
    _, err := h.DB.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
    return err
}
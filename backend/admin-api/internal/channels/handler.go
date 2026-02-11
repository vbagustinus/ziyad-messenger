package channels

import (
	"admin-service/internal/audit"
	"admin-service/internal/db"
	"net/http"
	"time"

	"admin-service/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Channel struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DepartmentID string `json:"department_id"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
	CreatedBy    string `json:"created_by"`
}

type CreateChannelRequest struct {
	Name         string `json:"name" binding:"required"`
	DepartmentID string `json:"department_id"`
}

func getClaims(c *gin.Context) *auth.Claims {
	val, _ := c.Get(auth.ClaimsKey)
	claims, _ := val.(*auth.Claims)
	return claims
}

func List(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, name, department_id, created_at, updated_at, created_by FROM channels ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []Channel{}
	for rows.Next() {
		var ch Channel
		var createdBy *string
		_ = rows.Scan(&ch.ID, &ch.Name, &ch.DepartmentID, &ch.CreatedAt, &ch.UpdatedAt, &createdBy)
		if createdBy != nil {
			ch.CreatedBy = *createdBy
		}
		list = append(list, ch)
	}
	c.JSON(http.StatusOK, gin.H{"channels": list})
}

func Create(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	id := uuid.New().String()
	now := time.Now().Unix()
	claims := getClaims(c)
	_, err := db.DB.Exec(
		`INSERT INTO channels (id, name, department_id, created_at, updated_at, created_by) VALUES (?, ?, ?, ?, ?, ?)`,
		id, req.Name, req.DepartmentID, now, now, claims.UserID,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "channel exists or db error"})
		return
	}
	_ = audit.Log(claims.UserID, claims.Username, "channel.create", "channels/"+id, req.Name, c.ClientIP())
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name})
}

func Delete(c *gin.Context) {
	id := c.Param("id")
	res, err := db.DB.Exec(`DELETE FROM channels WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	claims := getClaims(c)
	_ = audit.Log(claims.UserID, claims.Username, "channel.delete", "channels/"+id, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

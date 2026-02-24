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
	Type         string `json:"type"`
	DepartmentID string `json:"department_id"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
	CreatedBy    string `json:"created_by"`
}

type CreateChannelRequest struct {
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type"` // 'public', 'private'
	DepartmentID string `json:"department_id"`
}

func getClaims(c *gin.Context) *auth.Claims {
	val, _ := c.Get(auth.ClaimsKey)
	claims, _ := val.(*auth.Claims)
	return claims
}

func List(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, name, type, department_id, created_at, updated_at, created_by FROM channels ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []Channel{}
	for rows.Next() {
		var ch Channel
		var createdBy *string
		_ = rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.DepartmentID, &ch.CreatedAt, &ch.UpdatedAt, &createdBy)
		if createdBy != nil {
			ch.CreatedBy = *createdBy
		}
		list = append(list, ch)
	}
	c.JSON(http.StatusOK, gin.H{"channels": list})
}

func ListPublic(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, name, type, department_id, created_at, updated_at, created_by FROM channels WHERE type = 'public' ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []Channel{}
	for rows.Next() {
		var ch Channel
		var createdBy *string
		_ = rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.DepartmentID, &ch.CreatedAt, &ch.UpdatedAt, &createdBy)
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
	if req.Type == "" {
		req.Type = "public"
	}

	id := uuid.New().String()
	now := time.Now().Unix()
	claims := getClaims(c)

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO channels (id, name, type, department_id, created_at, updated_at, created_by) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, req.Name, req.Type, req.DepartmentID, now, now, claims.UserID,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "channel exists or db error"})
		return
	}

	// Auto-add creator as owner
	_, err = tx.Exec(
		`INSERT INTO channel_members (channel_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		id, claims.UserID, "owner", now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit error"})
		return
	}

	_ = audit.Log(claims.UserID, claims.Username, "channel.create", "channels/"+id, req.Name, c.ClientIP())
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "type": req.Type})
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

type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role"` // 'admin', 'member'
}

func ListMembers(c *gin.Context) {
	id := c.Param("id")
	rows, err := db.DB.Query(`
		SELECT u.id, u.username, u.full_name, m.role, m.joined_at 
		FROM channel_members m
		JOIN users u ON m.user_id = u.id
		WHERE m.channel_id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type MemberInfo struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
		JoinedAt int64  `json:"joined_at"`
	}

	members := []MemberInfo{}
	for rows.Next() {
		var m MemberInfo
		_ = rows.Scan(&m.ID, &m.Username, &m.FullName, &m.Role, &m.JoinedAt)
		members = append(members, m)
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

func AddMember(c *gin.Context) {
	id := c.Param("id")
	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}

	now := time.Now().Unix()
	_, err := db.DB.Exec(
		`INSERT INTO channel_members (channel_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		id, req.UserID, req.Role, now,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already in channel or database error"})
		return
	}

	claims := getClaims(c)
	_ = audit.Log(claims.UserID, claims.Username, "channel.member.add", "channels/"+id+"/members/"+req.UserID, req.Role, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func RemoveMember(c *gin.Context) {
	id := c.Param("id")
	user_id := c.Param("user_id")

	res, err := db.DB.Exec(`DELETE FROM channel_members WHERE channel_id = ? AND user_id = ?`, id, user_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}

	claims := getClaims(c)
	_ = audit.Log(claims.UserID, claims.Username, "channel.member.remove", "channels/"+id+"/members/"+user_id, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

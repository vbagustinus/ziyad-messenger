package roles

import (
	"admin-service/internal/audit"
	"admin-service/internal/db"
	"encoding/json"
	"net/http"
	"time"

	"admin-service/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	CreatedAt   int64    `json:"created_at"`
	UpdatedAt   int64    `json:"updated_at"`
}

type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
}

type UpdateRoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

func getClaims(c *gin.Context) *auth.Claims {
	val, _ := c.Get(auth.ClaimsKey)
	claims, _ := val.(*auth.Claims)
	return claims
}

func permEncode(p []string) string {
	if len(p) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(p)
	return string(b)
}

func permDecode(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func List(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, name, permissions, created_at, updated_at FROM roles ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []Role{}
	for rows.Next() {
		var r Role
		var perm string
		_ = rows.Scan(&r.ID, &r.Name, &perm, &r.CreatedAt, &r.UpdatedAt)
		r.Permissions = permDecode(perm)
		list = append(list, r)
	}
	c.JSON(http.StatusOK, gin.H{"roles": list})
}

func Create(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	id := uuid.New().String()
	now := time.Now().Unix()
	_, err := db.DB.Exec(
		`INSERT INTO roles (id, name, permissions, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, req.Name, permEncode(req.Permissions), now, now,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "role exists or db error"})
		return
	}
	claims := getClaims(c)
	_ = audit.Log(claims.UserID, claims.Username, "role.create", "roles/"+id, req.Name, c.ClientIP())
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name})
}

func Update(c *gin.Context) {
	id := c.Param("id")
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	now := time.Now().Unix()
	if req.Name != "" {
		_, _ = db.DB.Exec(`UPDATE roles SET name = ?, updated_at = ? WHERE id = ?`, req.Name, now, id)
	}
	if len(req.Permissions) > 0 {
		_, _ = db.DB.Exec(`UPDATE roles SET permissions = ?, updated_at = ? WHERE id = ?`, permEncode(req.Permissions), now, id)
	}
	claims := getClaims(c)
	_ = audit.Log(claims.UserID, claims.Username, "role.update", "roles/"+id, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func Delete(c *gin.Context) {
	id := c.Param("id")
	res, err := db.DB.Exec(`DELETE FROM roles WHERE id = ?`, id)
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
	_ = audit.Log(claims.UserID, claims.Username, "role.delete", "roles/"+id, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

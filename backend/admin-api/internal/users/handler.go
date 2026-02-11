package users

import (
	"admin-service/internal/audit"
	"admin-service/internal/auth"
	"admin-service/internal/db"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	FullName     string `json:"full_name"`
	RoleID       string `json:"role_id"`
	DepartmentID string `json:"department_id"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type CreateUserRequest struct {
	Username     string `json:"username" binding:"required"`
	FullName     string `json:"full_name"`
	Password     string `json:"password" binding:"required"`
	RoleID       string `json:"role_id"`
	DepartmentID string `json:"department_id"`
}

type UpdateUserRequest struct {
	Username     string `json:"username"`
	FullName     string `json:"full_name"`
	Password     string `json:"password"`
	RoleID       string `json:"role_id"`
	DepartmentID string `json:"department_id"`
}

func getClaims(c *gin.Context) *auth.Claims {
	val, _ := c.Get(auth.ClaimsKey)
	claims, _ := val.(*auth.Claims)
	return claims
}

func List(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, username, full_name, role_id, department_id, created_at, updated_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []User{}
	for rows.Next() {
		var u User
		_ = rows.Scan(&u.ID, &u.Username, &u.FullName, &u.RoleID, &u.DepartmentID, &u.CreatedAt, &u.UpdatedAt)
		list = append(list, u)
	}
	c.JSON(http.StatusOK, gin.H{"users": list})
}

func Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	id := uuid.New().String()
	now := time.Now().Unix()
	_, err = db.DB.Exec(
		`INSERT INTO users (id, username, full_name, password_hash, role_id, department_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.Username, req.FullName, hash, req.RoleID, req.DepartmentID, now, now,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user exists or db error"})
		return
	}
	claims := getClaims(c)
	_ = audit.Log(claims.UserID, claims.Username, "user.create", "users/"+id, req.Username, c.ClientIP())
	c.JSON(http.StatusCreated, gin.H{"id": id, "username": req.Username})
}

func Update(c *gin.Context) {
	id := c.Param("id")
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	now := time.Now().Unix()

	// Start building update query
	query := "UPDATE users SET updated_at = ?"
	args := []interface{}{now}

	if req.Username != "" {
		query += ", username = ?"
		args = append(args, req.Username)
	}
	if req.FullName != "" {
		query += ", full_name = ?"
		args = append(args, req.FullName)
	}
	if req.RoleID != "" {
		query += ", role_id = ?"
		args = append(args, req.RoleID)
	}
	if req.DepartmentID != "" {
		query += ", department_id = ?"
		args = append(args, req.DepartmentID)
	} else if c.Request.Method == "PUT" {
		// If it's a full update and department is missing, we might want to clear it
		// but for now let's only update if provided
	}

	query += " WHERE id = ?"
	args = append(args, id)

	_, err := db.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	claims := getClaims(c)
	_ = audit.Log(claims.UserID, claims.Username, "user.update", "users/"+id, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func ResetPassword(c *gin.Context) {
	id := c.Param("id")
	defaultPassword := "123456789"
	hash, err := auth.HashPassword(defaultPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}

	now := time.Now().Unix()
	_, err = db.DB.Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, hash, now, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	claims := getClaims(c)
	_ = audit.Log(claims.UserID, claims.Username, "user.reset_password", "users/"+id, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Password reset to default (123456789)"})
}

func Delete(c *gin.Context) {
	id := c.Param("id")
	res, err := db.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
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
	_ = audit.Log(claims.UserID, claims.Username, "user.delete", "users/"+id, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

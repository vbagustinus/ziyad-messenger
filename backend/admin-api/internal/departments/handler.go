package departments

import (
	"admin-service/internal/audit"
	"admin-service/internal/auth"
	"admin-service/internal/db"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Department struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func List(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, name, created_at, updated_at FROM departments ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []Department{}
	for rows.Next() {
		var d Department
		_ = rows.Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt)
		list = append(list, d)
	}
	c.JSON(http.StatusOK, gin.H{"departments": list})
}

func Create(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	id := uuid.New().String()
	now := time.Now().Unix()
	_, err := db.DB.Exec(
		`INSERT INTO departments (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		id, req.Name, now, now,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "department exists or db error"})
		return
	}

	val, _ := c.Get(auth.ClaimsKey)
	claims, _ := val.(*auth.Claims)
	_ = audit.Log(claims.UserID, claims.Username, "department.create", "departments/"+id, req.Name, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name})
}

func Delete(c *gin.Context) {
	id := c.Param("id")

	// Check if assigned to users or channels
	var count int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE department_id = ?`, id).Scan(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department is not empty (has users)"})
		return
	}

	res, err := db.DB.Exec(`DELETE FROM departments WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	val, _ := c.Get(auth.ClaimsKey)
	claims, _ := val.(*auth.Claims)
	_ = audit.Log(claims.UserID, claims.Username, "department.delete", "departments/"+id, "", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

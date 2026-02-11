package devices

import (
	"admin-service/internal/audit"
	"admin-service/internal/db"
	"net/http"

	"admin-service/internal/auth"

	"github.com/gin-gonic/gin"
)

type Device struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	DeviceName  string `json:"device_name"`
	Fingerprint string `json:"fingerprint"`
	LastSeen    int64  `json:"last_seen"`
	CreatedAt   int64  `json:"created_at"`
}

func getClaims(c *gin.Context) *auth.Claims {
	val, _ := c.Get(auth.ClaimsKey)
	claims, _ := val.(*auth.Claims)
	return claims
}

func List(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, user_id, device_name, fingerprint, last_seen, created_at FROM devices ORDER BY last_seen DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []Device{}
	for rows.Next() {
		var d Device
		var lastSeen *int64
		_ = rows.Scan(&d.ID, &d.UserID, &d.DeviceName, &d.Fingerprint, &lastSeen, &d.CreatedAt)
		if lastSeen != nil {
			d.LastSeen = *lastSeen
		}
		list = append(list, d)
	}
	c.JSON(http.StatusOK, gin.H{"devices": list})
}

func Delete(c *gin.Context) {
	id := c.Param("id")
	res, err := db.DB.Exec(`DELETE FROM devices WHERE id = ?`, id)
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
	_ = audit.Log(claims.UserID, claims.Username, "device.delete", "devices/"+id, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

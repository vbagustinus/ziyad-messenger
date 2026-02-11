package audit

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func ListHandler(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit > 200 {
		limit = 200
	}
	actorID := c.Query("actor_id")
	action := c.Query("action")

	list, err := GetList(offset, limit, actorID, action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": list})
}

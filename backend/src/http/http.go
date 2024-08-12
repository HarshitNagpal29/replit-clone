package http

import (
	"net/http"

	"github.com/HarshitNagpal29/replit-clone/backend/src/aws"
	"github.com/gin-gonic/gin"
)

// Initialize HTTP routes and middleware
func InitHttp(router *gin.Engine) {
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.POST("/project", func(c *gin.Context) {
		var reqBody struct {
			ReplId   string `json:"replId"`
			Language string `json:"language"`
		}

		if err := c.ShouldBindJSON(&reqBody); err != nil || reqBody.ReplId == "" {
			c.JSON(http.StatusBadRequest, "Bad request")
			return
		}

		err := aws.CopyS3Folder("base/"+reqBody.Language, "code/"+reqBody.ReplId, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "Failed to create project")
			return
		}

		c.String(http.StatusOK, "Project created")
	})
}

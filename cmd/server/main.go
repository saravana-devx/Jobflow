/**
* * Server entry point
 */

package main

// import "fmt"
import (
	"github.com/gin-gonic/gin"
	"net/http"
	"pulseDashboard/internal/config"
	"pulseDashboard/internal/database"
)

func init() {
	config.LoadEnvVariables()
	database.ConnectDB()
}

func main() {
	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	err := router.Run(":8080")
	if err != nil {
		panic(err)
	}
}

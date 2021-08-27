package rest

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"tg_sender/config"
	"tg_sender/controllers"
	"tg_sender/pkg/logger"
	"tg_sender/pkg/rabbit"
	"tg_sender/views"
)

// Server ...
type Server struct {
	config config.Config
	log    logger.Logger
	router *gin.Engine
	ctrl   controllers.ControllerMake
}

// New ...
func New(config config.Config, log logger.Logger, r *gin.Engine, conn *rabbit.Connection) *Server {
	s := &Server{
		config: config,
		log:    log,
		router: r,
		ctrl: controllers.New(log, config, conn),
	}
	s.endpoints()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) handleErrorResponse(c *gin.Context, errCodeHTTP, errCode int, errNote error) {
	c.JSON(errCodeHTTP, views.R{
		Status:    "Failure",
		ErrorCode: errCode,
		ErrorNote: errNote.Error(),
		Data:      nil,
	})
}

func (s *Server) handleSuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, views.R{
		Status:    "Success",
		ErrorCode: 0,
		ErrorNote: "",
		Data:      data,
	})
}

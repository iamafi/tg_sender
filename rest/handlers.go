package rest

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"tg_sender/controllers"
)

type Message struct {
	Num      int64 `json:"num"`
	Priority int64 `json:"priority"`
}

func (m Message) validate() error {
	var flag bool
	if m.Num <= 0 {
		return errors.New("num should be more than 0")
	}
	for _, v := range []int64{0, 5, 9} {
		if m.Priority == v {
			flag = true
		}
	}
	if !flag {
		return errors.New("priority should be 0 - for low priority, 5 - for medium, and 9 - for high")
	}
	return nil
}

func (s *Server) InitMessages(c *gin.Context) {
	var req Message
	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleErrorResponse(c, http.StatusUnprocessableEntity, -10, err)
		return
	}

	if err := req.validate(); err != nil {
		s.handleErrorResponse(c, http.StatusUnprocessableEntity, -10, err)
		return
	}

	err := s.ctrl.ProcessMessages(controllers.Message{
		Num: int(req.Num),
		Priority: int(req.Priority),
	})
	if err != nil {
		s.handleErrorResponse(c, http.StatusInternalServerError, -15, err)
		return
	}

	s.handleSuccessResponse(c, "OK")
}

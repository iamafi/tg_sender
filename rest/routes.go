package rest

func (s *Server) endpoints() {
	s.router.POST("/send", s.InitMessages)

	err := s.router.Run(s.config.HTTPPort)
	if err != nil {
		return
	}
}

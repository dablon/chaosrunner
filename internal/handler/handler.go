package handler

type Handler struct{}
func New()*Handler{return &Handler{}}
func(h*Handler)Process()string{return "ok"}
func(h*Handler)Health()map[string]string{return map[string]string{"status":"healthy"}}

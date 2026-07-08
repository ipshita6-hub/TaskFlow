package api

import (
"net/http"
"github.com/go-chi/chi/v5"
"github.com/go-chi/chi/v5/middleware"
)

type Handlers struct {
RegisterHandler    func(w http.ResponseWriter, r *http.Request)
LoginHandler       func(w http.ResponseWriter, r *http.Request)
CreateTaskHandler  func(w http.ResponseWriter, r *http.Request)
ListTasksHandler   func(w http.ResponseWriter, r *http.Request)
GetTaskHandler     func(w http.ResponseWriter, r *http.Request)
UpdateTaskHandler  func(w http.ResponseWriter, r *http.Request)
DeleteTaskHandler  func(w http.ResponseWriter, r *http.Request)
GetLogsHandler     func(w http.ResponseWriter, r *http.Request)
GetDLQHandler      func(w http.ResponseWriter, r *http.Request)
ListWorkersHandler func(w http.ResponseWriter, r *http.Request)
ValidateHandler    func(w http.ResponseWriter, r *http.Request)
}

func NewRouter(jwtMiddleware func(http.Handler) http.Handler, handlers *Handlers) *chi.Mux {
mux := chi.NewMux()
mux.Use(RequestID)
mux.Use(Logger)
mux.Use(Recoverer)
mux.Use(middleware.RealIP)
mux.Use(middleware.StripSlashes)
mux.Route("/api/v1", func(r chi.Router) {
r.Post("/auth/register", handlers.RegisterHandler)
r.Post("/auth/login", handlers.LoginHandler)
r.Get("/schedule/validate", handlers.ValidateHandler)
})
mux.Get("/api/docs", OpenAPIHandler)
mux.Route("/api/v1", func(r chi.Router) {
r.Use(jwtMiddleware)
r.Post("/tasks", handlers.CreateTaskHandler)
r.Get("/tasks", handlers.ListTasksHandler)
r.Get("/tasks/{id}", handlers.GetTaskHandler)
r.Patch("/tasks/{id}", handlers.UpdateTaskHandler)
r.Delete("/tasks/{id}", handlers.DeleteTaskHandler)
r.Get("/tasks/{id}/logs", handlers.GetLogsHandler)
r.Get("/jobs/dlq", handlers.GetDLQHandler)
r.Get("/workers", handlers.ListWorkersHandler)
})
return mux
}

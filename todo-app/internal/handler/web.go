package handler

import (
	"errors"
	"html/template"
	"net/http"

	"todo-app/internal/models"
	"todo-app/internal/repository"
	"todo-app/internal/service"
)

type WebHandler struct {
	svc  *service.TaskService
	tmpl *template.Template
}

func NewWebHandler(svc *service.TaskService, templatesGlob string) (*WebHandler, error) {
	tmpl, err := template.ParseGlob(templatesGlob)
	if err != nil {
		return nil, err
	}
	return &WebHandler{svc: svc, tmpl: tmpl}, nil
}

type listPageData struct {
	Tasks []models.Task
	Error string
}

func (h *WebHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.List(r.Context())
	data := listPageData{Tasks: tasks}
	if err != nil {
		data.Error = err.Error()
	}
	h.render(w, "list.html", data)
}

func (h *WebHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, "form.html", map[string]any{"Task": models.Task{}, "IsNew": true})
}

func (h *WebHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	title := r.FormValue("title")
	description := r.FormValue("description")
	if title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	if _, err := h.svc.Create(r.Context(), title, description); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *WebHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, err := idFromURL(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	task, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, "form.html", map[string]any{"Task": task, "IsNew": false})
}

func (h *WebHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := idFromURL(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	title := r.FormValue("title")
	description := r.FormValue("description")
	done := r.FormValue("done") == "on"

	if _, err := h.svc.Update(r.Context(), id, title, description, done); errors.Is(err, repository.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *WebHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := idFromURL(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil && !errors.Is(err, repository.ErrNotFound) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *WebHandler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

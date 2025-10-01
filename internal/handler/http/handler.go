package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/willjrcristo/go-sqlite-db/internal/domain"
	"github.com/willjrcristo/go-sqlite-db/internal/service"
)

// Para facilitar os testes, definimos uma interface que o nosso serviço deve satisfazer.
// O handler vai depender desta interface, não da implementação concreta do serviço.
type UsuarioService interface {
	CreateUser(ctx context.Context, usuario domain.Usuario) (int64, error)
	GetUserByID(ctx context.Context, id int64) (*domain.Usuario, error)
	GetAllUsers(ctx context.Context) ([]domain.Usuario, error)
	UpdateUser(ctx context.Context, id int64, usuario domain.Usuario) error
	DeleteUser(ctx context.Context, id int64) error
}

// UsuarioHandler lida com as requisições HTTP para a entidade Usuário.
type UsuarioHandler struct {
	service UsuarioService
}

// NewUsuarioHandler cria uma nova instância do UsuarioHandler.
func NewUsuarioHandler(s UsuarioService) *UsuarioHandler {
	return &UsuarioHandler{
		service: s,
	}
}

// Routes define e retorna todas as rotas que este handler gerencia.
// Manter as rotas junto com o handler deixa o código mais organizado.
func (h *UsuarioHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.CreateUser)      // POST /usuarios
	r.Get("/", h.GetAllUsers)      // GET /usuarios
	r.Get("/{id}", h.GetUserByID)  // GET /usuarios/{id}
	r.Put("/{id}", h.UpdateUser)   // PUT /usuarios/{id}
	r.Delete("/{id}", h.DeleteUser) // DELETE /usuarios/{id}

	return r
}

// --- MÉTODOS DO HANDLER ---

func (h *UsuarioHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var usuario domain.Usuario
	if err := json.NewDecoder(r.Body).Decode(&usuario); err != nil {
		respondWithError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	newID, err := h.service.CreateUser(r.Context(), usuario)
	if err != nil {
		if err == service.ErrDadosInvalidos {
			respondWithError(w, http.StatusBadRequest, err.Error())
		} else {
			respondWithError(w, http.StatusInternalServerError, "Erro ao criar usuário")
		}
		return
	}

	usuario.ID = newID
	respondWithJSON(w, http.StatusCreated, usuario)
}

func (h *UsuarioHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	usuarios, err := h.service.GetAllUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Erro ao buscar usuários")
		return
	}
	respondWithJSON(w, http.StatusOK, usuarios)
}

func (h *UsuarioHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "ID inválido")
		return
	}

	usuario, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		if err == service.ErrUsuarioNaoEncontrado {
			respondWithError(w, http.StatusNotFound, err.Error())
		} else {
			respondWithError(w, http.StatusInternalServerError, "Erro ao buscar usuário")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, usuario)
}

func (h *UsuarioHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "ID inválido")
		return
	}

	var usuario domain.Usuario
	if err := json.NewDecoder(r.Body).Decode(&usuario); err != nil {
		respondWithError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	err = h.service.UpdateUser(r.Context(), id, usuario)
	if err != nil {
		switch err {
		case service.ErrUsuarioNaoEncontrado:
			respondWithError(w, http.StatusNotFound, err.Error())
		case service.ErrDadosInvalidos:
			respondWithError(w, http.StatusBadRequest, err.Error())
		default:
			respondWithError(w, http.StatusInternalServerError, "Erro ao atualizar usuário")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UsuarioHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "ID inválido")
		return
	}

	err = h.service.DeleteUser(r.Context(), id)
	if err != nil {
		if err == service.ErrUsuarioNaoEncontrado {
			respondWithError(w, http.StatusNotFound, err.Error())
		} else {
			respondWithError(w, http.StatusInternalServerError, "Erro ao deletar usuário")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- FUNÇÕES AUXILIARES ---

func respondWithError(w http.ResponseWriter, code int, message string) {
	slog.Error("API Error", "code", code, "message", message)
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal JSON response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
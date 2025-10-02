package http

import (
	"context"
	"encoding/json"
	"io" // Importa o pacote io
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/willjrcristo/go-sqlite-db/internal/domain"
	"github.com/willjrcristo/go-sqlite-db/internal/service"
)

// Para facilitar os testes, definimos uma interface que o nosso serviço deve satisfazer.
// O handler vai depender desta interface, não da implementação concreta do serviço.
// A interface para o nosso serviço agora inclui os métodos da Stripe.
type UsuarioService interface {
	CreateUser(ctx context.Context, usuario domain.Usuario) (int64, error)
	GetUserByID(ctx context.Context, id int64) (*domain.Usuario, error)
	GetAllUsers(ctx context.Context) ([]domain.Usuario, error)
	UpdateUser(ctx context.Context, id int64, usuario domain.Usuario) error
	DeleteUser(ctx context.Context, id int64) error
	CreateCheckoutSession(ctx context.Context, userID int64) (string, error)
	HandleStripeWebhook(payload []byte, signature string) error
}

// UsuarioHandler lida com as requisições HTTP para a entidade Usuário gerenciando as rotas de /usuarios.
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
// Routes agora inclui o endpoint para criar o checkout.
func (h *UsuarioHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.CreateUser)      // POST /usuarios
	r.Get("/", h.GetAllUsers)      // GET /usuarios
	r.Get("/{id}", h.GetUserByID)  // GET /usuarios/{id}
	r.Put("/{id}", h.UpdateUser)   // PUT /usuarios/{id}
	r.Delete("/{id}", h.DeleteUser) // DELETE /usuarios/{id}
	// --- NOVA ROTA ---
	// POST /usuarios/{id}/criar-checkout
	r.Post("/{id}/criar-checkout", h.CreateCheckoutSession)

	return r
}

// --- NOVO HANDLER PARA CHECKOUT ---
// @Summary      Cria uma sessão de checkout na Stripe
// @Description  Gera uma URL de pagamento para um usuário iniciar uma assinatura
// @Tags         assinaturas
// @Produce      json
// @Param        id   path      int  true  "ID do Usuário"
// @Success      200  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /usuarios/{id}/criar-checkout [post]
func (h *UsuarioHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "ID de usuário inválido")
		return
	}

	checkoutURL, err := h.service.CreateCheckoutSession(r.Context(), id)
	if err != nil {
		switch err {
		case service.ErrUsuarioNaoEncontrado:
			respondWithError(w, http.StatusNotFound, err.Error())
		case service.ErrAssinaturaJaAtiva:
			respondWithError(w, http.StatusConflict, err.Error()) // 409 Conflict é um bom status para este caso
		default:
			respondWithError(w, http.StatusInternalServerError, "Erro ao criar sessão de checkout")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"checkout_url": checkoutURL})
}


// --- NOVO HANDLER PARA O WEBHOOK ---
// (Criamos uma struct separada para manter a lógica do webhook isolada)

type StripeWebhookHandler struct {
	service UsuarioService
}

func NewStripeWebhookHandler(s UsuarioService) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		service: s,
	}
}

// HandleStripeWebhook é o handler para a rota que recebe os eventos da Stripe.
func (h *StripeWebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = int64(65536) // Limite de 64KB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Erro ao ler o corpo do webhook", "error", err)
		respondWithError(w, http.StatusServiceUnavailable, "Erro ao ler corpo da requisição")
		return
	}

	signature := r.Header.Get("Stripe-Signature")

	err = h.service.HandleStripeWebhook(payload, signature)
	if err != nil {
		if err == service.ErrWebhookStripe {
			respondWithError(w, http.StatusBadRequest, "Falha na verificação da assinatura do webhook")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Erro interno ao processar webhook")
		}
		return
	}

	// Responda com 200 OK para a Stripe saber que recebemos o evento com sucesso.
	w.WriteHeader(http.StatusOK)
}

// --- MÉTODOS DO HANDLER ---

// @Summary      Cria um novo usuário
// @Description  Adiciona um novo usuário ao banco de dados com base nos dados fornecidos
// @Tags         usuarios
// @Accept       json
// @Produce      json
// @Param        usuario  body      domain.Usuario  true  "Dados do usuário para criação"
// @Success      201      {object}  domain.Usuario
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /usuarios [post]
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

// @Summary      Lista todos os usuários
// @Description  Retorna uma lista com todos os usuários cadastrados
// @Tags         usuarios
// @Produce      json
// @Success      200  {array}   domain.Usuario
// @Failure      500  {object}  map[string]string
// @Router       /usuarios [get]
func (h *UsuarioHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	usuarios, err := h.service.GetAllUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Erro ao buscar usuários")
		return
	}
	respondWithJSON(w, http.StatusOK, usuarios)
}

// @Summary      Busca um usuário por ID
// @Description  Retorna os dados de um usuário específico com base no seu ID
// @Tags         usuarios
// @Produce      json
// @Param        id   path      int  true  "ID do Usuário"
// @Success      200  {object}  domain.Usuario
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /usuarios/{id} [get]
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

// @Summary      Atualiza um usuário
// @Description  Atualiza os dados de um usuário existente com base no seu ID
// @Tags         usuarios
// @Accept       json
// @Produce      json
// @Param        id       path      int             true  "ID do Usuário"
// @Param        usuario  body      domain.Usuario  true  "Dados do usuário para atualização"
// @Success      204      {string}  string "No Content"
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /usuarios/{id} [put]
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

// @Summary      Deleta um usuário
// @Description  Remove um usuário do banco de dados com base no seu ID
// @Tags         usuarios
// @Produce      json
// @Param        id   path      int  true  "ID do Usuário"
// @Success      204  {string}  string "No Content"
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /usuarios/{id} [delete]
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
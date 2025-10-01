package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/willjrcristo/go-sqlite-db/internal/domain"
	"github.com/willjrcristo/go-sqlite-db/internal/service"
)

// --- Mock da Camada de Serviço ---

// MockUsuarioService é uma implementação falsa da nossa interface UsuarioService.
// Nós controlamos o que cada função vai retornar para simular diferentes cenários.
type MockUsuarioService struct {
	CreateUserFn  func(ctx context.Context, usuario domain.Usuario) (int64, error)
	GetUserByIDFn func(ctx context.Context, id int64) (*domain.Usuario, error)
}

// Implementamos os métodos da interface, mas eles apenas chamam as funções que definimos no mock.
func (m *MockUsuarioService) CreateUser(ctx context.Context, usuario domain.Usuario) (int64, error) {
	return m.CreateUserFn(ctx, usuario)
}

func (m *MockUsuarioService) GetUserByID(ctx context.Context, id int64) (*domain.Usuario, error) {
	return m.GetUserByIDFn(ctx, id)
}
// OBS: Para um teste completo, você implementaria todos os outros métodos da interface aqui também.
func (m *MockUsuarioService) GetAllUsers(ctx context.Context) ([]domain.Usuario, error) { return nil, nil }
func (m *MockUsuarioService) UpdateUser(ctx context.Context, id int64, usuario domain.Usuario) error { return nil }
func (m *MockUsuarioService) DeleteUser(ctx context.Context, id int64) error { return nil }


// --- Testes do Handler ---

func TestUsuarioHandler_GetUserByID(t *testing.T) {
	// t.Run nos permite criar sub-testes para diferentes cenários.
	t.Run("sucesso - deve retornar usuário e status 200", func(t *testing.T) {
		// Arrange (Organização)
		mockService := &MockUsuarioService{
			GetUserByIDFn: func(ctx context.Context, id int64) (*domain.Usuario, error) {
				// Simula o serviço encontrando o usuário com ID 1
				assert.Equal(t, int64(1), id)
				return &domain.Usuario{ID: 1, Nome: "Teste", Email: "teste@email.com"}, nil
			},
		}
		handler := NewUsuarioHandler(mockService)
		
		req := httptest.NewRequest("GET", "/usuarios/1", nil)
		rr := httptest.NewRecorder() // Captura a resposta

		// Precisamos de um roteador para que o chi possa extrair o {id} da URL
		router := chi.NewRouter()
		router.Get("/usuarios/{id}", handler.GetUserByID)

		// Act (Ação)
		router.ServeHTTP(rr, req)

		// Assert (Verificação)
		assert.Equal(t, http.StatusOK, rr.Code) // Verifica o status code

		var usuarioRetornado domain.Usuario
		err := json.Unmarshal(rr.Body.Bytes(), &usuarioRetornado)
		assert.NoError(t, err) // Verifica se o corpo da resposta é um JSON válido
		assert.Equal(t, int64(1), usuarioRetornado.ID)
		assert.Equal(t, "Teste", usuarioRetornado.Nome)
	})

	t.Run("erro - deve retornar não encontrado e status 404", func(t *testing.T) {
		// Arrange
		mockService := &MockUsuarioService{
			GetUserByIDFn: func(ctx context.Context, id int64) (*domain.Usuario, error) {
				// Simula o serviço não encontrando o usuário
				return nil, service.ErrUsuarioNaoEncontrado
			},
		}
		handler := NewUsuarioHandler(mockService)
		req := httptest.NewRequest("GET", "/usuarios/999", nil)
		rr := httptest.NewRecorder()
		router := chi.NewRouter()
		router.Get("/usuarios/{id}", handler.GetUserByID)

		// Act
		router.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestUsuarioHandler_CreateUser(t *testing.T) {
	t.Run("sucesso - deve criar usuário e retornar status 201", func(t *testing.T) {
		// Arrange
		usuarioParaCriar := domain.Usuario{Nome: "Novo User", Email: "novo@email.com"}
		mockService := &MockUsuarioService{
			CreateUserFn: func(ctx context.Context, usuario domain.Usuario) (int64, error) {
				// Simula o serviço criando o usuário e retornando o ID 5
				assert.Equal(t, usuarioParaCriar.Nome, usuario.Nome)
				return 5, nil
			},
		}
		handler := NewUsuarioHandler(mockService)

		// Converte a struct para JSON para enviar no corpo da requisição
		body, _ := json.Marshal(usuarioParaCriar)
		req := httptest.NewRequest("POST", "/usuarios", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler.CreateUser(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)
		var usuarioRetornado domain.Usuario
		json.Unmarshal(rr.Body.Bytes(), &usuarioRetornado)
		assert.Equal(t, int64(5), usuarioRetornado.ID) // Verifica se o ID retornado é o que o mock forneceu
		assert.Equal(t, usuarioParaCriar.Nome, usuarioRetornado.Nome)
	})
}
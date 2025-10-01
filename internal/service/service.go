package service

import (
	"context"
	"errors"
	"strings"

	"github.com/willjrcristo/go-sqlite-db/internal/domain"
	"github.com/willjrcristo/go-sqlite-db/internal/repository"
)

// Erros de negócio que podem ser retornados pela camada de serviço.
var (
	ErrUsuarioNaoEncontrado = errors.New("usuário não encontrado")
	ErrDadosInvalidos       = errors.New("dados do usuário inválidos")
)

// UsuarioService encapsula a lógica de negócio para usuários.
// Ele depende da interface UsuarioRepository para acessar os dados.
type UsuarioService struct {
	repo repository.UsuarioRepository
}

// NewUsuarioService cria uma nova instância do UsuarioService.
func NewUsuarioService(repo repository.UsuarioRepository) *UsuarioService {
	return &UsuarioService{
		repo: repo,
	}
}

// --- MÉTODOS DE LÓGICA DE NEGÓCIO ---

// CreateUser valida os dados de um novo usuário antes de criá-lo.
func (s *UsuarioService) CreateUser(ctx context.Context, usuario domain.Usuario) (int64, error) {
	// Exemplo de regra de negócio: campos não podem ser vazios.
	if usuario.Nome == "" || usuario.Email == "" {
		return 0, ErrDadosInvalidos
	}
	// Exemplo de regra de negócio: email deve conter '@'.
	if !strings.Contains(usuario.Email, "@") {
		return 0, ErrDadosInvalidos
	}

	// Se a validação passar, chama o repositório para criar o usuário.
	return s.repo.Create(ctx, usuario)
}

// GetUserByID busca um usuário pelo ID.
func (s *UsuarioService) GetUserByID(ctx context.Context, id int64) (*domain.Usuario, error) {
	usuario, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err // Repassa erros do repositório (ex: erro de conexão).
	}
	if usuario == nil {
		return nil, ErrUsuarioNaoEncontrado // Retorna um erro de negócio específico se não encontrar.
	}
	return usuario, nil
}

// GetAllUsers busca todos os usuários.
// Por enquanto, não temos lógica de negócio complexa aqui, então é um simples repasse.
func (s *UsuarioService) GetAllUsers(ctx context.Context) ([]domain.Usuario, error) {
	return s.repo.GetAll(ctx)
}

// UpdateUser valida os dados e verifica se o usuário existe antes de atualizar.
func (s *UsuarioService) UpdateUser(ctx context.Context, id int64, usuario domain.Usuario) error {
	// Validação dos dados de entrada.
	if usuario.Nome == "" || usuario.Email == "" {
		return ErrDadosInvalidos
	}

	// Regra de negócio: verificar se o usuário a ser atualizado realmente existe.
	_, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err // Retorna ErrUsuarioNaoEncontrado ou outro erro.
	}

	return s.repo.Update(ctx, id, usuario)
}

// DeleteUser deleta um usuário pelo ID.
func (s *UsuarioService) DeleteUser(ctx context.Context, id int64) error {
	// Regra de negócio: verificar se o usuário existe antes de deletar.
	_, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}
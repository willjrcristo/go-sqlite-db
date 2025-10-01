package repository

import (
	"context"
	"database/sql"

	"github.com/willjrcristo/go-sqlite-db/internal/domain" // Importa a nossa struct
)

// UsuarioRepository define a interface para as operações de persistência de usuários.
// Usar uma interface nos permite 'mockar' o repositório em testes e trocar a implementação do banco de dados facilmente.
type UsuarioRepository interface {
	Create(ctx context.Context, usuario domain.Usuario) (int64, error)
	GetAll(ctx context.Context) ([]domain.Usuario, error)
	GetByID(ctx context.Context, id int64) (*domain.Usuario, error)
	Update(ctx context.Context, id int64, usuario domain.Usuario) error
	Delete(ctx context.Context, id int64) error
}

// sqliteRepository é a implementação do UsuarioRepository para SQLite.
// Ela precisa de uma conexão com o banco de dados (*sql.DB) para funcionar.
type sqliteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository é uma "fábrica" que cria uma nova instância do nosso repositório.
// É assim que vamos injetar a dependência do banco de dados no nosso repositório.
func NewSQLiteRepository(db *sql.DB) UsuarioRepository {
	return &sqliteRepository{
		db: db,
	}
}

// --- MÉTODOS DA IMPLEMENTAÇÃO ---

func (r *sqliteRepository) Create(ctx context.Context, usuario domain.Usuario) (int64, error) {
	stmt, err := r.db.PrepareContext(ctx, "INSERT INTO usuarios(nome, email) VALUES(?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, usuario.Nome, usuario.Email)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (r *sqliteRepository) GetAll(ctx context.Context) ([]domain.Usuario, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, nome, email FROM usuarios")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usuarios []domain.Usuario
	for rows.Next() {
		var u domain.Usuario
		if err := rows.Scan(&u.ID, &u.Nome, &u.Email); err != nil {
			return nil, err
		}
		usuarios = append(usuarios, u)
	}

	return usuarios, nil
}

func (r *sqliteRepository) GetByID(ctx context.Context, id int64) (*domain.Usuario, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, nome, email FROM usuarios WHERE id = ?", id)

	var u domain.Usuario
	if err := row.Scan(&u.ID, &u.Nome, &u.Email); err != nil {
		// É uma boa prática tratar o erro 'sql.ErrNoRows' separadamente.
		if err == sql.ErrNoRows {
			return nil, nil // Retorna nil, nil se o usuário não for encontrado.
		}
		return nil, err
	}

	return &u, nil
}

func (r *sqliteRepository) Update(ctx context.Context, id int64, usuario domain.Usuario) error {
	stmt, err := r.db.PrepareContext(ctx, "UPDATE usuarios SET nome = ?, email = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, usuario.Nome, usuario.Email, id)
	return err
}

func (r *sqliteRepository) Delete(ctx context.Context, id int64) error {
	stmt, err := r.db.PrepareContext(ctx, "DELETE FROM usuarios WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, id)
	return err
}
package repository

import (
	"context"
	"database/sql"

	// Importa o pacote time
	"github.com/willjrcristo/go-sqlite-db/internal/domain" // Ajuste o nome do seu módulo se necessário
)

// UsuarioRepository define a interface para as operações de persistência de usuários.
type UsuarioRepository interface {
	Create(ctx context.Context, usuario domain.Usuario) (int64, error)
	GetAll(ctx context.Context) ([]domain.Usuario, error)
	GetByID(ctx context.Context, id int64) (*domain.Usuario, error)
	Update(ctx context.Context, id int64, usuario domain.Usuario) error
	Delete(ctx context.Context, id int64) error

	// Método específico para atualizar apenas os detalhes da assinatura.
	UpdateSubscriptionDetails(ctx context.Context, id int64, usuario domain.Usuario) error
	// Método para buscar um usuário pelo seu ID de cliente na Stripe.
	GetByStripeID(ctx context.Context, stripeID string) (*domain.Usuario, error)
}

// sqliteRepository é a implementação do UsuarioRepository para SQLite.
type sqliteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository é a fábrica que cria uma nova instância do nosso repositório.
func NewSQLiteRepository(db *sql.DB) UsuarioRepository {
	return &sqliteRepository{
		db: db,
	}
}

// Create não precisa de alterações. Os campos de assinatura terão seus valores padrão do DB.
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
	// Query atualizada para incluir os novos campos.
	query := `
		SELECT id, nome, email,
		       stripe_customer_id, stripe_subscription_id, subscription_status, subscription_current_period_end
		FROM usuarios`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usuarios []domain.Usuario
	for rows.Next() {
		var u domain.Usuario
		// Usamos tipos Null* para lidar com possíveis valores NULL do banco.
		var stripeCustomerID, stripeSubscriptionID, subscriptionStatus sql.NullString
		var subscriptionCurrentPeriodEnd sql.NullTime

		if err := rows.Scan(
			&u.ID, &u.Nome, &u.Email,
			&stripeCustomerID, &stripeSubscriptionID, &subscriptionStatus, &subscriptionCurrentPeriodEnd,
		); err != nil {
			return nil, err
		}

		// Atribuímos os valores para a struct, tratando os casos nulos.
		u.StripeCustomerID = stripeCustomerID.String
		u.StripeSubscriptionID = stripeSubscriptionID.String
		u.SubscriptionStatus = subscriptionStatus.String
		u.SubscriptionCurrentPeriodEnd = subscriptionCurrentPeriodEnd.Time

		usuarios = append(usuarios, u)
	}
	return usuarios, nil
}

func (r *sqliteRepository) GetByID(ctx context.Context, id int64) (*domain.Usuario, error) {
	// Query atualizada para incluir os novos campos.
	query := `
		SELECT id, nome, email,
		       stripe_customer_id, stripe_subscription_id, subscription_status, subscription_current_period_end
		FROM usuarios WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)

	var u domain.Usuario
	var stripeCustomerID, stripeSubscriptionID, subscriptionStatus sql.NullString
	var subscriptionCurrentPeriodEnd sql.NullTime

	if err := row.Scan(
		&u.ID, &u.Nome, &u.Email,
		&stripeCustomerID, &stripeSubscriptionID, &subscriptionStatus, &subscriptionCurrentPeriodEnd,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	u.StripeCustomerID = stripeCustomerID.String
	u.StripeSubscriptionID = stripeSubscriptionID.String
	u.SubscriptionStatus = subscriptionStatus.String
	u.SubscriptionCurrentPeriodEnd = subscriptionCurrentPeriodEnd.Time

	return &u, nil
}

// Update (para nome e e-mail) continua o mesmo.
func (r *sqliteRepository) Update(ctx context.Context, id int64, usuario domain.Usuario) error {
	stmt, err := r.db.PrepareContext(ctx, "UPDATE usuarios SET nome = ?, email = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, usuario.Nome, usuario.Email, id)
	return err
}

// Delete continua o mesmo.
func (r *sqliteRepository) Delete(ctx context.Context, id int64) error {
	stmt, err := r.db.PrepareContext(ctx, "DELETE FROM usuarios WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, id)
	return err
}

// UpdateSubscriptionDetails atualiza apenas os campos relacionados à assinatura Stripe.
func (r *sqliteRepository) UpdateSubscriptionDetails(ctx context.Context, id int64, usuario domain.Usuario) error {
	query := `
		UPDATE usuarios
		SET stripe_customer_id = ?, stripe_subscription_id = ?,
		    subscription_status = ?, subscription_current_period_end = ?
		WHERE id = ?`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		usuario.StripeCustomerID,
		usuario.StripeSubscriptionID,
		usuario.SubscriptionStatus,
		usuario.SubscriptionCurrentPeriodEnd,
		id,
	)
	return err
}

// GetByStripeID busca um usuário pelo seu Stripe Customer ID.
func (r *sqliteRepository) GetByStripeID(ctx context.Context, stripeID string) (*domain.Usuario, error) {
	query := `
		SELECT id, nome, email,
		       stripe_customer_id, stripe_subscription_id, subscription_status, subscription_current_period_end
		FROM usuarios WHERE stripe_customer_id = ?`

	row := r.db.QueryRowContext(ctx, query, stripeID)

	var u domain.Usuario
	var stripeCustomerID, stripeSubscriptionID, subscriptionStatus sql.NullString
	var subscriptionCurrentPeriodEnd sql.NullTime

	if err := row.Scan(
		&u.ID, &u.Nome, &u.Email,
		&stripeCustomerID, &stripeSubscriptionID, &subscriptionStatus, &subscriptionCurrentPeriodEnd,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Retorna nil, nil se não for encontrado, o que é um estado válido.
		}
		return nil, err
	}

	u.StripeCustomerID = stripeCustomerID.String
	u.StripeSubscriptionID = stripeSubscriptionID.String
	u.SubscriptionStatus = subscriptionStatus.String
	u.SubscriptionCurrentPeriodEnd = subscriptionCurrentPeriodEnd.Time

	return &u, nil
}


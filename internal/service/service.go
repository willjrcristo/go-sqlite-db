package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/willjrcristo/go-sqlite-db/internal/domain"
	"github.com/willjrcristo/go-sqlite-db/internal/repository"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/customer"
	"github.com/stripe/stripe-go/v78/subscription"
	"github.com/stripe/stripe-go/v78/webhook"
)

// Erros de negócio relacionados à assinatura.
var (
	ErrUsuarioNaoEncontrado = errors.New("usuário não encontrado")
	ErrDadosInvalidos       = errors.New("dados do usuário inválidos")
	ErrAssinaturaJaAtiva    = errors.New("usuário já possui uma assinatura ativa")
	ErrWebhookStripe        = errors.New("erro ao processar webhook da stripe")
)

// UsuarioService encapsula a lógica de negócio para usuários e assinaturas.
type UsuarioService struct {
	repo repository.UsuarioRepository
}

// NewUsuarioService cria uma nova instância do UsuarioService.
func NewUsuarioService(repo repository.UsuarioRepository) *UsuarioService {
	return &UsuarioService{
		repo: repo,
	}
}

// --- MÉTODOS CRUD EXISTENTES ---
// (CreateUser, GetUserByID, etc. continuam aqui, sem alterações)
func (s *UsuarioService) CreateUser(ctx context.Context, usuario domain.Usuario) (int64, error) {
	if usuario.Nome == "" || usuario.Email == "" {
		return 0, ErrDadosInvalidos
	}
	if !strings.Contains(usuario.Email, "@") {
		return 0, ErrDadosInvalidos
	}
	return s.repo.Create(ctx, usuario)
}

func (s *UsuarioService) GetUserByID(ctx context.Context, id int64) (*domain.Usuario, error) {
	usuario, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		return nil, ErrUsuarioNaoEncontrado
	}
	return usuario, nil
}

func (s *UsuarioService) GetAllUsers(ctx context.Context) ([]domain.Usuario, error) {
	return s.repo.GetAll(ctx)
}

func (s *UsuarioService) UpdateUser(ctx context.Context, id int64, usuario domain.Usuario) error {
	if usuario.Nome == "" || usuario.Email == "" {
		return ErrDadosInvalidos
	}
	_, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Update(ctx, id, usuario)
}

func (s *UsuarioService) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// --- NOVOS MÉTODOS PARA STRIPE ---

// CreateCheckoutSession cria uma sessão de pagamento na Stripe.
func (s *UsuarioService) CreateCheckoutSession(ctx context.Context, userID int64) (string, error) {
	// 1. Buscar o usuário no nosso banco
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrUsuarioNaoEncontrado
	}

	// 2. Regra de negócio: não permitir criar uma nova sessão se a assinatura já estiver ativa.
	if user.SubscriptionStatus == "active" {
		return "", ErrAssinaturaJaAtiva
	}

	stripeCustomerID := user.StripeCustomerID
	// 3. Se o usuário ainda não for um cliente na Stripe, crie um.
	if stripeCustomerID == "" {
		params := &stripe.CustomerParams{
			Name:  stripe.String(user.Nome),
			Email: stripe.String(user.Email),
		}
		c, err := customer.New(params)
		if err != nil {
			slog.Error("Falha ao criar cliente na Stripe", "error", err)
			return "", err
		}
		stripeCustomerID = c.ID
		// Salva o novo ID do cliente no nosso banco
		user.StripeCustomerID = stripeCustomerID
		if err := s.repo.UpdateSubscriptionDetails(ctx, user.ID, *user); err != nil {
			return "", err
		}
	}

	// 4. Criar a Sessão de Checkout
	// IMPORTANTE: Substitua os valores de Price ID e URLs pelos seus.
	params := &stripe.CheckoutSessionParams{
		Customer:   stripe.String(stripeCustomerID),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String("http://localhost:3000/sucesso?session_id={CHECKOUT_SESSION_ID}"), // URL do seu frontend
		CancelURL:  stripe.String("http://localhost:3000/cancelou"),                                // URL do seu frontend
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String("price_SEU_PRICE_ID_AQUI"), // Crie um produto e preço no Dashboard da Stripe
				Quantity: stripe.Int64(1),
			},
		},
	}

	sess, err := session.New(params)
	if err != nil {
		slog.Error("Falha ao criar a sessão de checkout na Stripe", "error", err)
		return "", err
	}

	return sess.URL, nil
}

// HandleStripeWebhook processa os eventos recebidos da Stripe.
func (s *UsuarioService) HandleStripeWebhook(payload []byte, signature string) error {
	// IMPORTANTE: Obtenha este segredo do Dashboard da Stripe (seção Webhooks)
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	// 1. Verificar a assinatura do evento
	event, err := webhook.ConstructEvent(payload, signature, webhookSecret)
	if err != nil {
		slog.Error("Erro ao verificar a assinatura do webhook", "error", err)
		return ErrWebhookStripe
	}

	// 2. Processar o evento com base no seu tipo
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return err
		}

		// Obtenha a assinatura completa para ter a data de expiração
		sub, err := subscription.Get(session.Subscription.ID, nil)
		if err != nil {
			return err
		}

		// Encontre nosso usuário pelo ID do cliente Stripe
		user, err := s.repo.GetByStripeID(context.Background(), session.Customer.ID)
		if err != nil || user == nil {
			return err
		}

		// Atualize os dados da assinatura do usuário
		user.StripeSubscriptionID = sub.ID
		user.SubscriptionStatus = string(sub.Status)
		user.SubscriptionCurrentPeriodEnd = time.Unix(sub.CurrentPeriodEnd, 0)

		return s.repo.UpdateSubscriptionDetails(context.Background(), user.ID, *user)

	case "customer.subscription.updated", "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return err
		}
		user, err := s.repo.GetByStripeID(context.Background(), sub.Customer.ID)
		if err != nil || user == nil {
			return err
		}
		user.SubscriptionStatus = string(sub.Status)
		user.SubscriptionCurrentPeriodEnd = time.Unix(sub.CurrentPeriodEnd, 0)
		return s.repo.UpdateSubscriptionDetails(context.Background(), user.ID, *user)

	default:
		slog.Info("Webhook da Stripe recebido, mas não tratado", "event_type", event.Type)
	}

	return nil
}

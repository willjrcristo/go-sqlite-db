package domain

import "time" // Precisaremos do pacote time

type Usuario struct {
	ID    int64  `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`

	// --- NOVOS CAMPOS PARA ASSINATURA ---

	// ID do cliente no Stripe (ex: "cus_...")
	// Essencial para ligar nosso usuário ao cliente na Stripe.
	StripeCustomerID string `json:"-"` // O "-" significa que este campo não será exposto na nossa API JSON.

	// ID da assinatura na Stripe (ex: "sub_...")
	// Para identificar a assinatura específica do usuário.
	StripeSubscriptionID string `json:"-"`

	// Status da assinatura (ex: "active", "canceled", "past_due").
	// Este campo será nossa "fonte da verdade" interna.
	SubscriptionStatus string `json:"subscription_status"`

	// Data de expiração do período atual da assinatura.
	// É a "vigência" que você mencionou.
	SubscriptionCurrentPeriodEnd time.Time `json:"subscription_current_period_end"`
}
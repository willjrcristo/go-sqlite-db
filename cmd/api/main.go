package main

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file" // O driver para ler migrations do disco

	_ "github.com/mattn/go-sqlite3"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/willjrcristo/go-sqlite-db/docs" // Importa a pasta docs gerada

	// Nossos pacotes internos da aplicação!
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
	httphandler "github.com/willjrcristo/go-sqlite-db/internal/handler/http"
	"github.com/willjrcristo/go-sqlite-db/internal/repository"
	"github.com/willjrcristo/go-sqlite-db/internal/service"
)

// @title           API de Usuários
// @version         1.0
// @description     Esta é uma API completa para gerenciamento de usuários, como parte de um projeto de portfólio.
// @termsOfService  http://swagger.io/terms/
//
// @contact.name   Will Cristo
// @contact.url    https://linkedin.com/in/willjrcristo
// @contact.email  willjrcristo@gmail.com
//
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
//
// @host      localhost:8080
// @BasePath  /
func main() {
	// --- CONFIGURAÇÃO DO LOGGER ---
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("🚀 Iniciando a API de Usuários...")

	// --- CONEXÃO COM O BANCO DE DADOS ---
	db, err := initDB("./sqlite-database.db")
	if err != nil {
		slog.Error("Erro ao inicializar o banco de dados", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("💾 Conexão com o banco de dados estabelecida com sucesso.")

	slog.Info("⏳ Executando migrations do banco de dados...")
	if err := runMigrations(db); err != nil {
		slog.Error("Erro ao executar as migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ Migrations executadas com sucesso.")

	// --- INJEÇÃO DE DEPENDÊNCIAS (WIRING) ---
	// Criamos as instâncias de cada camada, passando a dependência para a camada seguinte.
	// DB -> Repository -> Service -> Handler

	// Camada de Repositório
	usuarioRepo := repository.NewSQLiteRepository(db)
	slog.Info("Camada de repositório inicializada")

	// Camada de Serviço
	usuarioService := service.NewUsuarioService(usuarioRepo)
	slog.Info("Camada de serviço inicializada")

	// Camada de Handler
	usuarioHandler := httphandler.NewUsuarioHandler(usuarioService)
	slog.Info("Camada de handler inicializada")


	// --- CONFIGURAÇÃO DO ROTEADOR E ROTAS ---
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger) // Renomeado de slog.Logger para evitar conflito
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(prometheusMiddleware)

	// Rota de Health Check
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("API de Usuários está no ar! 🚀"))
	})

	// Rota para a documentação Swagger
    // A URL será http://localhost:8080/swagger/index.html
    r.Get("/swagger/*", httpSwagger.WrapHandler)
    slog.Info("📖 Documentação Swagger disponível em http://localhost:8080/swagger/index.html")

	// Rota para expor as métricas do Prometheus
	r.Handle("/metrics", promhttp.Handler())
	slog.Info("📊 Métricas Prometheus disponíveis em http://localhost:8080/metrics")

	// "Montamos" todas as rotas de usuário sob o prefixo /usuarios
	r.Mount("/usuarios", usuarioHandler.Routes())
	slog.Info("🛰️  Rotas de /usuarios registradas")


	// --- INICIALIZAÇÃO DO SERVIDOR HTTP ---
	slog.Info("✅ Servidor pronto para receber requisições na porta :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		slog.Error("Erro ao iniciar o servidor", "error", err)
		os.Exit(1)
	}
}

// runMigrations executa as migrations do banco de dados na inicialização.
func runMigrations(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return err
	}

	// Aponta para a pasta de migrations
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"sqlite3",
		driver,
	)
	if err != nil {
		return err
	}

	// Executa as migrations para cima (aplicando as novas)
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
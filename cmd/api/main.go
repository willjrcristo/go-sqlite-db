package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/mattn/go-sqlite3"

	// Nossos pacotes internos da aplicação!
	httphandler "github.com/willjrcristo/go-sqlite-db/internal/handler/http"
	"github.com/willjrcristo/go-sqlite-db/internal/repository"
	"github.com/willjrcristo/go-sqlite-db/internal/service"
)

func main() {
	// --- 1. CONFIGURAÇÃO DO LOGGER ---
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("🚀 Iniciando a API de Usuários...")

	// --- 2. CONEXÃO COM O BANCO DE DADOS ---
	db, err := initDB("./sqlite-database.db")
	if err != nil {
		slog.Error("Erro ao inicializar o banco de dados", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("💾 Conexão com o banco de dados estabelecida com sucesso.")

	// --- 3. INJEÇÃO DE DEPENDÊNCIAS (WIRING) ---
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


	// --- 4. CONFIGURAÇÃO DO ROTEADOR E ROTAS ---
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger) // Renomeado de slog.Logger para evitar conflito
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Rota de Health Check
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("API de Usuários está no ar! 🚀"))
	})

	// "Montamos" todas as rotas de usuário sob o prefixo /usuarios
	r.Mount("/usuarios", usuarioHandler.Routes())
	slog.Info("🛰️  Rotas de /usuarios registradas")


	// --- 5. INICIALIZAÇÃO DO SERVIDOR HTTP ---
	slog.Info("✅ Servidor pronto para receber requisições na porta :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		slog.Error("Erro ao iniciar o servidor", "error", err)
		os.Exit(1)
	}
}

// initDB (a função continua a mesma de antes)
func initDB(filepath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS usuarios (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		nome TEXT,
		email TEXT
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		return nil, err
	}
	return db, nil
}
package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/mattn/go-sqlite3" // Driver do SQLite
)

func main() {
	// --- 1. CONFIGURAÇÃO DO LOGGER ---
	// Usamos o slog para ter logs estruturados (em JSON), o que é ótimo para a observabilidade.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("🚀 Iniciando a API de Usuários...")

	// --- 2. CONEXÃO COM O BANCO DE DADOS ---
	db, err := initDB("./sqlite-database.db")
	if err != nil {
		slog.Error("Erro ao inicializar o banco de dados", "error", err)
		os.Exit(1) // Encerra a aplicação se não conseguir conectar ao DB.
	}
	defer db.Close()
	slog.Info("💾 Conexão com o banco de dados estabelecida com sucesso.")

	// --- 3. CONFIGURAÇÃO DO ROTEADOR (CHI) ---
	r := chi.NewRouter()

	// Middlewares são "filtros" que rodam em toda requisição.
	r.Use(middleware.RequestID)      // Adiciona um ID único para cada requisição.
	r.Use(middleware.RealIP)         // Adiciona o IP real do cliente.
	r.Use(middleware.Logger)         // Loga o início e o fim de cada requisição.
	r.Use(middleware.Recoverer)      // Se a aplicação entrar em pânico, ele se recupera e retorna um erro 500.
	r.Use(middleware.Timeout(60 * time.Second)) // Define um timeout para as requisições.

	// --- 4. DEFINIÇÃO DAS ROTAS (ainda vazias) ---
	// Por enquanto, uma rota raiz para sabermos que o servidor está no ar.
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Bem-vindo à API de Usuários!"))
	})

	// TODO: Aqui vamos registrar as rotas do CRUD de usuários (ex: r.Mount("/usuarios", ...))
	// quando tivermos o nosso handler.

	slog.Info("🛰️  Servidor escutando na porta :8080")

	// --- 5. INICIALIZAÇÃO DO SERVIDOR HTTP ---
	if err := http.ListenAndServe(":8080", r); err != nil {
		slog.Error("Erro ao iniciar o servidor", "error", err)
		os.Exit(1)
	}
}

// initDB inicializa a conexão com o banco de dados SQLite e cria a tabela de usuários se ela não existir.
func initDB(filepath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}

	// Verifica se a conexão com o banco de dados é bem-sucedida.
	if err = db.Ping(); err != nil {
		return nil, err
	}

	// Cria a tabela de usuários
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
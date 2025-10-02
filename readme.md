### Dockerização

Docker Swarm - Docker compose com aplicação e serviços Prometheus, Grafana, Kibana, DB

### Documentação

Swagger:
swag init -g cmd/api/main.go

### Migration

Instalar Scoop no Windows:
irm get.scoop.sh | iex
scoop bucket add extras

Instalar pacote migrate do scoop:
scoop install migrate

### Pacotes

go get github.com/gin-gonic/gin
go get -u github.com/go-chi/chi/v5
go get -u github.com/golang-migrate/migrate/v4
go get -u github.com/golang-migrate/migrate/v4/database/sqlite3
go get -u github.com/golang-migrate/migrate/v4/source/file
go get github.com/joho/godotenv
go get github.com/mattn/go-sqlite3
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
go get github.com/stretchr/testify
go get github.com/stripe/stripe-go/v78
go get github.com/swaggo/files
go get -u github.com/swaggo/http-swagger
go get github.com/swaggo/gin-swagger
go get -u github.com/swaggo/swag
go get gorm.io/gorm
go get gorm.io/driver/sqlite

### Teste de Carga

Benchmark

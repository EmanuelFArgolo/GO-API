package store

import (
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Driver do Postgres
	_ "github.com/golang-migrate/migrate/v4/source/file"       // Driver para ler de arquivos
)

// RunMigrations executa as migrações do banco de dados
func RunMigrations(connStr string) {
	// A pasta onde os arquivos SQL estão
	// (Note: 'file://' é necessário)
	migrationPath := "file://internal/store/migrations"

	log.Println("Iniciando migrações do banco de dados...")

	m, err := migrate.New(migrationPath, connStr)
	if err != nil {
		log.Fatalf("Falha ao inicializar migração: %v", err)
	}

	// Executa a migração (sobe a versão)
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("Migração: Nenhuma mudança detectada. Banco de dados já está atualizado.")
		} else {
			log.Fatalf("Falha ao aplicar migração 'up': %v", err)
		}
	} else {
		log.Println("Migrações do banco de dados aplicadas com sucesso!")
	}
}

// Package db initialise la connexion SQLite et applique les migrations
// embedded au démarrage. Aucune dépendance externe (CGO-free via modernc.org/sqlite).
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite" // driver SQLite pur Go — pas de CGO
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open ouvre (ou crée) la base SQLite au chemin dbPath,
// configure les paramètres de performance et de sécurité,
// puis applique les migrations manquantes.
func Open(dbPath string) (*sql.DB, error) {
	// Création du répertoire parent si nécessaire
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("db: création du répertoire %q : %w", filepath.Dir(dbPath), err)
	}

	// DSN SQLite avec paramètres de sécurité et performance :
	// - WAL journal  : lectures concurrentes sans bloquer les écritures
	// - foreign_keys : intégrité référentielle activée
	// - busy_timeout : attente max 5s avant SQLITE_BUSY
	// - cache=shared  : mémoire partagée entre connexions du même process
	dsn := fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000&cache=shared",
		dbPath,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: ouverture %q : %w", dbPath, err)
	}

	// SQLite supporte UNE seule connexion en écriture simultanée.
	// On limite le pool pour éviter les SQLITE_BUSY en écriture concurrente.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Ping de vérification
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db: ping %q : %w", dbPath, err)
	}

	// Application des migrations au démarrage
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("db: migrations : %w", err)
	}

	log.Info().Str("path", dbPath).Msg("db: base SQLite initialisée")
	return db, nil
}

// ── Migrations minimalistes embedded ──────────────────────────────────────
// Pas de dépendance goose/migrate : les migrations sont des fichiers SQL
// numérotés (001_init.sql, 002_xxx.sql…) lus depuis l'FS embedded.
// La table schema_migrations trace les fichiers déjà appliqués.

// runMigrations applique dans l'ordre les fichiers *.sql non encore traités.
func runMigrations(db *sql.DB) error {
	// Création de la table de suivi si absente
	const createTracker = `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		filename   TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT (datetime('now'))
	);`
	if _, err := db.Exec(createTracker); err != nil {
		return fmt.Errorf("création schema_migrations : %w", err)
	}

	// Récupération des migrations déjà appliquées
	applied, err := appliedMigrations(db)
	if err != nil {
		return err
	}

	// Lecture et tri des fichiers SQL embedded
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("lecture migrations FS : %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Application des migrations manquantes dans une transaction
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") || applied[name] {
			continue
		}

		content, err := fs.ReadFile(migrationsFS, "migrations/"+name)
		if err != nil {
			return fmt.Errorf("lecture %s : %w", name, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("transaction %s : %w", name, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback() //nolint:errcheck
			return fmt.Errorf("exécution %s : %w", name, err)
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (filename) VALUES (?)`, name,
		); err != nil {
			tx.Rollback() //nolint:errcheck
			return fmt.Errorf("enregistrement migration %s : %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s : %w", name, err)
		}

		log.Info().Str("migration", name).Msg("db: migration appliquée")
	}

	return nil
}

func appliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT filename FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

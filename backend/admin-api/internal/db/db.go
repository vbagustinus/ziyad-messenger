package db

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DB   *sql.DB
	once sync.Once
)

func Init(dbPath string) error {
	var err error
	once.Do(func() {
		DB, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			return
		}
		DB.SetMaxOpenConns(10)
		DB.SetMaxIdleConns(5)
		err = migrate()
	})
	return err
}

func migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS admin_users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_admin_users_username ON admin_users(username);

	CREATE TABLE IF NOT EXISTS roles (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		permissions TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS departments (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_departments_name ON departments(name);

	CREATE TABLE IF NOT EXISTS channels (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT DEFAULT 'public', -- 'public', 'private', 'dm'
		department_id TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		created_by TEXT,
		FOREIGN KEY (department_id) REFERENCES departments(id)
	);
	CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);

	CREATE TABLE IF NOT EXISTS channel_members (
		channel_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		role TEXT DEFAULT 'member', -- 'owner', 'admin', 'member'
		joined_at INTEGER NOT NULL,
		PRIMARY KEY (channel_id, user_id),
		FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_channel_members_user ON channel_members(user_id);

	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		full_name TEXT,
		password_hash TEXT NOT NULL,
		role_id TEXT,
		department_id TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		FOREIGN KEY (role_id) REFERENCES roles(id),
		FOREIGN KEY (department_id) REFERENCES departments(id)
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		actor_id TEXT NOT NULL,
		actor_username TEXT,
		action TEXT NOT NULL,
		target_resource TEXT NOT NULL,
		details TEXT,
		ip_address TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_id);
	CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
	`
	_, err := DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

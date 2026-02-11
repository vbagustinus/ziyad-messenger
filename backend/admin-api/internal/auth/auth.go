package auth

import (
	"admin-service/internal/db"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AdminUser struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	User      AdminUser `json:"user"`
	ExpiresAt int64     `json:"expires_at"`
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func Login(username, password string) (*AdminUser, error) {
	var u AdminUser
	err := db.DB.QueryRow(
		`SELECT id, username, password_hash, role, created_at, updated_at FROM admin_users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if !CheckPassword(u.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	return &u, nil
}

func CreateAdminUser(username, password, role string) (*AdminUser, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	id := uuid.New().String()
	now := time.Now().Unix()
	_, err = db.DB.Exec(
		`INSERT INTO admin_users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, username, hash, role, now, now,
	)
	if err != nil {
		return nil, err
	}
	return &AdminUser{
		ID: id, Username: username, Role: role, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func GetAdminByID(id string) (*AdminUser, error) {
	var u AdminUser
	err := db.DB.QueryRow(
		`SELECT id, username, password_hash, role, created_at, updated_at FROM admin_users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func EnsureSuperAdmin(username, password string) error {
	var count int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM admin_users`).Scan(&count)
	if count > 0 {
		return nil
	}
	_, err := CreateAdminUser(username, password, "super_admin")
	return err
}

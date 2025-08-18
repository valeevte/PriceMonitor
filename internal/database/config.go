package database

import (
	//"fmt"
	"net/url"
	"os"
)

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
}

func NewDBConfigFromEnv() DBConfig {
	return DBConfig{
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		DBName:   os.Getenv("DB_NAME"),
	}
}

// TargetDSN создаёт корректный DSN (URL encoded)
func (c DBConfig) TargetDSN() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   c.Host + ":" + c.Port,
		Path:   "/" + c.DBName,
	}
	// добавляем sslmode=disable для локальной разработки
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

// AdminDSN строит DSN для суперпользователя
func (c DBConfig) AdminDSN(superUser, superPass string) string {
	if superUser == "" {
		return ""
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(superUser, superPass),
		Host:   c.Host + ":" + c.Port,
		Path:   "/postgres",
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

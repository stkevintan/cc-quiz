package handlers

import (
	"database/sql"
	"net/http"

	authpkg "classical-chinese-quiz/backend/internal/auth"
	"classical-chinese-quiz/backend/internal/middleware"
	"classical-chinese-quiz/backend/internal/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB        *sql.DB
	JWTSecret string
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}

	var user models.User
	var passwordHash string
	err := h.DB.QueryRow(`SELECT id, username, password_hash, display_name, role, status, created_at, updated_at FROM users WHERE username = ?`, req.Username).
		Scan(&user.ID, &user.Username, &passwordHash, &user.DisplayName, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil || user.Status != "active" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	token, err := authpkg.IssueToken(h.JWTSecret, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (h AuthHandler) Me(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

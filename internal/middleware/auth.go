package middleware

import (
	"database/sql"
	"net/http"
	"strings"

	"classical-chinese-quiz/internal/auth"
	"classical-chinese-quiz/internal/models"
	"github.com/gin-gonic/gin"
)

const CurrentUserKey = "currentUser"

func Auth(secret string, database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := auth.ParseToken(secret, strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		user, err := loadActiveUser(database, claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found or disabled"})
			return
		}
		c.Set(CurrentUserKey, user)
		c.Next()
	}
}

func RequireRoles(roles ...models.Role) gin.HandlerFunc {
	allowed := make(map[models.Role]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok || !allowed[user.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (models.User, bool) {
	value, exists := c.Get(CurrentUserKey)
	if !exists {
		return models.User{}, false
	}
	user, ok := value.(models.User)
	return user, ok
}

func loadActiveUser(database *sql.DB, id int64) (models.User, error) {
	var user models.User
	err := database.QueryRow(`SELECT id, username, display_name, role, status, created_at, updated_at FROM users WHERE id = ? AND status = 'active'`, id).
		Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

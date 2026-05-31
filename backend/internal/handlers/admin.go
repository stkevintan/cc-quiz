package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"classical-chinese-quiz/backend/internal/middleware"
	"classical-chinese-quiz/backend/internal/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	DB *sql.DB
}

type createUserRequest struct {
	Username    string      `json:"username" binding:"required"`
	Password    string      `json:"password" binding:"required"`
	DisplayName string      `json:"displayName" binding:"required"`
	Role        models.Role `json:"role" binding:"required"`
	ClassIDs    []int64     `json:"classIds"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

type createClassRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

func (h AdminHandler) ListUsers(c *gin.Context) {
	role := models.Role(c.Query("role"))
	query := `SELECT id, username, display_name, role, status, created_at, updated_at FROM users WHERE role IN ('teacher', 'student')`
	args := []any{}
	if role == models.RoleTeacher || role == models.RoleStudent {
		query += ` AND role = ?`
		args = append(args, role)
	}
	query += ` ORDER BY id DESC`

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	defer rows.Close()

	rawUsers := []models.User{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read users"})
			return
		}
		rawUsers = append(rawUsers, user)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read users"})
		return
	}
	rows.Close()

	users := []gin.H{}
	for _, user := range rawUsers {
		users = append(users, gin.H{
			"id": user.ID, "username": user.Username, "displayName": user.DisplayName, "role": user.Role,
			"status": user.Status, "createdAt": user.CreatedAt, "updatedAt": user.UpdatedAt,
			"classes": h.classSummariesForUser(user),
		})
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h AdminHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user payload"})
		return
	}
	if req.Role != models.RoleTeacher && req.Role != models.RoleStudent {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin can only create teacher or student accounts"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.Username == "" || req.DisplayName == "" || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username/displayName are required and password must be at least 6 characters"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	admin, _ := middleware.CurrentUser(c)

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`INSERT INTO users (username, password_hash, display_name, role, created_by) VALUES (?, ?, ?, ?, ?)`,
		req.Username, string(hash), req.DisplayName, req.Role, admin.ID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists or payload is invalid"})
		return
	}
	userID, _ := result.LastInsertId()
	if err := bindUserClasses(tx, userID, req.Role, req.ClassIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": userID})
}

func (h AdminHandler) DisableUser(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	result, err := h.DB.Exec(`UPDATE users SET status = 'disabled', updated_at = CURRENT_TIMESTAMP WHERE id = ? AND role IN ('teacher', 'student')`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable user"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h AdminHandler) ResetPassword(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req resetPasswordRequest
	_ = c.ShouldBindJSON(&req)
	if strings.TrimSpace(req.Password) == "" {
		req.Password = "ChangeMe123"
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	result, err := h.DB.Exec(`UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND role IN ('teacher', 'student')`, string(hash), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "password": req.Password})
}

func (h AdminHandler) ListClasses(c *gin.Context) {
	rows, err := h.DB.Query(`SELECT id, name, description, status, created_at, updated_at FROM classes ORDER BY id DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list classes"})
		return
	}
	defer rows.Close()
	rawClasses := []models.Class{}
	for rows.Next() {
		class, err := scanClass(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read classes"})
			return
		}
		rawClasses = append(rawClasses, class)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read classes"})
		return
	}
	rows.Close()

	classes := []gin.H{}
	for _, class := range rawClasses {
		classes = append(classes, gin.H{
			"id": class.ID, "name": class.Name, "description": class.Description, "status": class.Status,
			"createdAt": class.CreatedAt, "updatedAt": class.UpdatedAt,
			"teachers": h.classMembers(class.ID, models.RoleTeacher),
			"students": h.classMembers(class.ID, models.RoleStudent),
		})
	}
	c.JSON(http.StatusOK, gin.H{"classes": classes})
}

func (h AdminHandler) CreateClass(c *gin.Context) {
	var req createClassRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class name is required"})
		return
	}
	admin, _ := middleware.CurrentUser(c)
	result, err := h.DB.Exec(`INSERT INTO classes (name, description, created_by) VALUES (?, ?, ?)`, strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), admin.ID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "class name already exists"})
		return
	}
	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h AdminHandler) UpdateClass(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req createClassRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class name is required"})
		return
	}
	result, err := h.DB.Exec(`UPDATE classes SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "class name already exists"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h AdminHandler) AddClassTeacher(c *gin.Context) {
	h.addClassMember(c, "teacherId", models.RoleTeacher)
}

func (h AdminHandler) RemoveClassTeacher(c *gin.Context) {
	h.removeClassMember(c, "teacherId", models.RoleTeacher)
}

func (h AdminHandler) AddClassStudent(c *gin.Context) {
	h.addClassMember(c, "studentId", models.RoleStudent)
}

func (h AdminHandler) RemoveClassStudent(c *gin.Context) {
	h.removeClassMember(c, "studentId", models.RoleStudent)
}

func (h AdminHandler) addClassMember(c *gin.Context, param string, role models.Role) {
	classID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	userID, ok := parseIDParam(c, param)
	if !ok {
		return
	}
	if !h.userHasRole(userID, role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user role does not match binding"})
		return
	}
	table, column := classBindingTable(role)
	_, err := h.DB.Exec(fmt.Sprintf(`INSERT OR IGNORE INTO %s (class_id, %s) VALUES (?, ?)`, table, column), classID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to bind class member"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h AdminHandler) removeClassMember(c *gin.Context, param string, role models.Role) {
	classID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	userID, ok := parseIDParam(c, param)
	if !ok {
		return
	}
	table, column := classBindingTable(role)
	_, err := h.DB.Exec(fmt.Sprintf(`DELETE FROM %s WHERE class_id = ? AND %s = ?`, table, column), classID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove class member"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func bindUserClasses(tx *sql.Tx, userID int64, role models.Role, classIDs []int64) error {
	table, column := classBindingTable(role)
	for _, classID := range classIDs {
		if classID <= 0 {
			return errors.New("classIds must be positive")
		}
		if _, err := tx.Exec(fmt.Sprintf(`INSERT OR IGNORE INTO %s (class_id, %s) VALUES (?, ?)`, table, column), classID, userID); err != nil {
			return errors.New("failed to bind user to class")
		}
	}
	return nil
}

func classBindingTable(role models.Role) (string, string) {
	if role == models.RoleTeacher {
		return "class_teachers", "teacher_id"
	}
	return "class_students", "student_id"
}

func (h AdminHandler) userHasRole(id int64, role models.Role) bool {
	var count int
	_ = h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ? AND role = ? AND status = 'active'`, id, role).Scan(&count)
	return count > 0
}

func (h AdminHandler) classSummariesForUser(user models.User) []gin.H {
	if user.Role != models.RoleTeacher && user.Role != models.RoleStudent {
		return []gin.H{}
	}
	table, column := classBindingTable(user.Role)
	rows, err := h.DB.Query(fmt.Sprintf(`SELECT c.id, c.name FROM classes c JOIN %s b ON b.class_id = c.id WHERE b.%s = ? ORDER BY c.name`, table, column), user.ID)
	if err != nil {
		return []gin.H{}
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var id int64
		var name string
		if rows.Scan(&id, &name) == nil {
			items = append(items, gin.H{"id": id, "name": name})
		}
	}
	return items
}

func (h AdminHandler) classMembers(classID int64, role models.Role) []models.User {
	table, column := classBindingTable(role)
	rows, err := h.DB.Query(fmt.Sprintf(`SELECT u.id, u.username, u.display_name, u.role, u.status, u.created_at, u.updated_at FROM users u JOIN %s b ON b.%s = u.id WHERE b.class_id = ? ORDER BY u.display_name`, table, column), classID)
	if err != nil {
		return []models.User{}
	}
	defer rows.Close()
	users := []models.User{}
	for rows.Next() {
		if user, err := scanUser(rows); err == nil {
			users = append(users, user)
		}
	}
	return users
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner rowScanner) (models.User, error) {
	var user models.User
	err := scanner.Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func scanClass(scanner rowScanner) (models.Class, error) {
	var class models.Class
	err := scanner.Scan(&class.ID, &class.Name, &class.Description, &class.Status, &class.CreatedAt, &class.UpdatedAt)
	return class, err
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

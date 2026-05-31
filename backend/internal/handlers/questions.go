package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"classical-chinese-quiz/backend/internal/middleware"
	"classical-chinese-quiz/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type QuestionHandler struct {
	DB *sql.DB
}

type questionRequest struct {
	Prompt        string   `json:"prompt"`
	Options       []string `json:"options"`
	CorrectOption int      `json:"correctOption"`
	Explanation   string   `json:"explanation"`
}

func (h QuestionHandler) List(c *gin.Context) {
	rows, err := h.DB.Query(`SELECT id, prompt, option_count, option_a, option_b, option_c, option_d, correct_option, explanation, status, created_by, created_at, updated_at FROM questions ORDER BY id DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list questions"})
		return
	}
	defer rows.Close()

	questions := []models.Question{}
	for rows.Next() {
		question, err := scanQuestion(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read questions"})
			return
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read questions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"questions": questions})
}

func (h QuestionHandler) Create(c *gin.Context) {
	var req questionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question payload"})
		return
	}
	question, err := validateQuestionRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, _ := middleware.CurrentUser(c)

	result, err := h.DB.Exec(`INSERT INTO questions (prompt, option_count, option_a, option_b, option_c, option_d, correct_option, explanation, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		question.Prompt, question.OptionCount, question.Options[0], question.Options[1], nullableOption(question.Options, 2), nullableOption(question.Options, 3), question.CorrectOption, question.Explanation, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create question"})
		return
	}
	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h QuestionHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	question, err := h.questionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"question": question})
}

func (h QuestionHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req questionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question payload"})
		return
	}
	question, err := validateQuestionRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.DB.Exec(`UPDATE questions SET prompt = ?, option_count = ?, option_a = ?, option_b = ?, option_c = ?, option_d = ?, correct_option = ?, explanation = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		question.Prompt, question.OptionCount, question.Options[0], question.Options[1], nullableOption(question.Options, 2), nullableOption(question.Options, 3), question.CorrectOption, question.Explanation, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update question"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h QuestionHandler) Disable(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	result, err := h.DB.Exec(`UPDATE questions SET status = 'disabled', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable question"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h QuestionHandler) questionByID(id int64) (models.Question, error) {
	return scanQuestion(h.DB.QueryRow(`SELECT id, prompt, option_count, option_a, option_b, option_c, option_d, correct_option, explanation, status, created_by, created_at, updated_at FROM questions WHERE id = ?`, id))
}

func validateQuestionRequest(req questionRequest) (models.Question, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return models.Question{}, errors.New("prompt is required")
	}
	if len(req.Options) < 2 || len(req.Options) > 4 {
		return models.Question{}, errors.New("questions must have 2 to 4 options")
	}
	options := []string{}
	for _, option := range req.Options {
		trimmed := strings.TrimSpace(option)
		if trimmed == "" {
			return models.Question{}, errors.New("options cannot be empty")
		}
		options = append(options, trimmed)
	}
	if req.CorrectOption < 0 || req.CorrectOption >= len(options) {
		return models.Question{}, errors.New("correctOption must reference an existing option")
	}
	return models.Question{
		Prompt:        prompt,
		Options:       options,
		OptionCount:   len(options),
		CorrectOption: req.CorrectOption,
		Explanation:   strings.TrimSpace(req.Explanation),
	}, nil
}

func nullableOption(options []string, index int) any {
	if len(options) <= index {
		return nil
	}
	return options[index]
}

func scanQuestion(scanner rowScanner) (models.Question, error) {
	var question models.Question
	var optionA, optionB string
	var optionC, optionD sql.NullString
	var createdBy sql.NullInt64
	if err := scanner.Scan(&question.ID, &question.Prompt, &question.OptionCount, &optionA, &optionB, &optionC, &optionD, &question.CorrectOption, &question.Explanation, &question.Status, &createdBy, &question.CreatedAt, &question.UpdatedAt); err != nil {
		return models.Question{}, err
	}
	question.CreatedBy = createdBy.Int64
	question.Options = []string{optionA, optionB}
	if question.OptionCount >= 3 && optionC.Valid {
		question.Options = append(question.Options, optionC.String)
	}
	if question.OptionCount >= 4 && optionD.Valid {
		question.Options = append(question.Options, optionD.String)
	}
	return question, nil
}

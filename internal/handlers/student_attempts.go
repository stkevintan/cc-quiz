package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"classical-chinese-quiz/internal/middleware"
	"classical-chinese-quiz/internal/models"
	"github.com/gin-gonic/gin"
)

const attemptQuestionCount = 10

type StudentAttemptHandler struct {
	DB *sql.DB
}

type answerRequest struct {
	AttemptQuestionID int64 `json:"attemptQuestionId" binding:"required"`
	SelectedOption    int   `json:"selectedOption"`
}

func (h StudentAttemptHandler) Create(c *gin.Context) {
	student, _ := middleware.CurrentUser(c)
	if attempt, ok, err := h.currentAttempt(student.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find current attempt"})
		return
	} else if ok {
		c.JSON(http.StatusOK, gin.H{"attempt": attempt})
		return
	}

	var activeCount int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM questions WHERE status = 'active'`).Scan(&activeCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count active questions"})
		return
	}
	if activeCount < attemptQuestionCount {
		c.JSON(http.StatusConflict, gin.H{"error": "at least 10 active questions are required"})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start attempt"})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`INSERT INTO attempts (student_id, question_count) VALUES (?, ?)`, student.ID, attemptQuestionCount)
	if err != nil {
		if attempt, ok, findErr := h.currentAttempt(student.ID); findErr == nil && ok {
			c.JSON(http.StatusOK, gin.H{"attempt": attempt})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create attempt"})
		return
	}
	attemptID, _ := result.LastInsertId()

	rows, err := tx.Query(`
SELECT id, prompt, option_count, option_a, option_b, option_c, option_d, correct_option, explanation
FROM questions
WHERE status = 'active'
ORDER BY RANDOM()
LIMIT ?`, attemptQuestionCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to select questions"})
		return
	}
	defer rows.Close()

	position := 0
	for rows.Next() {
		var questionID int64
		var prompt, optionA, optionB, explanation string
		var optionCount, correctOption int
		var optionC, optionD sql.NullString
		if err := rows.Scan(&questionID, &prompt, &optionCount, &optionA, &optionB, &optionC, &optionD, &correctOption, &explanation); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read selected question"})
			return
		}
		if _, err := tx.Exec(`
INSERT INTO attempt_questions (attempt_id, question_id, position, prompt, option_count, option_a, option_b, option_c, option_d, correct_option, explanation)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			attemptID, questionID, position, prompt, optionCount, optionA, optionB, nullStringValue(optionC), nullStringValue(optionD), correctOption, explanation); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save attempt question"})
			return
		}
		position++
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read selected questions"})
		return
	}
	if position != attemptQuestionCount {
		c.JSON(http.StatusConflict, gin.H{"error": "not enough active questions"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save attempt"})
		return
	}

	attempt, err := h.attemptDetail(attemptID, student.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load attempt"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"attempt": attempt})
}

func (h StudentAttemptHandler) Current(c *gin.Context) {
	student, _ := middleware.CurrentUser(c)
	attempt, ok, err := h.currentAttempt(student.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find current attempt"})
		return
	}
	if !ok {
		c.JSON(http.StatusOK, gin.H{"attempt": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"attempt": attempt})
}

func (h StudentAttemptHandler) Get(c *gin.Context) {
	student, _ := middleware.CurrentUser(c)
	attemptID, ok := parseIDParam(c, "attemptId")
	if !ok {
		return
	}
	attempt, err := h.attemptDetail(attemptID, student.ID)
	if err != nil {
		status := http.StatusNotFound
		if !errors.Is(err, sql.ErrNoRows) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": "attempt not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"attempt": attempt})
}

func (h StudentAttemptHandler) Answer(c *gin.Context) {
	student, _ := middleware.CurrentUser(c)
	attemptID, ok := parseIDParam(c, "attemptId")
	if !ok {
		return
	}
	var req answerRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.AttemptQuestionID <= 0 || req.SelectedOption < 0 || req.SelectedOption > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer payload"})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit answer"})
		return
	}
	defer tx.Rollback()

	var status string
	if err := tx.QueryRow(`SELECT status FROM attempts WHERE id = ? AND student_id = ?`, attemptID, student.ID).Scan(&status); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "attempt not found"})
		return
	}
	if status != "in_progress" {
		c.JSON(http.StatusConflict, gin.H{"error": "attempt is already completed"})
		return
	}

	var questionID int64
	var optionCount, correctOption int
	if err := tx.QueryRow(`SELECT question_id, option_count, correct_option FROM attempt_questions WHERE id = ? AND attempt_id = ?`, req.AttemptQuestionID, attemptID).Scan(&questionID, &optionCount, &correctOption); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "attempt question not found"})
		return
	}
	if req.SelectedOption >= optionCount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "selectedOption must reference an existing option"})
		return
	}

	isCorrect := req.SelectedOption == correctOption
	scoreDelta := 0
	if isCorrect {
		var hasEarnedScore int
		err := tx.QueryRow(`SELECT has_earned_score FROM student_question_progress WHERE student_id = ? AND question_id = ?`, student.ID, questionID).Scan(&hasEarnedScore)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load scoring progress"})
			return
		}
		if errors.Is(err, sql.ErrNoRows) || hasEarnedScore == 0 {
			scoreDelta = 1
		}
	}
	result, err := tx.Exec(`
INSERT OR IGNORE INTO attempt_answers (attempt_id, attempt_question_id, student_id, selected_option, is_correct, score_delta)
VALUES (?, ?, ?, ?, ?, ?)`, attemptID, req.AttemptQuestionID, student.ID, req.SelectedOption, boolToInt(isCorrect), scoreDelta)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save answer"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "question already answered"})
		return
	}

	if isCorrect {
		if _, err := tx.Exec(`
INSERT INTO student_question_progress (student_id, question_id, has_earned_score, first_correct_attempt_id, first_correct_at, last_answered_at, last_is_correct)
VALUES (?, ?, 1, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
ON CONFLICT(student_id, question_id) DO UPDATE SET
	has_earned_score = 1,
	first_correct_attempt_id = CASE WHEN student_question_progress.has_earned_score = 0 THEN excluded.first_correct_attempt_id ELSE student_question_progress.first_correct_attempt_id END,
	first_correct_at = CASE WHEN student_question_progress.has_earned_score = 0 THEN excluded.first_correct_at ELSE student_question_progress.first_correct_at END,
	last_answered_at = CURRENT_TIMESTAMP,
	last_is_correct = 1`, student.ID, questionID, attemptID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update scoring progress"})
			return
		}
	} else {
		if _, err := tx.Exec(`
INSERT INTO student_question_progress (student_id, question_id, has_earned_score, last_answered_at, last_is_correct)
VALUES (?, ?, 0, CURRENT_TIMESTAMP, 0)
ON CONFLICT(student_id, question_id) DO UPDATE SET
	last_answered_at = CURRENT_TIMESTAMP,
	last_is_correct = 0`, student.ID, questionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update scoring progress"})
			return
		}
		if _, err := tx.Exec(`
INSERT INTO wrong_answers (student_id, attempt_id, question_id, selected_option, correct_option, updated_at)
VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(student_id, question_id) DO UPDATE SET
	attempt_id = excluded.attempt_id,
	selected_option = excluded.selected_option,
	correct_option = excluded.correct_option,
	updated_at = CURRENT_TIMESTAMP`, student.ID, attemptID, questionID, req.SelectedOption, correctOption); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update wrong answers"})
			return
		}
	}

	if scoreDelta == 1 {
		if _, err := tx.Exec(`
INSERT INTO student_scores (student_id, total_score, updated_at)
VALUES (?, 1, CURRENT_TIMESTAMP)
ON CONFLICT(student_id) DO UPDATE SET
	total_score = student_scores.total_score + 1,
	updated_at = CURRENT_TIMESTAMP`, student.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update score"})
			return
		}
	}

	var answeredCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM attempt_answers WHERE attempt_id = ?`, attemptID).Scan(&answeredCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count answers"})
		return
	}
	if answeredCount >= attemptQuestionCount {
		if _, err := tx.Exec(`UPDATE attempts SET status = 'completed', updated_at = CURRENT_TIMESTAMP, completed_at = CURRENT_TIMESTAMP WHERE id = ?`, attemptID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete attempt"})
			return
		}
	} else {
		if _, err := tx.Exec(`UPDATE attempts SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, attemptID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update attempt"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save answer"})
		return
	}

	attempt, err := h.attemptDetail(attemptID, student.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load attempt"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"answer": gin.H{
			"attemptQuestionId": req.AttemptQuestionID,
			"selectedOption":    req.SelectedOption,
			"isCorrect":         isCorrect,
			"correctOption":     correctOption,
			"scoreDelta":        scoreDelta,
		},
		"attempt": attempt,
	})
}

func (h StudentAttemptHandler) Score(c *gin.Context) {
	student, _ := middleware.CurrentUser(c)
	score, err := h.studentScore(student.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load score"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"score": score})
}

func (h StudentAttemptHandler) WrongAnswers(c *gin.Context) {
	student, _ := middleware.CurrentUser(c)
	rows, err := h.DB.Query(`
SELECT wa.id, wa.student_id, wa.attempt_id, wa.question_id,
       aq.prompt, aq.option_count, aq.option_a, aq.option_b, aq.option_c, aq.option_d,
       wa.selected_option, wa.correct_option, aq.explanation, wa.updated_at
FROM wrong_answers wa
JOIN attempt_questions aq ON aq.attempt_id = wa.attempt_id AND aq.question_id = wa.question_id
WHERE wa.student_id = ?
ORDER BY wa.updated_at DESC, wa.id DESC`, student.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wrong answers"})
		return
	}
	defer rows.Close()

	wrongAnswers := []models.WrongAnswer{}
	for rows.Next() {
		wrongAnswer, err := scanWrongAnswer(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read wrong answers"})
			return
		}
		wrongAnswers = append(wrongAnswers, wrongAnswer)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read wrong answers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wrongAnswers": wrongAnswers})
}

func (h StudentAttemptHandler) currentAttempt(studentID int64) (models.Attempt, bool, error) {
	var attemptID int64
	err := h.DB.QueryRow(`SELECT id FROM attempts WHERE student_id = ? AND status = 'in_progress' ORDER BY id DESC LIMIT 1`, studentID).Scan(&attemptID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Attempt{}, false, nil
		}
		return models.Attempt{}, false, err
	}
	attempt, err := h.attemptDetail(attemptID, studentID)
	return attempt, err == nil, err
}

func (h StudentAttemptHandler) attemptDetail(attemptID, studentID int64) (models.Attempt, error) {
	var attempt models.Attempt
	var completedAt sql.NullString
	err := h.DB.QueryRow(`
SELECT id, student_id, status, question_count, created_at, updated_at, completed_at
FROM attempts
WHERE id = ? AND student_id = ?`, attemptID, studentID).
		Scan(&attempt.ID, &attempt.StudentID, &attempt.Status, &attempt.QuestionCount, &attempt.CreatedAt, &attempt.UpdatedAt, &completedAt)
	if err != nil {
		return models.Attempt{}, err
	}
	attempt.CompletedAt = completedAt.String

	rows, err := h.DB.Query(`
SELECT aq.id, aq.attempt_id, aq.question_id, aq.position, aq.prompt, aq.option_count, aq.option_a, aq.option_b, aq.option_c, aq.option_d,
       aq.correct_option, aq.explanation,
       aa.id, aa.selected_option, aa.is_correct, aa.score_delta, aa.answered_at
FROM attempt_questions aq
LEFT JOIN attempt_answers aa ON aa.attempt_question_id = aq.id
WHERE aq.attempt_id = ?
ORDER BY aq.position`, attemptID)
	if err != nil {
		return models.Attempt{}, err
	}
	defer rows.Close()

	attempt.Questions = []models.AttemptQuestion{}
	for rows.Next() {
		question, answered, err := scanAttemptQuestion(rows)
		if err != nil {
			return models.Attempt{}, err
		}
		if answered {
			attempt.AnsweredCount++
			if question.Answer != nil {
				attempt.ScoreDelta += question.Answer.ScoreDelta
			}
		}
		attempt.Questions = append(attempt.Questions, question)
	}
	return attempt, rows.Err()
}

func (h StudentAttemptHandler) studentScore(studentID int64) (models.StudentScore, error) {
	var score models.StudentScore
	var updatedAt sql.NullString
	err := h.DB.QueryRow(`SELECT student_id, total_score, updated_at FROM student_scores WHERE student_id = ?`, studentID).
		Scan(&score.StudentID, &score.TotalScore, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.StudentScore{StudentID: studentID, TotalScore: 0}, nil
		}
		return models.StudentScore{}, err
	}
	score.UpdatedAt = updatedAt.String
	return score, nil
}

func scanAttemptQuestion(scanner rowScanner) (models.AttemptQuestion, bool, error) {
	var question models.AttemptQuestion
	var optionA, optionB string
	var optionC, optionD sql.NullString
	var correctOption int
	var explanation string
	var answerID sql.NullInt64
	var selectedOption sql.NullInt64
	var isCorrect sql.NullInt64
	var scoreDelta sql.NullInt64
	var answeredAt sql.NullString
	if err := scanner.Scan(&question.ID, &question.AttemptID, &question.QuestionID, &question.Position, &question.Prompt, &question.OptionCount,
		&optionA, &optionB, &optionC, &optionD, &correctOption, &explanation,
		&answerID, &selectedOption, &isCorrect, &scoreDelta, &answeredAt); err != nil {
		return models.AttemptQuestion{}, false, err
	}
	question.Options = []string{optionA, optionB}
	if question.OptionCount >= 3 && optionC.Valid {
		question.Options = append(question.Options, optionC.String)
	}
	if question.OptionCount >= 4 && optionD.Valid {
		question.Options = append(question.Options, optionD.String)
	}
	if answerID.Valid {
		question.CorrectOption = &correctOption
		question.Explanation = explanation
		question.Answer = &models.AttemptAnswer{
			ID:                answerID.Int64,
			AttemptID:         question.AttemptID,
			AttemptQuestionID: question.ID,
			SelectedOption:    int(selectedOption.Int64),
			IsCorrect:         isCorrect.Int64 == 1,
			ScoreDelta:        int(scoreDelta.Int64),
			AnsweredAt:        answeredAt.String,
		}

		return question, true, nil
	}
	return question, false, nil
}

func scanWrongAnswer(scanner rowScanner) (models.WrongAnswer, error) {
	var wrongAnswer models.WrongAnswer
	var optionA, optionB string
	var optionC, optionD sql.NullString
	if err := scanner.Scan(&wrongAnswer.ID, &wrongAnswer.StudentID, &wrongAnswer.AttemptID, &wrongAnswer.QuestionID,
		&wrongAnswer.Prompt, &wrongAnswer.OptionCount, &optionA, &optionB, &optionC, &optionD,
		&wrongAnswer.SelectedOption, &wrongAnswer.CorrectOption, &wrongAnswer.Explanation, &wrongAnswer.UpdatedAt); err != nil {
		return models.WrongAnswer{}, err
	}
	wrongAnswer.Options = []string{optionA, optionB}
	if wrongAnswer.OptionCount >= 3 && optionC.Valid {
		wrongAnswer.Options = append(wrongAnswer.Options, optionC.String)
	}
	if wrongAnswer.OptionCount >= 4 && optionD.Valid {
		wrongAnswer.Options = append(wrongAnswer.Options, optionD.String)
	}
	return wrongAnswer, nil
}

func nullStringValue(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	return value.String
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

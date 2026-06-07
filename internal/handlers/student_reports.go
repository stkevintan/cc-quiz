package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"classical-chinese-quiz/internal/middleware"
	"classical-chinese-quiz/internal/models"
	"github.com/gin-gonic/gin"
)

func (h AdminHandler) ListStudentsReport(c *gin.Context) {
	reports, err := h.studentReports("")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list students"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"students": reports})
}

func (h AdminHandler) GetStudentReport(c *gin.Context) {
	studentID, ok := parseIDParam(c, "studentId")
	if !ok {
		return
	}
	report, err := h.studentReportByID(studentID)
	if err != nil {
		status := http.StatusNotFound
		if !errors.Is(err, sql.ErrNoRows) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": "student not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"student": report})
}

func (h AdminHandler) GetStudentWrongAnswers(c *gin.Context) {
	studentID, ok := parseIDParam(c, "studentId")
	if !ok {
		return
	}
	if !h.userHasRoleIncludingDisabled(studentID, models.RoleStudent) {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}
	wrongAnswers, err := h.studentWrongAnswers(studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wrong answers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wrongAnswers": wrongAnswers})
}

func (h AdminHandler) ResetStudentScore(c *gin.Context) {
	studentID, ok := parseIDParam(c, "studentId")
	if !ok {
		return
	}
	if !h.userHasRoleIncludingDisabled(studentID, models.RoleStudent) {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset score"})
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM student_question_progress WHERE student_id = ?`, studentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset scoring progress"})
		return
	}
	if _, err := tx.Exec(`
INSERT INTO student_scores (student_id, total_score, updated_at)
VALUES (?, 0, CURRENT_TIMESTAMP)
ON CONFLICT(student_id) DO UPDATE SET total_score = 0, updated_at = CURRENT_TIMESTAMP`, studentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset score"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset score"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h AdminHandler) ResetStudentAttempts(c *gin.Context) {
	studentID, ok := parseIDParam(c, "studentId")
	if !ok {
		return
	}
	if !h.userHasRoleIncludingDisabled(studentID, models.RoleStudent) {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset attempts"})
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM wrong_answers WHERE student_id = ?`, studentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset wrong answers"})
		return
	}
	if _, err := tx.Exec(`DELETE FROM student_question_progress WHERE student_id = ?`, studentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset scoring progress"})
		return
	}
	if _, err := tx.Exec(`DELETE FROM attempts WHERE student_id = ?`, studentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset attempts"})
		return
	}
	if _, err := tx.Exec(`
INSERT INTO student_scores (student_id, total_score, updated_at)
VALUES (?, 0, CURRENT_TIMESTAMP)
ON CONFLICT(student_id) DO UPDATE SET total_score = 0, updated_at = CURRENT_TIMESTAMP`, studentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset score"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset attempts"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h TeacherHandler) GetStudentWrongAnswers(c *gin.Context) {
	teacher, _ := middleware.CurrentUser(c)
	studentID, ok := parseIDParam(c, "studentId")
	if !ok {
		return
	}
	if !h.canSeeStudent(teacher.ID, studentID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "student is outside teacher class scope"})
		return
	}
	wrongAnswers, err := studentWrongAnswers(h.DB, studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wrong answers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wrongAnswers": wrongAnswers})
}

func (h AdminHandler) studentReports(scopeSQL string, args ...any) ([]models.StudentReport, error) {
	query := `
SELECT u.id, u.username, u.display_name, u.role, u.status,
       COALESCE(ss.total_score, 0), COALESCE(ss.updated_at, ''),
       COUNT(DISTINCT a.id),
       COUNT(DISTINCT CASE WHEN a.status = 'completed' THEN a.id END),
       COUNT(DISTINCT CASE WHEN a.status = 'in_progress' THEN a.id END),
       COUNT(DISTINCT wa.id)
FROM users u
LEFT JOIN student_scores ss ON ss.student_id = u.id
LEFT JOIN attempts a ON a.student_id = u.id
LEFT JOIN wrong_answers wa ON wa.student_id = u.id
WHERE u.role = 'student'` + scopeSQL + `
GROUP BY u.id, u.username, u.display_name, u.role, u.status, ss.total_score, ss.updated_at
ORDER BY u.display_name`
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reports := []models.StudentReport{}
	for rows.Next() {
		report, err := scanStudentReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range reports {
		reports[i].Classes = classSummariesForStudent(h.DB, reports[i].ID)
	}
	return reports, nil
}

func (h AdminHandler) studentReportByID(studentID int64) (models.StudentReport, error) {
	reports, err := h.studentReports(" AND u.id = ?", studentID)
	if err != nil {
		return models.StudentReport{}, err
	}
	if len(reports) == 0 {
		return models.StudentReport{}, sql.ErrNoRows
	}
	attempts, err := studentAttemptSummaries(h.DB, studentID)
	if err != nil {
		return models.StudentReport{}, err
	}
	reports[0].Attempts = attempts
	return reports[0], nil
}

func (h AdminHandler) studentWrongAnswers(studentID int64) ([]models.WrongAnswer, error) {
	return studentWrongAnswers(h.DB, studentID)
}

func (h AdminHandler) userHasRoleIncludingDisabled(id int64, role models.Role) bool {
	var count int
	_ = h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ? AND role = ?`, id, role).Scan(&count)
	return count > 0
}

func teacherStudentReports(database *sql.DB, teacherID int64) ([]models.StudentReport, error) {
	query := `
SELECT u.id, u.username, u.display_name, u.role, u.status,
       COALESCE(ss.total_score, 0), COALESCE(ss.updated_at, ''),
       COUNT(DISTINCT a.id),
       COUNT(DISTINCT CASE WHEN a.status = 'completed' THEN a.id END),
       COUNT(DISTINCT CASE WHEN a.status = 'in_progress' THEN a.id END),
       COUNT(DISTINCT wa.id)
FROM users u
JOIN class_students cs ON cs.student_id = u.id
JOIN class_teachers ct ON ct.class_id = cs.class_id
LEFT JOIN student_scores ss ON ss.student_id = u.id
LEFT JOIN attempts a ON a.student_id = u.id
LEFT JOIN wrong_answers wa ON wa.student_id = u.id
WHERE ct.teacher_id = ? AND u.role = 'student' AND u.status = 'active'
GROUP BY u.id, u.username, u.display_name, u.role, u.status, ss.total_score, ss.updated_at
ORDER BY u.display_name`
	rows, err := database.Query(query, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reports := []models.StudentReport{}
	for rows.Next() {
		report, err := scanStudentReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range reports {
		reports[i].Classes = visibleClassSummaries(database, teacherID, reports[i].ID)
	}
	return reports, nil
}

func classSummariesForStudent(database *sql.DB, studentID int64) []models.ClassSummary {
	rows, err := database.Query(`
SELECT c.id, c.name
FROM classes c
JOIN class_students cs ON cs.class_id = c.id
WHERE cs.student_id = ?
ORDER BY c.name`, studentID)
	if err != nil {
		return []models.ClassSummary{}
	}
	defer rows.Close()
	items := []models.ClassSummary{}
	for rows.Next() {
		var item models.ClassSummary
		if rows.Scan(&item.ID, &item.Name) == nil {
			items = append(items, item)
		}
	}
	return items
}

func visibleClassSummaries(database *sql.DB, teacherID, studentID int64) []models.ClassSummary {
	rows, err := database.Query(`
SELECT c.id, c.name
FROM classes c
JOIN class_teachers ct ON ct.class_id = c.id
JOIN class_students cs ON cs.class_id = c.id
WHERE ct.teacher_id = ? AND cs.student_id = ?
ORDER BY c.name`, teacherID, studentID)
	if err != nil {
		return []models.ClassSummary{}
	}
	defer rows.Close()
	items := []models.ClassSummary{}
	for rows.Next() {
		var item models.ClassSummary
		if rows.Scan(&item.ID, &item.Name) == nil {
			items = append(items, item)
		}
	}
	return items
}

func teacherStudentReport(database *sql.DB, teacherID, studentID int64) (models.StudentReport, error) {
	reports, err := teacherStudentReports(database, teacherID)
	if err != nil {
		return models.StudentReport{}, err
	}
	for _, report := range reports {
		if report.ID == studentID {
			attempts, err := studentAttemptSummaries(database, studentID)
			if err != nil {
				return models.StudentReport{}, err
			}
			report.Attempts = attempts
			return report, nil
		}
	}
	return models.StudentReport{}, sql.ErrNoRows
}

func scanStudentReport(scanner rowScanner) (models.StudentReport, error) {
	var report models.StudentReport
	var scoreUpdatedAt string
	var inProgressCount int
	err := scanner.Scan(&report.ID, &report.Username, &report.DisplayName, &report.Role, &report.Status,
		&report.Score.TotalScore, &scoreUpdatedAt, &report.AttemptCount, &report.CompletedCount, &inProgressCount, &report.WrongCount)
	report.Score.StudentID = report.ID
	report.Score.UpdatedAt = scoreUpdatedAt
	report.InProgress = inProgressCount > 0
	return report, err
}

func studentAttemptSummaries(database *sql.DB, studentID int64) ([]models.AttemptSummary, error) {
	rows, err := database.Query(`
SELECT a.id, a.student_id, a.status, a.question_count, a.created_at, a.updated_at, a.completed_at,
       COUNT(aa.id), COALESCE(SUM(aa.score_delta), 0)
FROM attempts a
LEFT JOIN attempt_answers aa ON aa.attempt_id = a.id
WHERE a.student_id = ?
GROUP BY a.id, a.student_id, a.status, a.question_count, a.created_at, a.updated_at, a.completed_at
ORDER BY a.id DESC`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	attempts := []models.AttemptSummary{}
	for rows.Next() {
		var attempt models.AttemptSummary
		var completedAt sql.NullString
		if err := rows.Scan(&attempt.ID, &attempt.StudentID, &attempt.Status, &attempt.QuestionCount,
			&attempt.CreatedAt, &attempt.UpdatedAt, &completedAt, &attempt.AnsweredCount, &attempt.ScoreDelta); err != nil {
			return nil, err
		}
		attempt.CompletedAt = completedAt.String
		attempts = append(attempts, attempt)
	}
	return attempts, rows.Err()
}

func studentWrongAnswers(database *sql.DB, studentID int64) ([]models.WrongAnswer, error) {
	rows, err := database.Query(`
SELECT wa.id, wa.student_id, wa.attempt_id, wa.question_id,
       aq.prompt, aq.option_count, aq.option_a, aq.option_b, aq.option_c, aq.option_d,
       wa.selected_option, wa.correct_option, aq.explanation, wa.updated_at
FROM wrong_answers wa
JOIN attempt_questions aq ON aq.attempt_id = wa.attempt_id AND aq.question_id = wa.question_id
WHERE wa.student_id = ?
ORDER BY wa.updated_at DESC, wa.id DESC`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	wrongAnswers := []models.WrongAnswer{}
	for rows.Next() {
		wrongAnswer, err := scanWrongAnswer(rows)
		if err != nil {
			return nil, err
		}
		wrongAnswers = append(wrongAnswers, wrongAnswer)
	}
	return wrongAnswers, rows.Err()
}

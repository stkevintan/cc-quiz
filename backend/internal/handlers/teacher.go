package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"classical-chinese-quiz/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

type TeacherHandler struct {
	DB *sql.DB
}

func (h TeacherHandler) ListStudents(c *gin.Context) {
	teacher, _ := middleware.CurrentUser(c)
	students, err := teacherStudentReports(h.DB, teacher.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list students"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"students": students})
}

func (h TeacherHandler) GetStudent(c *gin.Context) {
	teacher, _ := middleware.CurrentUser(c)
	studentID, ok := parseIDParam(c, "studentId")
	if !ok {
		return
	}
	if !h.canSeeStudent(teacher.ID, studentID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "student is outside teacher class scope"})
		return
	}
	report, err := teacherStudentReport(h.DB, teacher.ID, studentID)
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

func (h TeacherHandler) canSeeStudent(teacherID, studentID int64) bool {
	var count int
	_ = h.DB.QueryRow(`
SELECT COUNT(*)
FROM class_teachers ct
JOIN class_students cs ON cs.class_id = ct.class_id
WHERE ct.teacher_id = ? AND cs.student_id = ?`, teacherID, studentID).Scan(&count)
	return count > 0
}

func (h TeacherHandler) visibleClasses(teacherID, studentID int64) []gin.H {
	rows, err := h.DB.Query(`
SELECT c.id, c.name
FROM classes c
JOIN class_teachers ct ON ct.class_id = c.id
JOIN class_students cs ON cs.class_id = c.id
WHERE ct.teacher_id = ? AND cs.student_id = ?
ORDER BY c.name`, teacherID, studentID)
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

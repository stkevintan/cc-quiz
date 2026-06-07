package routes_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"classical-chinese-quiz/internal/config"
	appdb "classical-chinese-quiz/internal/db"
	"classical-chinese-quiz/internal/models"
	"classical-chinese-quiz/internal/routes"
	"github.com/gin-gonic/gin"
)

type testApp struct {
	t      *testing.T
	DB     *sql.DB
	Router http.Handler
}

type attemptEnvelope struct {
	Attempt models.Attempt `json:"attempt"`
}

type answerEnvelope struct {
	Answer struct {
		AttemptQuestionID int64 `json:"attemptQuestionId"`
		SelectedOption    int   `json:"selectedOption"`
		IsCorrect         bool  `json:"isCorrect"`
		CorrectOption     int   `json:"correctOption"`
		ScoreDelta        int   `json:"scoreDelta"`
	} `json:"answer"`
	Attempt models.Attempt `json:"attempt"`
}

type scoreEnvelope struct {
	Score models.StudentScore `json:"score"`
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	gin.SetMode(gin.TestMode)
	database, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if _, err := database.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		t.Fatalf("set busy timeout: %v", err)
	}
	if _, err := appdb.Migrate(database); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	seedExactlyTenActiveQuestions(t, database)

	cfg := config.Config{AppEnv: "test", JWTSecret: "test-secret", CORSAllowedOrigins: []string{"http://localhost:5173"}}
	return &testApp{t: t, DB: database, Router: routes.NewRouter(cfg, database)}
}

func seedExactlyTenActiveQuestions(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, err := database.Exec(`DELETE FROM questions`); err != nil {
		t.Fatalf("clear seed questions: %v", err)
	}
	for i := 1; i <= 10; i++ {
		if _, err := database.Exec(`
INSERT INTO questions (prompt, option_count, option_a, option_b, option_c, option_d, correct_option, explanation, status, created_by)
VALUES (?, 4, '正确', '错一', '错二', '错三', 0, ?, 'active', (SELECT id FROM users WHERE username = 'admin'))`,
			fmt.Sprintf("回归题 %02d", i), fmt.Sprintf("解析 %02d", i)); err != nil {
			t.Fatalf("insert controlled question %d: %v", i, err)
		}
	}
}

func (app *testApp) login(username, password string) string {
	var body struct {
		Token string `json:"token"`
	}
	app.doJSON(http.MethodPost, "/api/auth/login", "", gin.H{"username": username, "password": password}, http.StatusOK, &body)
	if body.Token == "" {
		app.t.Fatalf("login for %s returned empty token", username)
	}
	return body.Token
}

func (app *testApp) doJSON(method, path, token string, payload any, wantStatus int, out any) *httptest.ResponseRecorder {
	app.t.Helper()
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			app.t.Fatalf("encode request payload: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	app.Router.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		app.t.Fatalf("%s %s status = %d, want %d, body: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			app.t.Fatalf("decode response from %s %s: %v; body: %s", method, path, err, rec.Body.String())
		}
	}
	return rec
}

func startAttempt(t *testing.T, app *testApp, token string, wantStatus int) models.Attempt {
	t.Helper()
	var response attemptEnvelope
	app.doJSON(http.MethodPost, "/api/student/attempts", token, nil, wantStatus, &response)
	if response.Attempt.ID == 0 {
		t.Fatal("start attempt returned empty attempt")
	}
	return response.Attempt
}

func answerQuestion(t *testing.T, app *testApp, token string, attemptID, attemptQuestionID int64, selected int) answerEnvelope {
	t.Helper()
	var response answerEnvelope
	app.doJSON(http.MethodPost, fmt.Sprintf("/api/student/attempts/%d/answers", attemptID), token, gin.H{
		"attemptQuestionId": attemptQuestionID,
		"selectedOption":    selected,
	}, http.StatusOK, &response)
	return response
}

func questionByQuestionID(t *testing.T, attempt models.Attempt, questionID int64) models.AttemptQuestion {
	t.Helper()
	for _, question := range attempt.Questions {
		if question.QuestionID == questionID {
			return question
		}
	}
	t.Fatalf("question id %d not present in attempt %d", questionID, attempt.ID)
	return models.AttemptQuestion{}
}

func TestMilestone7AttemptScoringAndWrongAnswerRules(t *testing.T) {
	app := newTestApp(t)
	defer app.DB.Close()
	studentToken := app.login("student1", "student123")

	first := startAttempt(t, app, studentToken, http.StatusCreated)
	if first.QuestionCount != 10 || len(first.Questions) != 10 {
		t.Fatalf("attempt should contain exactly 10 questions, got count=%d len=%d", first.QuestionCount, len(first.Questions))
	}
	seenQuestions := map[int64]bool{}
	for _, question := range first.Questions {
		if seenQuestions[question.QuestionID] {
			t.Fatalf("attempt contains duplicate source question %d", question.QuestionID)
		}
		seenQuestions[question.QuestionID] = true
	}

	same := startAttempt(t, app, studentToken, http.StatusOK)
	if same.ID != first.ID {
		t.Fatalf("second start while in progress returned attempt %d, want existing attempt %d", same.ID, first.ID)
	}
	var inProgressCount int
	if err := app.DB.QueryRow(`SELECT COUNT(*) FROM attempts WHERE student_id = (SELECT id FROM users WHERE username = 'student1') AND status = 'in_progress'`).Scan(&inProgressCount); err != nil {
		t.Fatalf("count in-progress attempts: %v", err)
	}
	if inProgressCount != 1 {
		t.Fatalf("in-progress attempts = %d, want 1", inProgressCount)
	}

	firstCorrectQuestion := first.Questions[0]
	latestWrongQuestion := first.Questions[1]
	correctResponse := answerQuestion(t, app, studentToken, first.ID, firstCorrectQuestion.ID, 0)
	if !correctResponse.Answer.IsCorrect || correctResponse.Answer.CorrectOption != 0 || correctResponse.Answer.ScoreDelta != 1 {
		t.Fatalf("first correct answer feedback = %+v, want correct option 0 with +1", correctResponse.Answer)
	}
	wrongResponse := answerQuestion(t, app, studentToken, first.ID, latestWrongQuestion.ID, 1)
	if wrongResponse.Answer.IsCorrect || wrongResponse.Answer.CorrectOption != 0 || wrongResponse.Answer.ScoreDelta != 0 {
		t.Fatalf("wrong answer feedback = %+v, want immediate incorrect feedback with no score", wrongResponse.Answer)
	}

	latest := wrongResponse.Attempt
	for _, question := range latest.Questions {
		if question.Answer == nil {
			latest = answerQuestion(t, app, studentToken, first.ID, question.ID, 0).Attempt
		}
	}
	if latest.Status != "completed" || latest.AnsweredCount != 10 {
		t.Fatalf("completed first attempt status=%s answered=%d, want completed/10", latest.Status, latest.AnsweredCount)
	}

	second := startAttempt(t, app, studentToken, http.StatusCreated)
	repeatCorrect := questionByQuestionID(t, second, firstCorrectQuestion.QuestionID)
	repeatCorrectResponse := answerQuestion(t, app, studentToken, second.ID, repeatCorrect.ID, 0)
	if !repeatCorrectResponse.Answer.IsCorrect || repeatCorrectResponse.Answer.ScoreDelta != 0 {
		t.Fatalf("repeat correct answer feedback = %+v, want correct with no additional score", repeatCorrectResponse.Answer)
	}
	repeatWrong := questionByQuestionID(t, repeatCorrectResponse.Attempt, latestWrongQuestion.QuestionID)
	answerQuestion(t, app, studentToken, second.ID, repeatWrong.ID, 2)

	var latestSelected int
	var latestAttemptID int64
	if err := app.DB.QueryRow(`SELECT selected_option, attempt_id FROM wrong_answers WHERE student_id = (SELECT id FROM users WHERE username = 'student1') AND question_id = ?`, latestWrongQuestion.QuestionID).Scan(&latestSelected, &latestAttemptID); err != nil {
		t.Fatalf("read latest wrong answer: %v", err)
	}
	if latestSelected != 2 || latestAttemptID != second.ID {
		t.Fatalf("wrong answer record selected=%d attempt=%d, want latest selected=2 attempt=%d", latestSelected, latestAttemptID, second.ID)
	}

	var score scoreEnvelope
	app.doJSON(http.MethodGet, "/api/student/score", studentToken, nil, http.StatusOK, &score)
	if score.Score.TotalScore != 9 {
		t.Fatalf("student total score = %d, want only first correct per question counted (9)", score.Score.TotalScore)
	}
}

func TestMilestone7AdminResetAttemptsClearsStudentState(t *testing.T) {
	app := newTestApp(t)
	defer app.DB.Close()
	studentToken := app.login("student1", "student123")
	adminToken := app.login("admin", "admin123")

	attempt := startAttempt(t, app, studentToken, http.StatusCreated)
	answerQuestion(t, app, studentToken, attempt.ID, attempt.Questions[0].ID, 0)
	answerQuestion(t, app, studentToken, attempt.ID, attempt.Questions[1].ID, 1)

	var studentID int64
	if err := app.DB.QueryRow(`SELECT id FROM users WHERE username = 'student1'`).Scan(&studentID); err != nil {
		t.Fatalf("find seeded student: %v", err)
	}
	app.doJSON(http.MethodPost, fmt.Sprintf("/api/admin/students/%d/reset-attempts", studentID), adminToken, nil, http.StatusOK, nil)

	for table, query := range map[string]string{
		"attempts":                  `SELECT COUNT(*) FROM attempts WHERE student_id = ?`,
		"wrong_answers":             `SELECT COUNT(*) FROM wrong_answers WHERE student_id = ?`,
		"student_question_progress": `SELECT COUNT(*) FROM student_question_progress WHERE student_id = ?`,
	} {
		var count int
		if err := app.DB.QueryRow(query, studentID).Scan(&count); err != nil {
			t.Fatalf("count %s after reset: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("%s rows after reset = %d, want 0", table, count)
		}
	}
	var score scoreEnvelope
	app.doJSON(http.MethodGet, "/api/student/score", studentToken, nil, http.StatusOK, &score)
	if score.Score.TotalScore != 0 {
		t.Fatalf("score after attempt reset = %d, want 0", score.Score.TotalScore)
	}
	var current attemptEnvelope
	app.doJSON(http.MethodGet, "/api/student/attempts/current", studentToken, nil, http.StatusOK, &current)
	if current.Attempt.ID != 0 {
		t.Fatalf("current attempt after reset = %d, want none", current.Attempt.ID)
	}
}

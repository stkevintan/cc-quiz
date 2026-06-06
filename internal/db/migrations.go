package db

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type MigrationInfo struct {
	AppliedCount int `json:"appliedCount"`
}

func Migrate(database *sql.DB) (MigrationInfo, error) {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (
version TEXT PRIMARY KEY,
applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
)`,
		`CREATE TABLE IF NOT EXISTS app_metadata (
key TEXT PRIMARY KEY,
value TEXT NOT NULL,
updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
)`,
		`INSERT OR IGNORE INTO schema_migrations (version) VALUES ('0001_bootstrap')`,
		`INSERT OR IGNORE INTO app_metadata (key, value) VALUES ('schema_version', '0001_bootstrap')`,
		`CREATE TABLE IF NOT EXISTS users (
id INTEGER PRIMARY KEY AUTOINCREMENT,
username TEXT NOT NULL UNIQUE,
password_hash TEXT NOT NULL,
display_name TEXT NOT NULL,
role TEXT NOT NULL CHECK (role IN ('admin', 'teacher', 'student')),
status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
created_by INTEGER,
created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
FOREIGN KEY (created_by) REFERENCES users(id)
)`,
		`CREATE TABLE IF NOT EXISTS classes (
id INTEGER PRIMARY KEY AUTOINCREMENT,
name TEXT NOT NULL UNIQUE,
description TEXT NOT NULL DEFAULT '',
status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
created_by INTEGER,
created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
FOREIGN KEY (created_by) REFERENCES users(id)
)`,
		`CREATE TABLE IF NOT EXISTS class_teachers (
class_id INTEGER NOT NULL,
teacher_id INTEGER NOT NULL,
created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
PRIMARY KEY (class_id, teacher_id),
FOREIGN KEY (class_id) REFERENCES classes(id) ON DELETE CASCADE,
FOREIGN KEY (teacher_id) REFERENCES users(id) ON DELETE CASCADE
)`,
		`CREATE TABLE IF NOT EXISTS class_students (
class_id INTEGER NOT NULL,
student_id INTEGER NOT NULL UNIQUE,
created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
PRIMARY KEY (class_id, student_id),
FOREIGN KEY (class_id) REFERENCES classes(id) ON DELETE CASCADE,
FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE
)`,
		`CREATE INDEX IF NOT EXISTS idx_users_role_status ON users(role, status)`,
		`CREATE INDEX IF NOT EXISTS idx_class_teachers_teacher ON class_teachers(teacher_id)`,
		`CREATE INDEX IF NOT EXISTS idx_class_students_student ON class_students(student_id)`,
		`INSERT OR IGNORE INTO schema_migrations (version) VALUES ('0002_accounts_roles_permissions')`,
		`UPDATE app_metadata SET value = '0002_accounts_roles_permissions', updated_at = CURRENT_TIMESTAMP WHERE key = 'schema_version'`,
		`CREATE TABLE IF NOT EXISTS questions (
id INTEGER PRIMARY KEY AUTOINCREMENT,
prompt TEXT NOT NULL,
option_count INTEGER NOT NULL CHECK (option_count BETWEEN 2 AND 4),
option_a TEXT NOT NULL,
option_b TEXT NOT NULL,
option_c TEXT,
option_d TEXT,
correct_option INTEGER NOT NULL CHECK (correct_option BETWEEN 0 AND 3),
explanation TEXT NOT NULL DEFAULT '',
status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
created_by INTEGER,
created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
CHECK (correct_option < option_count),
CHECK (option_count >= 3 OR option_c IS NULL),
CHECK (option_count >= 4 OR option_d IS NULL),
FOREIGN KEY (created_by) REFERENCES users(id)
)`,
		`CREATE INDEX IF NOT EXISTS idx_questions_status ON questions(status)`,
		`INSERT OR IGNORE INTO schema_migrations (version) VALUES ('0003_question_bank')`,
		`UPDATE app_metadata SET value = '0003_question_bank', updated_at = CURRENT_TIMESTAMP WHERE key = 'schema_version'`,
		`CREATE TABLE IF NOT EXISTS attempts (
id INTEGER PRIMARY KEY AUTOINCREMENT,
student_id INTEGER NOT NULL,
status TEXT NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'completed')),
question_count INTEGER NOT NULL DEFAULT 10 CHECK (question_count = 10),
created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
completed_at TEXT,
FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE
)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_attempts_one_in_progress ON attempts(student_id) WHERE status = 'in_progress'`,
		`CREATE INDEX IF NOT EXISTS idx_attempts_student_status ON attempts(student_id, status)`,
		`CREATE TABLE IF NOT EXISTS attempt_questions (
id INTEGER PRIMARY KEY AUTOINCREMENT,
attempt_id INTEGER NOT NULL,
question_id INTEGER NOT NULL,
position INTEGER NOT NULL CHECK (position BETWEEN 0 AND 9),
prompt TEXT NOT NULL,
option_count INTEGER NOT NULL CHECK (option_count BETWEEN 2 AND 4),
option_a TEXT NOT NULL,
option_b TEXT NOT NULL,
option_c TEXT,
option_d TEXT,
correct_option INTEGER NOT NULL CHECK (correct_option BETWEEN 0 AND 3),
explanation TEXT NOT NULL DEFAULT '',
created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
UNIQUE (attempt_id, position),
UNIQUE (attempt_id, question_id),
CHECK (correct_option < option_count),
CHECK (option_count >= 3 OR option_c IS NULL),
CHECK (option_count >= 4 OR option_d IS NULL),
FOREIGN KEY (attempt_id) REFERENCES attempts(id) ON DELETE CASCADE,
FOREIGN KEY (question_id) REFERENCES questions(id)
)`,
		`CREATE INDEX IF NOT EXISTS idx_attempt_questions_attempt ON attempt_questions(attempt_id, position)`,
		`CREATE TABLE IF NOT EXISTS attempt_answers (
id INTEGER PRIMARY KEY AUTOINCREMENT,
attempt_id INTEGER NOT NULL,
attempt_question_id INTEGER NOT NULL UNIQUE,
student_id INTEGER NOT NULL,
selected_option INTEGER NOT NULL CHECK (selected_option BETWEEN 0 AND 3),
is_correct INTEGER NOT NULL CHECK (is_correct IN (0, 1)),
score_delta INTEGER NOT NULL DEFAULT 0 CHECK (score_delta IN (0, 1)),
answered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
FOREIGN KEY (attempt_id) REFERENCES attempts(id) ON DELETE CASCADE,
FOREIGN KEY (attempt_question_id) REFERENCES attempt_questions(id) ON DELETE CASCADE,
FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE
)`,
		`CREATE INDEX IF NOT EXISTS idx_attempt_answers_attempt ON attempt_answers(attempt_id)`,
		`INSERT OR IGNORE INTO schema_migrations (version) VALUES ('0004_student_attempts')`,
		`UPDATE app_metadata SET value = '0004_student_attempts', updated_at = CURRENT_TIMESTAMP WHERE key = 'schema_version'`,
		`CREATE TABLE IF NOT EXISTS student_question_progress (
student_id INTEGER NOT NULL,
question_id INTEGER NOT NULL,
has_earned_score INTEGER NOT NULL DEFAULT 0 CHECK (has_earned_score IN (0, 1)),
first_correct_attempt_id INTEGER,
first_correct_at TEXT,
last_answered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
last_is_correct INTEGER NOT NULL DEFAULT 0 CHECK (last_is_correct IN (0, 1)),
PRIMARY KEY (student_id, question_id),
FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE,
FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE,
FOREIGN KEY (first_correct_attempt_id) REFERENCES attempts(id) ON DELETE SET NULL
)`,
		`CREATE TABLE IF NOT EXISTS wrong_answers (
id INTEGER PRIMARY KEY AUTOINCREMENT,
student_id INTEGER NOT NULL,
attempt_id INTEGER NOT NULL,
question_id INTEGER NOT NULL,
selected_option INTEGER NOT NULL CHECK (selected_option BETWEEN 0 AND 3),
correct_option INTEGER NOT NULL CHECK (correct_option BETWEEN 0 AND 3),
updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
UNIQUE (student_id, question_id),
FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE,
FOREIGN KEY (attempt_id) REFERENCES attempts(id) ON DELETE CASCADE,
FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE
)`,
		`CREATE TABLE IF NOT EXISTS student_scores (
student_id INTEGER PRIMARY KEY,
total_score INTEGER NOT NULL DEFAULT 0 CHECK (total_score >= 0),
updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE
)`,
		`CREATE INDEX IF NOT EXISTS idx_wrong_answers_student ON wrong_answers(student_id, updated_at DESC)`,
		`INSERT OR IGNORE INTO schema_migrations (version) VALUES ('0005_scoring_progress_wrong_answers')`,
		`UPDATE app_metadata SET value = '0005_scoring_progress_wrong_answers', updated_at = CURRENT_TIMESTAMP WHERE key = 'schema_version'`,
	}

	for _, statement := range statements {
		if _, err := database.Exec(statement); err != nil {
			return MigrationInfo{}, fmt.Errorf("run migration statement: %w", err)
		}
	}
	if err := ensureColumn(database, "attempt_answers", "score_delta", "INTEGER NOT NULL DEFAULT 0 CHECK (score_delta IN (0, 1))"); err != nil {
		return MigrationInfo{}, err
	}
	if err := backfillScoring(database); err != nil {
		return MigrationInfo{}, err
	}

	if err := SeedDevelopmentData(database); err != nil {
		return MigrationInfo{}, err
	}
	if err := SeedQuestionData(database); err != nil {
		return MigrationInfo{}, err
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		return MigrationInfo{}, fmt.Errorf("count migrations: %w", err)
	}
	return MigrationInfo{AppliedCount: count}, nil
}

func ensureColumn(database *sql.DB, table, column, definition string) error {
	rows, err := database.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan %s columns: %w", table, err)
		}
		if name == column {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read %s columns: %w", table, err)
	}
	if _, err := database.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, definition)); err != nil {
		return fmt.Errorf("add %s.%s: %w", table, column, err)
	}
	return nil
}

func backfillScoring(database *sql.DB) error {
	statements := []string{
		`UPDATE attempt_answers
SET score_delta = CASE WHEN is_correct = 1 AND id = (
	SELECT aa2.id
	FROM attempt_answers aa2
	JOIN attempt_questions aq2 ON aq2.id = aa2.attempt_question_id
	WHERE aa2.student_id = attempt_answers.student_id
	  AND aq2.question_id = (SELECT aq1.question_id FROM attempt_questions aq1 WHERE aq1.id = attempt_answers.attempt_question_id)
	  AND aa2.is_correct = 1
	ORDER BY aa2.answered_at, aa2.id
	LIMIT 1
) THEN 1 ELSE 0 END`,
		`INSERT INTO student_scores (student_id, total_score, updated_at)
SELECT student_id, SUM(score_delta), CURRENT_TIMESTAMP
FROM attempt_answers
GROUP BY student_id
ON CONFLICT(student_id) DO UPDATE SET total_score = excluded.total_score, updated_at = CURRENT_TIMESTAMP`,
		`INSERT INTO student_question_progress (
	student_id, question_id, has_earned_score, first_correct_attempt_id, first_correct_at, last_answered_at, last_is_correct
)
SELECT
	pairs.student_id,
	pairs.question_id,
	CASE WHEN first_correct.id IS NULL THEN 0 ELSE 1 END,
	first_correct.attempt_id,
	first_correct.answered_at,
	last_answer.answered_at,
	last_answer.is_correct
FROM (
	SELECT DISTINCT aa.student_id, aq.question_id
	FROM attempt_answers aa
	JOIN attempt_questions aq ON aq.id = aa.attempt_question_id
) pairs
LEFT JOIN attempt_answers first_correct ON first_correct.id = (
	SELECT aa2.id
	FROM attempt_answers aa2
	JOIN attempt_questions aq2 ON aq2.id = aa2.attempt_question_id
	WHERE aa2.student_id = pairs.student_id
	  AND aq2.question_id = pairs.question_id
	  AND aa2.is_correct = 1
	ORDER BY aa2.answered_at, aa2.id
	LIMIT 1
)
JOIN attempt_answers last_answer ON last_answer.id = (
	SELECT aa3.id
	FROM attempt_answers aa3
	JOIN attempt_questions aq3 ON aq3.id = aa3.attempt_question_id
	WHERE aa3.student_id = pairs.student_id
	  AND aq3.question_id = pairs.question_id
	ORDER BY aa3.answered_at DESC, aa3.id DESC
	LIMIT 1
)
ON CONFLICT(student_id, question_id) DO UPDATE SET
	has_earned_score = excluded.has_earned_score,
	first_correct_attempt_id = excluded.first_correct_attempt_id,
	first_correct_at = excluded.first_correct_at,
	last_answered_at = excluded.last_answered_at,
	last_is_correct = excluded.last_is_correct`,
		`INSERT INTO wrong_answers (student_id, attempt_id, question_id, selected_option, correct_option, updated_at)
SELECT aa.student_id, aa.attempt_id, aq.question_id, aa.selected_option, aq.correct_option, aa.answered_at
FROM attempt_answers aa
JOIN attempt_questions aq ON aq.id = aa.attempt_question_id
WHERE aa.is_correct = 0
  AND aa.id = (
	SELECT aa2.id
	FROM attempt_answers aa2
	JOIN attempt_questions aq2 ON aq2.id = aa2.attempt_question_id
	WHERE aa2.student_id = aa.student_id
	  AND aq2.question_id = aq.question_id
	  AND aa2.is_correct = 0
	ORDER BY aa2.answered_at DESC, aa2.id DESC
	LIMIT 1
  )
ON CONFLICT(student_id, question_id) DO UPDATE SET
	attempt_id = excluded.attempt_id,
	selected_option = excluded.selected_option,
	correct_option = excluded.correct_option,
	updated_at = excluded.updated_at`,
	}
	for _, statement := range statements {
		if _, err := database.Exec(statement); err != nil {
			return fmt.Errorf("backfill scoring: %w", err)
		}
	}
	return nil
}

func SeedQuestionData(database *sql.DB) error {
	var questionCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM questions`).Scan(&questionCount); err != nil {
		return fmt.Errorf("count questions before seed: %w", err)
	}
	if questionCount >= 10 {
		return nil
	}

	var creatorID sql.NullInt64
	if err := database.QueryRow(`SELECT id FROM users WHERE role IN ('admin', 'teacher') ORDER BY CASE role WHEN 'admin' THEN 0 ELSE 1 END, id LIMIT 1`).Scan(&creatorID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("find question seed creator: %w", err)
	}

	samples := []struct {
		prompt        string
		options       []string
		correctOption int
		explanation   string
	}{
		{
			prompt:        "“学而时习之，不亦说乎？”中“说”的意思是？",
			options:       []string{"说话", "通“悦”，高兴", "劝说", "解释"},
			correctOption: 1,
			explanation:   "“说”在此处通“悦”，表示愉快、高兴。",
		},
		{
			prompt:        "“温故而知新”中“故”的意思是？",
			options:       []string{"旧的知识", "缘故", "故意", "故事"},
			correctOption: 0,
			explanation:   "“故”指已经学过的旧知识。",
		},
		{
			prompt:        "下列哪一项最适合解释“其恕乎”？",
			options:       []string{"大概是恕道吧", "他原谅了吧", "这是书信吧", "其中有错误吧"},
			correctOption: 0,
			explanation:   "“其……乎”常表揣测语气，可译为“大概……吧”。",
		},
		{
			prompt:        "“吾日三省吾身”中“省”的意思是？",
			options:       []string{"节省", "反省、检查", "省略", "探望"},
			correctOption: 1,
			explanation:   "“省”读 xǐng，表示反省、检查自己的言行。",
		},
		{
			prompt:        "“人不知而不愠”中“愠”的意思是？",
			options:       []string{"怨恨、生气", "温暖", "忧虑", "困倦"},
			correctOption: 0,
			explanation:   "“愠”指恼怒、怨恨。",
		},
		{
			prompt:        "“学而不思则罔”中“罔”的意思是？",
			options:       []string{"迷惑而无所得", "网罗", "欺骗", "广阔"},
			correctOption: 0,
			explanation:   "此处“罔”表示迷惑、没有收获。",
		},
		{
			prompt:        "“思而不学则殆”中“殆”的意思是？",
			options:       []string{"危险、有害", "几乎", "等待", "懈怠"},
			correctOption: 0,
			explanation:   "“殆”在这里表示疑惑危险，指精神疲殆而无所得。",
		},
		{
			prompt:        "“知之者不如好之者”中“好”的意思是？",
			options:       []string{"美好", "喜爱", "容易", "完成"},
			correctOption: 1,
			explanation:   "“好”读 hào，表示喜爱。",
		},
		{
			prompt:        "“逝者如斯夫”中“斯”的意思是？",
			options:       []string{"这、此", "慢慢", "停止", "河岸"},
			correctOption: 0,
			explanation:   "“斯”是代词，指这流水。",
		},
		{
			prompt:        "“三人行，必有我师焉”中“焉”的用法是？",
			options:       []string{"兼词，于此/在其中", "疑问代词", "句首发语词", "通假字"},
			correctOption: 0,
			explanation:   "“焉”在此相当于“于之”，可译为“在其中”。",
		},
		{
			prompt:        "“择其善者而从之”中“从”的意思是？",
			options:       []string{"跟随、学习", "从前", "纵使", "从容"},
			correctOption: 0,
			explanation:   "“从”表示跟随、学习、采纳。",
		},
	}
	for _, sample := range samples {
		var exists int
		if err := database.QueryRow(`SELECT COUNT(*) FROM questions WHERE prompt = ?`, sample.prompt).Scan(&exists); err != nil {
			return fmt.Errorf("check seed question: %w", err)
		}
		if exists > 0 {
			continue
		}
		if _, err := database.Exec(`INSERT INTO questions (prompt, option_count, option_a, option_b, option_c, option_d, correct_option, explanation, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sample.prompt, len(sample.options), sample.options[0], sample.options[1], optionalSeedOption(sample.options, 2), optionalSeedOption(sample.options, 3), sample.correctOption, sample.explanation, creatorID.Int64); err != nil {
			return fmt.Errorf("seed question: %w", err)
		}
	}
	return nil
}

func optionalSeedOption(options []string, index int) any {
	if len(options) <= index {
		return nil
	}
	return options[index]
}

func SeedDevelopmentData(database *sql.DB) error {
	var userCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return fmt.Errorf("count users before seed: %w", err)
	}
	if userCount > 0 {
		return nil
	}

	adminHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	teacherHash, err := bcrypt.GenerateFromPassword([]byte("teacher123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash teacher password: %w", err)
	}
	studentHash, err := bcrypt.GenerateFromPassword([]byte("student123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash student password: %w", err)
	}

	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback()

	adminResult, err := tx.Exec(`INSERT INTO users (username, password_hash, display_name, role) VALUES (?, ?, ?, ?)`,
		"admin", string(adminHash), "默认管理员", "admin")
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	adminID, _ := adminResult.LastInsertId()

	teacherResult, err := tx.Exec(`INSERT INTO users (username, password_hash, display_name, role, created_by) VALUES (?, ?, ?, ?, ?)`,
		"teacher1", string(teacherHash), "示例教师", "teacher", adminID)
	if err != nil {
		return fmt.Errorf("seed teacher: %w", err)
	}
	teacherID, _ := teacherResult.LastInsertId()

	studentResult, err := tx.Exec(`INSERT INTO users (username, password_hash, display_name, role, created_by) VALUES (?, ?, ?, ?, ?)`,
		"student1", string(studentHash), "示例学生", "student", adminID)
	if err != nil {
		return fmt.Errorf("seed student: %w", err)
	}
	studentID, _ := studentResult.LastInsertId()

	classResult, err := tx.Exec(`INSERT INTO classes (name, description, created_by) VALUES (?, ?, ?)`,
		"一班", "本地开发示例班级", adminID)
	if err != nil {
		return fmt.Errorf("seed class: %w", err)
	}
	classID, _ := classResult.LastInsertId()

	if _, err := tx.Exec(`INSERT INTO class_teachers (class_id, teacher_id) VALUES (?, ?)`, classID, teacherID); err != nil {
		return fmt.Errorf("seed class teacher: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO class_students (class_id, student_id) VALUES (?, ?)`, classID, studentID); err != nil {
		return fmt.Errorf("seed class student: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	return nil
}

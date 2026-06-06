package models

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
)

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Role        Role   `json:"role"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type Class struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type ClassSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Question struct {
	ID            int64    `json:"id"`
	Prompt        string   `json:"prompt"`
	Options       []string `json:"options"`
	OptionCount   int      `json:"optionCount"`
	CorrectOption int      `json:"correctOption"`
	Explanation   string   `json:"explanation"`
	Status        string   `json:"status"`
	CreatedBy     int64    `json:"createdBy,omitempty"`
	CreatedAt     string   `json:"createdAt,omitempty"`
	UpdatedAt     string   `json:"updatedAt,omitempty"`
}

type Attempt struct {
	ID            int64             `json:"id"`
	StudentID     int64             `json:"studentId"`
	Status        string            `json:"status"`
	QuestionCount int               `json:"questionCount"`
	AnsweredCount int               `json:"answeredCount"`
	ScoreDelta    int               `json:"scoreDelta"`
	CreatedAt     string            `json:"createdAt"`
	UpdatedAt     string            `json:"updatedAt"`
	CompletedAt   string            `json:"completedAt,omitempty"`
	Questions     []AttemptQuestion `json:"questions"`
}

type AttemptQuestion struct {
	ID            int64          `json:"id"`
	AttemptID     int64          `json:"attemptId"`
	QuestionID    int64          `json:"questionId"`
	Position      int            `json:"position"`
	Prompt        string         `json:"prompt"`
	Options       []string       `json:"options"`
	OptionCount   int            `json:"optionCount"`
	CorrectOption *int           `json:"correctOption,omitempty"`
	Explanation   string         `json:"explanation,omitempty"`
	Answer        *AttemptAnswer `json:"answer,omitempty"`
}

type AttemptAnswer struct {
	ID                int64  `json:"id"`
	AttemptID         int64  `json:"attemptId"`
	AttemptQuestionID int64  `json:"attemptQuestionId"`
	SelectedOption    int    `json:"selectedOption"`
	IsCorrect         bool   `json:"isCorrect"`
	ScoreDelta        int    `json:"scoreDelta"`
	AnsweredAt        string `json:"answeredAt"`
}

type StudentScore struct {
	StudentID  int64  `json:"studentId"`
	TotalScore int    `json:"totalScore"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

type WrongAnswer struct {
	ID             int64    `json:"id"`
	StudentID      int64    `json:"studentId"`
	AttemptID      int64    `json:"attemptId"`
	QuestionID     int64    `json:"questionId"`
	Prompt         string   `json:"prompt"`
	Options        []string `json:"options"`
	OptionCount    int      `json:"optionCount"`
	SelectedOption int      `json:"selectedOption"`
	CorrectOption  int      `json:"correctOption"`
	Explanation    string   `json:"explanation"`
	UpdatedAt      string   `json:"updatedAt"`
}

type AttemptSummary struct {
	ID            int64  `json:"id"`
	StudentID     int64  `json:"studentId"`
	Status        string `json:"status"`
	QuestionCount int    `json:"questionCount"`
	AnsweredCount int    `json:"answeredCount"`
	ScoreDelta    int    `json:"scoreDelta"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	CompletedAt   string `json:"completedAt,omitempty"`
}

type StudentReport struct {
	ID             int64            `json:"id"`
	Username       string           `json:"username"`
	DisplayName    string           `json:"displayName"`
	Role           Role             `json:"role"`
	Status         string           `json:"status"`
	Classes        []ClassSummary   `json:"classes"`
	Score          StudentScore     `json:"score"`
	AttemptCount   int              `json:"attemptCount"`
	CompletedCount int              `json:"completedCount"`
	InProgress     bool             `json:"inProgress"`
	WrongCount     int              `json:"wrongCount"`
	Attempts       []AttemptSummary `json:"attempts,omitempty"`
}

package routes

import (
	"database/sql"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"classical-chinese-quiz/backend/internal/config"
	"classical-chinese-quiz/backend/internal/handlers"
	"classical-chinese-quiz/backend/internal/middleware"
	"classical-chinese-quiz/backend/web"
	"github.com/gin-gonic/gin"
)

type frontendAssets interface {
	HasDistIndex() bool
	HasDistAsset(name string) bool
	ReadDistAsset(name string) ([]byte, error)
	FallbackIndexHTML() ([]byte, error)
}

type embeddedFrontendAssets struct{}

func (embeddedFrontendAssets) HasDistIndex() bool {
	return web.HasDistIndex()
}

func (embeddedFrontendAssets) HasDistAsset(name string) bool {
	return web.HasDistAsset(name)
}

func (embeddedFrontendAssets) ReadDistAsset(name string) ([]byte, error) {
	return web.ReadDistAsset(name)
}

func (embeddedFrontendAssets) FallbackIndexHTML() ([]byte, error) {
	return web.FallbackIndexHTML()
}

func NewRouter(cfg config.Config, database *sql.DB) *gin.Engine {
	return newRouter(cfg, database, embeddedFrontendAssets{})
}

func newRouter(cfg config.Config, database *sql.DB, assets frontendAssets) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORS(cfg.CORSAllowedOrigins))

	healthHandler := handlers.HealthHandler{
		DB:          database,
		AppEnv:      cfg.AppEnv,
		ServiceName: "classical-chinese-quiz-api",
	}
	authHandler := handlers.AuthHandler{DB: database, JWTSecret: cfg.JWTSecret}
	adminHandler := handlers.AdminHandler{DB: database}
	teacherHandler := handlers.TeacherHandler{DB: database}
	questionHandler := handlers.QuestionHandler{DB: database}
	studentAttemptHandler := handlers.StudentAttemptHandler{DB: database}

	api := router.Group("/api")
	api.GET("/health", healthHandler.Get)
	api.POST("/auth/login", authHandler.Login)

	authenticated := api.Group("")
	authenticated.Use(middleware.Auth(cfg.JWTSecret, database))
	authenticated.GET("/auth/me", authHandler.Me)
	authenticated.POST("/auth/logout", authHandler.Logout)

	questions := authenticated.Group("/questions")
	questions.Use(middleware.RequireRoles("admin", "teacher"))
	questions.GET("", questionHandler.List)
	questions.POST("", questionHandler.Create)
	questions.GET("/:id", questionHandler.Get)
	questions.PATCH("/:id", questionHandler.Update)
	questions.POST("/:id/disable", questionHandler.Disable)

	admin := authenticated.Group("/admin")
	admin.Use(middleware.RequireRoles("admin"))
	admin.GET("/users", adminHandler.ListUsers)
	admin.POST("/users", adminHandler.CreateUser)
	admin.POST("/users/:id/disable", adminHandler.DisableUser)
	admin.POST("/users/:id/reset-password", adminHandler.ResetPassword)
	admin.GET("/students", adminHandler.ListStudentsReport)
	admin.GET("/students/:studentId", adminHandler.GetStudentReport)
	admin.GET("/students/:studentId/wrong-answers", adminHandler.GetStudentWrongAnswers)
	admin.POST("/students/:studentId/reset-score", adminHandler.ResetStudentScore)
	admin.POST("/students/:studentId/reset-attempts", adminHandler.ResetStudentAttempts)
	admin.GET("/classes", adminHandler.ListClasses)
	admin.POST("/classes", adminHandler.CreateClass)
	admin.PATCH("/classes/:id", adminHandler.UpdateClass)
	admin.POST("/classes/:id/teachers/:teacherId", adminHandler.AddClassTeacher)
	admin.DELETE("/classes/:id/teachers/:teacherId", adminHandler.RemoveClassTeacher)
	admin.POST("/classes/:id/students/:studentId", adminHandler.AddClassStudent)
	admin.DELETE("/classes/:id/students/:studentId", adminHandler.RemoveClassStudent)

	teacher := authenticated.Group("/teacher")
	teacher.Use(middleware.RequireRoles("teacher"))
	teacher.GET("/students", teacherHandler.ListStudents)
	teacher.GET("/students/:studentId", teacherHandler.GetStudent)
	teacher.GET("/students/:studentId/wrong-answers", teacherHandler.GetStudentWrongAnswers)

	student := authenticated.Group("/student")
	student.Use(middleware.RequireRoles("student"))
	student.GET("/score", studentAttemptHandler.Score)
	student.GET("/wrong-answers", studentAttemptHandler.WrongAnswers)
	student.POST("/attempts", studentAttemptHandler.Create)
	student.GET("/attempts/current", studentAttemptHandler.Current)
	student.GET("/attempts/:attemptId", studentAttemptHandler.Get)
	student.POST("/attempts/:attemptId/answers", studentAttemptHandler.Answer)

	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}

		requestPath := path.Clean(c.Request.URL.Path)
		if strings.HasPrefix(requestPath, "/api/") || requestPath == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		if assets.HasDistIndex() {
			assetPath := strings.TrimPrefix(requestPath, "/")
			if assetPath == "" {
				if err := serveEmbeddedAsset(c, assets, "index.html"); err != nil {
					c.Status(http.StatusInternalServerError)
				}
				return
			}
			if assets.HasDistAsset(assetPath) {
				if err := serveEmbeddedAsset(c, assets, assetPath); err != nil {
					c.Status(http.StatusInternalServerError)
				}
				return
			}
			if path.Ext(assetPath) == "" {
				if err := serveEmbeddedAsset(c, assets, "index.html"); err != nil {
					c.Status(http.StatusInternalServerError)
				}
				return
			}
			c.Status(http.StatusNotFound)
			return
		}

		fallbackIndex, err := assets.FallbackIndexHTML()
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", fallbackIndex)
	})

	return router
}

func serveEmbeddedAsset(c *gin.Context, assets frontendAssets, assetPath string) error {
	body, err := assets.ReadDistAsset(assetPath)
	if err != nil {
		return err
	}

	contentType := mime.TypeByExtension(path.Ext(assetPath))
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.Itoa(len(body)))
	c.Status(http.StatusOK)
	if c.Request.Method == http.MethodHead {
		return nil
	}

	_, err = c.Writer.Write(body)
	return err
}

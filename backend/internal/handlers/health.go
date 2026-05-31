package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	DB          *sql.DB
	AppEnv      string
	ServiceName string
}

type healthResponse struct {
	Status      string         `json:"status"`
	Service     string         `json:"service"`
	Environment string         `json:"environment"`
	Time        string         `json:"time"`
	Database    databaseHealth `json:"database"`
}

type databaseHealth struct {
	Status            string `json:"status"`
	AppliedMigrations int    `json:"appliedMigrations"`
	Error             string `json:"error,omitempty"`
}

func (h HealthHandler) Get(c *gin.Context) {
	dbHealth := databaseHealth{Status: "ok"}
	statusCode := http.StatusOK
	status := "ok"

	if err := h.DB.PingContext(c.Request.Context()); err != nil {
		dbHealth.Status = "error"
		dbHealth.Error = err.Error()
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	} else if err := h.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM schema_migrations`).Scan(&dbHealth.AppliedMigrations); err != nil {
		dbHealth.Status = "error"
		dbHealth.Error = err.Error()
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, healthResponse{
		Status:      status,
		Service:     h.ServiceName,
		Environment: h.AppEnv,
		Time:        time.Now().UTC().Format(time.RFC3339),
		Database:    dbHealth,
	})
}

package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a cached user from auth service
type User struct {
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Task represents a task in the system
type Task struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description,omitempty" db:"description"`
	Status      string     `json:"status" db:"status"`
	Priority    string     `json:"priority" db:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty" db:"due_date"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateTaskRequest represents the request body for creating a task
type CreateTaskRequest struct {
	Title       string     `json:"title" binding:"required,min=1,max=255"`
	Description *string    `json:"description,omitempty"`
	Status      *string    `json:"status,omitempty"`
	Priority    *string    `json:"priority,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// UpdateTaskRequest represents the request body for updating a task
type UpdateTaskRequest struct {
	Title       *string    `json:"title,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string    `json:"description,omitempty"`
	Status      *string    `json:"status,omitempty"`
	Priority    *string    `json:"priority,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// TaskFilters represents query parameters for filtering tasks
type TaskFilters struct {
	Status   string `form:"status"`
	Priority string `form:"priority"`
	Page     int    `form:"page,default=1"`
	Limit    int    `form:"limit,default=10"`
}

// TaskStats represents task statistics
type TaskStats struct {
	TotalTasks     int            `json:"total_tasks"`
	ByStatus       map[string]int `json:"by_status"`
	ByPriority     map[string]int `json:"by_priority"`
	OverdueTasks   int            `json:"overdue_tasks"`
	CompletedToday int            `json:"completed_today"`
}

// UserEvent represents an event received from auth service
type UserEvent struct {
	EventType string    `json:"eventType"`
	UserID    uuid.UUID `json:"userId"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

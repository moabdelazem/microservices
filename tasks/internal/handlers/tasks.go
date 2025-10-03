package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/moabdelazem/microservices/tasks/internal/database"
	"github.com/moabdelazem/microservices/tasks/internal/models"
)

type TaskHandler struct {
	db *database.DB
}

func NewTaskHandler(db *database.DB) *TaskHandler {
	return &TaskHandler{db: db}
}

// CreateTask creates a new task
func (h *TaskHandler) CreateTask(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	status := "pending"
	if req.Status != nil {
		status = *req.Status
	}

	priority := "medium"
	if req.Priority != nil {
		priority = *req.Priority
	}

	// Validate status and priority
	if !isValidStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Must be: pending, in_progress, completed, or cancelled"})
		return
	}

	if !isValidPriority(priority) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid priority. Must be: low, medium, high, or urgent"})
		return
	}

	task := models.Task{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
		DueDate:     req.DueDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO tasks (id, user_id, title, description, status, priority, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := h.db.Exec(query, task.ID, task.UserID, task.Title, task.Description, task.Status, task.Priority, task.DueDate, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Task created successfully",
		"task":    task,
	})
}

// GetTasks retrieves tasks with optional filters
func (h *TaskHandler) GetTasks(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var filters models.TaskFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build query
	query := "SELECT * FROM tasks WHERE user_id = $1"
	args := []interface{}{userID}
	argCount := 1

	if filters.Status != "" {
		argCount++
		query += " AND status = $" + string(rune(argCount+'0'))
		args = append(args, filters.Status)
	}

	if filters.Priority != "" {
		argCount++
		query += " AND priority = $" + string(rune(argCount+'0'))
		args = append(args, filters.Priority)
	}

	query += " ORDER BY created_at DESC"

	// Pagination
	if filters.Limit <= 0 {
		filters.Limit = 10
	}
	if filters.Page <= 0 {
		filters.Page = 1
	}

	offset := (filters.Page - 1) * filters.Limit
	argCount++
	query += " LIMIT $" + string(rune(argCount+'0'))
	args = append(args, filters.Limit)

	argCount++
	query += " OFFSET $" + string(rune(argCount+'0'))
	args = append(args, offset)

	var tasks []models.Task
	err := h.db.Select(&tasks, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM tasks WHERE user_id = $1"
	countArgs := []interface{}{userID}
	if filters.Status != "" {
		countQuery += " AND status = $2"
		countArgs = append(countArgs, filters.Status)
	}
	if filters.Priority != "" {
		idx := len(countArgs) + 1
		countQuery += " AND priority = $" + string(rune(idx+'0'))
		countArgs = append(countArgs, filters.Priority)
	}

	var total int
	err = h.db.Get(&total, countQuery, countArgs...)
	if err != nil {
		total = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"pagination": gin.H{
			"page":  filters.Page,
			"limit": filters.Limit,
			"total": total,
		},
	})
}

// GetTask retrieves a single task by ID
func (h *TaskHandler) GetTask(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var task models.Task
	query := "SELECT * FROM tasks WHERE id = $1 AND user_id = $2"
	err = h.db.Get(&task, query, taskID, userID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task})
}

// UpdateTask updates a task
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var req models.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check task exists and belongs to user
	var exists bool
	err = h.db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1 AND user_id = $2)", taskID, userID)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Build update query dynamically
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		if !isValidStatus(*req.Status) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
			return
		}
		updates["status"] = *req.Status
	}
	if req.Priority != nil {
		if !isValidPriority(*req.Priority) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid priority"})
			return
		}
		updates["priority"] = *req.Priority
	}
	if req.DueDate != nil {
		updates["due_date"] = *req.DueDate
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates["updated_at"] = time.Now()

	// Build SQL
	query := "UPDATE tasks SET "
	args := []interface{}{}
	i := 1
	for key, val := range updates {
		if i > 1 {
			query += ", "
		}
		query += key + " = $" + string(rune(i+'0'))
		args = append(args, val)
		i++
	}
	query += " WHERE id = $" + string(rune(i+'0')) + " AND user_id = $" + string(rune(i+1+'0'))
	args = append(args, taskID, userID)

	_, err = h.db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Fetch updated task
	var task models.Task
	err = h.db.Get(&task, "SELECT * FROM tasks WHERE id = $1", taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task updated successfully",
		"task":    task,
	})
}

// DeleteTask deletes a task
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	result, err := h.db.Exec("DELETE FROM tasks WHERE id = $1 AND user_id = $2", taskID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}

// GetStats retrieves task statistics
func (h *TaskHandler) GetStats(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	stats := models.TaskStats{
		ByStatus:   make(map[string]int),
		ByPriority: make(map[string]int),
	}

	// Total tasks
	h.db.Get(&stats.TotalTasks, "SELECT COUNT(*) FROM tasks WHERE user_id = $1", userID)

	// By status
	rows, _ := h.db.Query("SELECT status, COUNT(*) FROM tasks WHERE user_id = $1 GROUP BY status", userID)
	for rows.Next() {
		var status string
		var count int
		rows.Scan(&status, &count)
		stats.ByStatus[status] = count
	}
	rows.Close()

	// By priority
	rows, _ = h.db.Query("SELECT priority, COUNT(*) FROM tasks WHERE user_id = $1 GROUP BY priority", userID)
	for rows.Next() {
		var priority string
		var count int
		rows.Scan(&priority, &count)
		stats.ByPriority[priority] = count
	}
	rows.Close()

	// Overdue tasks
	h.db.Get(&stats.OverdueTasks,
		"SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND due_date < NOW() AND status != 'completed'",
		userID)

	// Completed today
	h.db.Get(&stats.CompletedToday,
		"SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND status = 'completed' AND DATE(updated_at) = CURRENT_DATE",
		userID)

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// Health check
func (h *TaskHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "tasks-service",
	})
}

// Helper functions
func isValidStatus(status string) bool {
	validStatuses := []string{"pending", "in_progress", "completed", "cancelled"}
	for _, s := range validStatuses {
		if s == status {
			return true
		}
	}
	return false
}

func isValidPriority(priority string) bool {
	validPriorities := []string{"low", "medium", "high", "urgent"}
	for _, p := range validPriorities {
		if p == priority {
			return true
		}
	}
	return false
}

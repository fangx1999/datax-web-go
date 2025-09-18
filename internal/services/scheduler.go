package services

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/robfig/cron/v3"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// runningTask 表示当前正在执行的 DataX 任务
type runningTask struct {
	cancel context.CancelFunc
	cmd    *exec.Cmd
}

// runningTaskFlow 表示当前正在执行的任务流
type runningTaskFlow struct {
	cancel context.CancelFunc
	flowID int
}

// Scheduler 处理任务执行和任务流调度
// 任务只能手动执行或作为任务流的一部分执行
// 任务流基于 cron 表达式进行调度
type Scheduler struct {
	db        *sql.DB
	cron      *cron.Cron
	dataxHome string
	tempDir   string

	// 锁
	tasksMu sync.RWMutex // 保护 running tasks
	flowsMu sync.RWMutex // 保护 running flows
	cronMu  sync.RWMutex // 保护 cron entries

	// 数据
	tasks       map[int]*runningTask
	flows       map[int]*runningTaskFlow
	cronEntries map[int]cron.EntryID // flowID -> EntryID 用于追踪 cron 任务
}

// NewScheduler 使用提供的依赖项创建 Scheduler 实例
func NewScheduler(db *sql.DB, c *cron.Cron, dataxHome, tempDir string) *Scheduler {
	s := &Scheduler{
		db:          db,
		cron:        c,
		dataxHome:   dataxHome,
		tempDir:     tempDir,
		tasks:       make(map[int]*runningTask),
		flows:       make(map[int]*runningTaskFlow),
		cronEntries: make(map[int]cron.EntryID),
	}

	// 初始化时检查并准备临时目录
	s.initTempDir()

	return s
}

// initTempDir 初始化临时目录，检查是否存在或创建
func (s *Scheduler) initTempDir() {
	// 先检查目录是否已存在
	if _, err := os.Stat(s.tempDir); err == nil {
		// 目录已存在，直接使用
		log.Printf("scheduler: using existing temp directory: %s", s.tempDir)
		return
	}

	// 目录不存在，创建它
	if err := os.MkdirAll(s.tempDir, 0755); err != nil {
		log.Printf("scheduler: failed to create temp directory %s: %v", s.tempDir, err)
		return
	}

	log.Printf("scheduler: created temp directory: %s", s.tempDir)
}

// LoadAndStart 查询数据库中启用的任务流并调度它们
// 任务不会单独调度 - 只有任务流会被调度
func (s *Scheduler) LoadAndStart() {
	rows, err := s.db.Query("SELECT id, cron_expr FROM task_flows WHERE enabled=1")
	if err != nil {
		log.Printf("scheduler: failed to load task flows: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var expr string
		if err := rows.Scan(&id, &expr); err == nil {
			flowID := id
			entryID, err := s.cron.AddFunc(expr, func() {
				if err := s.RunTaskFlow(context.Background(), flowID); err != nil {
					log.Printf("scheduler: task flow %d error: %v", flowID, err)
				}
			})
			if err != nil {
				log.Printf("scheduler: failed to schedule task flow %d: %v", id, err)
			} else {
				// 存储 cron 条目 ID 用于后续管理
				s.cronMu.Lock()
				s.cronEntries[flowID] = entryID
				s.cronMu.Unlock()
			}
		}
	}
	s.cron.Start()
	log.Println("scheduler: task flow scheduler started")
}

// ========== 调度器管理方法 ==========

// ReloadTaskFlow 从数据库重新加载特定任务流并更新其 cron 调度
func (s *Scheduler) ReloadTaskFlow(flowID int) error {
	// 首先移除现有的 cron 条目（如果存在）
	s.cronMu.Lock()
	entryID, exists := s.cronEntries[flowID]
	if exists {
		s.cron.Remove(entryID)
		delete(s.cronEntries, flowID)
		log.Printf("scheduler: removed task flow %d from cron scheduler", flowID)
	}
	s.cronMu.Unlock()

	// 从数据库查询任务流
	var enabled bool
	var cronExpr string
	err := s.db.QueryRow("SELECT enabled, cron_expr FROM task_flows WHERE id=?", flowID).
		Scan(&enabled, &cronExpr)
	if err != nil {
		return fmt.Errorf("failed to query task flow %d: %v", flowID, err)
	}

	// 只有在启用且有有效 cron 表达式时才调度
	if enabled && cronExpr != "" {
		entryID, err := s.cron.AddFunc(cronExpr, func() {
			if err := s.RunTaskFlow(context.Background(), flowID); err != nil {
				log.Printf("scheduler: task flow %d error: %v", flowID, err)
			}
		})
		if err != nil {
			return fmt.Errorf("failed to schedule task flow %d: %v", flowID, err)
		}

		// 存储 cron 条目 ID
		s.cronMu.Lock()
		s.cronEntries[flowID] = entryID
		s.cronMu.Unlock()

		log.Printf("scheduler: reloaded task flow %d with cron expression: %s", flowID, cronExpr)
	}

	return nil
}

// RemoveTaskFlowFromCron 从cron调度中移除任务流（不kill正在运行的任务）
func (s *Scheduler) RemoveTaskFlowFromCron(flowID int) error {
	s.cronMu.Lock()
	defer s.cronMu.Unlock()

	entryID, exists := s.cronEntries[flowID]
	if exists {
		s.cron.Remove(entryID)
		delete(s.cronEntries, flowID)
		log.Printf("scheduler: removed task flow %d from cron scheduler", flowID)
		return nil
	}

	log.Printf("scheduler: task flow %d not found in cron scheduler", flowID)
	return nil
}

// ========== 任务执行方法 ==========

// RunTask 立即执行任务。它将任务状态更新为 'running'，
// 必要时生成 DataX 作业配置并启动 DataX 进程。
// 日志被捕获并存储在 task_logs 中。完成后，状态更新为 'success' 或 'failed'。
// 当通过 KillTask 取消上下文时，底层命令将被终止，状态标记为 'killed'。
func (s *Scheduler) RunTask(ctx context.Context, taskID int) (string, error) {
	return s.RunTaskWithDate(ctx, taskID, time.Time{})
}

// RunTaskWithDate 执行任务并支持指定日期占位符替换日期
// 如果 executionDate 为零值，则使用默认的昨天日期
func (s *Scheduler) RunTaskWithDate(ctx context.Context, taskID int, executionDate time.Time) (string, error) {
	// 获取详细信息：JSON 配置、源、目标
	var jsonCfg, name string
	var srcID, tgtID int
	err := s.db.QueryRow(`SELECT name, COALESCE(json_config,''), source_id, target_id FROM tasks WHERE id=?`, taskID).
		Scan(&name, &jsonCfg, &srcID, &tgtID)
	if err != nil {
		errorMsg := fmt.Sprintf("查询任务失败: %v", err)
		return errorMsg, err
	}

	// 如果配置为空则构建配置
	if jsonCfg == "" {
		errorMsg := "任务配置为空，无法执行"
		s.appendTaskLog(taskID, time.Now(), time.Now(), "failed", errorMsg)
		return errorMsg, fmt.Errorf("task %d has empty configuration", taskID)
	}

	// 处理日期占位符
	var processedConfig string
	if executionDate.IsZero() {
		processedConfig = processDatePlaceholders(jsonCfg)
	} else {
		processedConfig = processDatePlaceholders(jsonCfg, executionDate)
	}

	// 准备命令
	tmp := filepath.Join(s.tempDir, fmt.Sprintf("job_%d_%d.json", taskID, time.Now().UnixNano()))
	if err := os.WriteFile(tmp, []byte(processedConfig), 0644); err != nil {
		errorMsg := fmt.Sprintf("写入配置文件失败: %v", err)
		s.appendTaskLog(taskID, time.Now(), time.Now(), "failed", errorMsg)
		return errorMsg, err
	}

	// 使用带取消功能的上下文以支持终止
	jobCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(jobCtx, "python", filepath.Join(s.dataxHome, "bin", "datax.py"), tmp)

	// 原子性地检查和设置运行状态
	s.tasksMu.Lock()
	if _, exists := s.tasks[taskID]; exists {
		s.tasksMu.Unlock()
		os.Remove(tmp) // 清理临时文件

		errorMsg := fmt.Sprintf("任务 %d 正在运行中", taskID)
		return errorMsg, fmt.Errorf("task %d already running", taskID)
	}
	s.tasks[taskID] = &runningTask{cancel: cancel, cmd: cmd}
	s.tasksMu.Unlock()

	start := time.Now()
	output, err := cmd.CombinedOutput()
	end := time.Now()

	// 移除运行状态
	s.tasksMu.Lock()
	delete(s.tasks, taskID)
	s.tasksMu.Unlock()

	// 确定状态
	status := "success"
	if err != nil {
		// 如果上下文被取消，标记为已终止
		if jobCtx.Err() == context.Canceled {
			status = "killed"
		} else {
			status = "failed"
		}
	}

	// 保存日志
	s.appendTaskLog(taskID, start, end, status, string(output))

	// 清理临时文件
	os.Remove(tmp)
	return string(output), err
}

// KillTask 通过任务 ID 取消正在运行的任务。如果任务未运行，
// 则不会发生任何操作。终止后，状态将设置为 'killed' 并记录日志条目。
func (s *Scheduler) KillTask(taskID int) error {
	s.tasksMu.RLock()
	rt, ok := s.tasks[taskID]
	if !ok {
		s.tasksMu.RUnlock()
		return fmt.Errorf("task %d not running", taskID)
	}
	// 取消上下文；命令将退出
	// 注意：cancel() 是线程安全的，可以在读锁内调用
	rt.cancel()
	s.tasksMu.RUnlock()
	return nil
}

// ========== 任务流方法 ==========

// RunTaskFlow 立即执行任务流
func (s *Scheduler) RunTaskFlow(ctx context.Context, flowID int) error {
	// 创建执行记录
	result, err := s.db.Exec(`
		INSERT INTO task_flow_executions (flow_id, status, start_time)
		VALUES (?, 'running', NOW())
	`, flowID)
	if err != nil {
		return fmt.Errorf("failed to create task flow execution record: %v", err)
	}
	execID64, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get execution ID: %v", err)
	}
	execID := int(execID64)

	// 使用带取消功能的上下文以支持终止
	flowCtx, cancel := context.WithCancel(ctx)

	// 原子性地检查和设置运行状态
	s.flowsMu.Lock()
	if _, exists := s.flows[flowID]; exists {
		s.flowsMu.Unlock()
		cancel() // 清理上下文
		return fmt.Errorf("task flow %d already running", flowID)
	}
	s.flows[flowID] = &runningTaskFlow{cancel: cancel, flowID: flowID}
	s.flowsMu.Unlock()

	status := "success"
	var logContent string

	// 执行任务流步骤
	err = s.executeFlowSteps(flowCtx, flowID, execID)
	if err != nil {
		if flowCtx.Err() == context.Canceled {
			status = "killed"
			logContent = "Task flow was killed by user"
		} else {
			status = "failed"
			logContent = fmt.Sprintf("Task flow execution failed: %v", err)
		}
	} else {
		logContent = "Task flow completed successfully"
	}

	end := time.Now()

	// 移除运行状态
	s.flowsMu.Lock()
	delete(s.flows, flowID)
	s.flowsMu.Unlock()

	// 更新执行记录
	_, updateErr := s.db.Exec(`
		UPDATE task_flow_executions 
		SET status=?, end_time=?, log=?
		WHERE id=?
	`, status, end, logContent, execID)
	if updateErr != nil {
		log.Printf("scheduler: failed to update task flow execution final status: %v", updateErr)
	}

	return err
}

// KillTaskFlow 取消正在运行的任务流
func (s *Scheduler) KillTaskFlow(flowID int) error {
	s.flowsMu.RLock()
	rt, ok := s.flows[flowID]
	if !ok {
		s.flowsMu.RUnlock()
		return fmt.Errorf("task flow %d not running", flowID)
	}
	// 取消上下文；执行将退出
	// 注意：cancel() 是线程安全的，可以在读锁内调用
	rt.cancel()
	s.flowsMu.RUnlock()
	return nil
}

// ========== 任务流步骤执行 ==========

// StepInfo 表示任务流步骤
type StepInfo struct {
	ID             int
	TaskID         int
	TimeoutMinutes *int
	TaskName       string
	JSONConfig     string
}

// executeFlowSteps 执行任务流中的所有步骤
func (s *Scheduler) executeFlowSteps(ctx context.Context, flowID, execID int) error {
	// 按 step_order 获取任务流步骤
	rows, err := s.db.Query(`
		SELECT s.id, s.task_id, s.timeout_minutes, t.name, t.json_config
		FROM task_flow_steps s
		JOIN tasks t ON s.task_id = t.id
		WHERE s.flow_id = ?
		ORDER BY s.step_order
	`, flowID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var steps []StepInfo

	for rows.Next() {
		var step StepInfo
		rows.Scan(&step.ID, &step.TaskID, &step.TimeoutMinutes, &step.TaskName, &step.JSONConfig)
		steps = append(steps, step)
	}

	// 按顺序执行所有步骤
	for i, step := range steps {
		// 创建步骤执行记录
		result, err := s.db.Exec(`
			INSERT INTO task_flow_step_executions (execution_id, step_id, task_id, status)
			VALUES (?, ?, ?, ?)
		`, execID, step.ID, step.TaskID, "pending")
		if err != nil {
			return fmt.Errorf("failed to create step execution record for step %d: %v", i+1, err)
		}
		stepExecID64, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get step execution ID for step %d: %v", i+1, err)
		}
		stepExecID := int(stepExecID64)

		// 执行步骤
		stepSuccess, err := s.executeStep(ctx, step, stepExecID)
		if err != nil {
			return fmt.Errorf("step %d (%s) failed: %v", i+1, step.TaskName, err)
		}
		if !stepSuccess {
			return fmt.Errorf("step %d (%s) failed", i+1, step.TaskName)
		}
	}

	return nil
}

// executeStep 执行单个步骤
func (s *Scheduler) executeStep(ctx context.Context, step StepInfo, stepExecID int) (bool, error) {
	// 将步骤状态更新为运行中
	_, err := s.db.Exec("UPDATE task_flow_step_executions SET status='running', start_time=NOW() WHERE id=?", stepExecID)
	if err != nil {
		log.Printf("scheduler: failed to update step execution status to running: %v", err)
		// 继续执行，不因为状态更新失败而停止
	}

	// 如果指定了超时时间则创建带超时的上下文
	stepCtx := ctx
	var cancel context.CancelFunc
	if step.TimeoutMinutes != nil && *step.TimeoutMinutes > 0 {
		stepCtx, cancel = context.WithTimeout(ctx, time.Duration(*step.TimeoutMinutes)*time.Minute)
		defer cancel()
	}

	// 使用任务执行方法执行任务
	output, err := s.RunTask(stepCtx, step.TaskID)

	// 确定成功状态
	success := err == nil
	status := "success"

	if err != nil {
		if stepCtx.Err() == context.Canceled {
			status = "killed"
		} else {
			status = "failed"
		}
	}

	// 更新步骤执行记录，直接使用返回的输出
	_, updateErr := s.db.Exec(`
		UPDATE task_flow_step_executions 
		SET status=?, end_time=NOW(), log=?
		WHERE id=?
	`, status, output, stepExecID)
	if updateErr != nil {
		log.Printf("scheduler: failed to update step execution final status: %v", updateErr)
	}

	return success, err
}

// ========== 辅助方法 ==========

// processDatePlaceholders 处理配置中的日期占位符
// 默认替换为执行日期前一天，但支持传入自定义日期
func processDatePlaceholders(config string, executionDate ...time.Time) string {
	var targetDate time.Time
	if len(executionDate) > 0 {
		targetDate = executionDate[0]
	} else {
		// 默认使用执行日期前一天
		targetDate = time.Now().AddDate(0, 0, -1)
	}

	// 支持的日期占位符格式
	placeholders := map[string]string{
		"${yyyy-mm-dd}": targetDate.Format("2006-01-02"),
		"${yyyy_mm_dd}": targetDate.Format("2006_01_02"),
	}

	result := config
	for placeholder, value := range placeholders {
		result = regexp.MustCompile(regexp.QuoteMeta(placeholder)).ReplaceAllString(result, value)
	}

	return result
}

// appendTaskLog 为任务插入日志条目
func (s *Scheduler) appendTaskLog(taskID int, start, end time.Time, status, text string) {
	_, err := s.db.Exec("INSERT INTO task_logs(task_id,start_time,end_time,status,log) VALUES(?,?,?,?,?)", taskID, start, end, status, text)
	if err != nil {
		log.Printf("scheduler: failed to append task log for task %d: %v", taskID, err)
	}
}

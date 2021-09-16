package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.TaskCheckRunService = (*TaskCheckRunService)(nil)
)

// TaskCheckRunService represents a service for managing taskCheckRun.
type TaskCheckRunService struct {
	l  *zap.Logger
	db *DB
}

// newTaskCheckRunService returns a new TaskCheckRunService.
func NewTaskCheckRunService(logger *zap.Logger, db *DB) *TaskCheckRunService {
	return &TaskCheckRunService{l: logger, db: db}
}

// CreateTaskCheckRun creates a new taskCheckRun. See interface for the expected behavior
func (s *TaskCheckRunService) CreateTaskCheckRunIfNeeded(ctx context.Context, create *api.TaskCheckRunCreate) (*api.TaskCheckRun, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	statusList := []api.TaskCheckRunStatus{api.TaskCheckRunRunning}
	if create.SkipIfAlreadyDone {
		statusList = append(statusList, api.TaskCheckRunDone)
	}
	taskCheckRunFind := &api.TaskCheckRunFind{
		TaskId:     &create.TaskId,
		StatusList: &statusList,
	}

	taskCheckRunList, err := s.FindTaskCheckRunListTx(ctx, tx.Tx, taskCheckRunFind)
	if err != nil {
		return nil, err
	}

	runningCount := 0
	if create.SkipIfAlreadyDone {
		for _, taskCheckRun := range taskCheckRunList {
			if taskCheckRun.Status == api.TaskCheckRunDone {
				return taskCheckRun, nil
			} else if taskCheckRun.Status == api.TaskCheckRunRunning {
				runningCount++
			}
		}
	} else {
		runningCount = len(taskCheckRunList)
	}

	if runningCount > 0 {
		if runningCount > 1 {
			// Normally, this should not happen, if it occurs, emit a warning
			s.l.Warn(fmt.Sprintf("Found %d task check run, expect at most 1", len(taskCheckRunList)),
				zap.Int("task_id", create.TaskId),
				zap.String("task_check_type", string(create.Type)),
			)
		}
		return taskCheckRunList[0], nil
	}

	taskCheckRun, err := s.CreateTaskCheckRunTx(ctx, tx.Tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return taskCheckRun, nil
}

// CreateTaskCheckRunTx creates a new taskCheckRun.
func (s *TaskCheckRunService) CreateTaskCheckRunTx(ctx context.Context, tx *sql.Tx, create *api.TaskCheckRunCreate) (*api.TaskCheckRun, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO task_check_run (
			creator_id,
			updater_id,
			task_id,
			name,
			`+"`status`,"+`
			`+"`type`,"+`
			comment,
			payload
		)
		VALUES (?, ?, ?, ?, 'RUNNING', ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, `+"`status`, `type`, comment, result, payload"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.TaskId,
		create.Name,
		create.Type,
		create.Comment,
		create.Payload,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var taskCheckRun api.TaskCheckRun
	if err := row.Scan(
		&taskCheckRun.ID,
		&taskCheckRun.CreatorId,
		&taskCheckRun.CreatedTs,
		&taskCheckRun.UpdaterId,
		&taskCheckRun.UpdatedTs,
		&taskCheckRun.TaskId,
		&taskCheckRun.Name,
		&taskCheckRun.Status,
		&taskCheckRun.Type,
		&taskCheckRun.Comment,
		&taskCheckRun.Result,
		&taskCheckRun.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &taskCheckRun, nil
}

// FindTaskCheckRunList retrieves a list of taskCheckRuns based on find.
func (s *TaskCheckRunService) FindTaskCheckRunList(ctx context.Context, find *api.TaskCheckRunFind) ([]*api.TaskCheckRun, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findTaskCheckRunList(ctx, tx.Tx, find)
	if err != nil {
		return []*api.TaskCheckRun{}, err
	}

	return list, nil
}

// FindTaskCheckRunListTx retrieves a list of taskCheckRuns based on find.
func (s *TaskCheckRunService) FindTaskCheckRunListTx(ctx context.Context, tx *sql.Tx, find *api.TaskCheckRunFind) ([]*api.TaskCheckRun, error) {
	list, err := s.findTaskCheckRunList(ctx, tx, find)
	if err != nil {
		return []*api.TaskCheckRun{}, err
	}

	return list, nil
}

// FindTaskCheckRunTx retrieves a single taskCheckRun based on find.
// Returns ENOTFOUND if no matching record.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *TaskCheckRunService) FindTaskCheckRunTx(ctx context.Context, tx *sql.Tx, find *api.TaskCheckRunFind) (*api.TaskCheckRun, error) {
	list, err := s.findTaskCheckRunList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &common.Error{Code: common.ENOTFOUND, Message: fmt.Sprintf("task run not found: %+v", find)}
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.ECONFLICT, Message: fmt.Sprintf("found %d task runs with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchTaskCheckRunStatus updates a taskCheckRun status. Returns the new state of the taskCheckRun after update.
func (s *TaskCheckRunService) PatchTaskCheckRunStatus(ctx context.Context, patch *api.TaskCheckRunStatusPatch) (*api.TaskCheckRun, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	taskCheckRun, err := s.PatchTaskCheckRunStatusTx(ctx, tx.Tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return taskCheckRun, nil
}

// PatchTaskCheckRunStatusTx updates a taskCheckRun status. Returns the new state of the taskCheckRun after update.
func (s *TaskCheckRunService) PatchTaskCheckRunStatusTx(ctx context.Context, tx *sql.Tx, patch *api.TaskCheckRunStatusPatch) (*api.TaskCheckRun, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	set, args = append(set, "`status` = ?"), append(args, patch.Status)
	set, args = append(set, "comment = ?"), append(args, patch.Comment)
	set, args = append(set, "result = ?"), append(args, patch.Result)

	// Build WHERE clause.
	where := []string{"1 = 1"}
	if v := patch.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}

	row, err := tx.QueryContext(ctx, `
		UPDATE task_check_run
		SET `+strings.Join(set, ", ")+`
		WHERE `+strings.Join(where, " AND ")+`
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, task_id, name, `+"`status`, `type`, comment, result, payload"+`
	`,
		args...,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var taskCheckRun api.TaskCheckRun
	if err := row.Scan(
		&taskCheckRun.ID,
		&taskCheckRun.CreatorId,
		&taskCheckRun.CreatedTs,
		&taskCheckRun.UpdaterId,
		&taskCheckRun.UpdatedTs,
		&taskCheckRun.TaskId,
		&taskCheckRun.Name,
		&taskCheckRun.Status,
		&taskCheckRun.Type,
		&taskCheckRun.Comment,
		&taskCheckRun.Result,
		&taskCheckRun.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	return &taskCheckRun, nil
}

func (s *TaskCheckRunService) findTaskCheckRunList(ctx context.Context, tx *sql.Tx, find *api.TaskCheckRunFind) (_ []*api.TaskCheckRun, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.TaskId; v != nil {
		where, args = append(where, "task_id = ?"), append(args, *v)
	}
	if v := find.StatusList; v != nil {
		list := []string{}
		for _, status := range *v {
			list = append(list, "?")
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("`status` in (%s)", strings.Join(list, ",")))
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
		    created_ts,
			updater_id,
		    updated_ts,
			task_id,
			name,
			`+"`status`,"+`
			`+"`type`,"+`
			comment,
			result,
			payload
		FROM task_check_run
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.TaskCheckRun, 0)
	for rows.Next() {
		var taskCheckRun api.TaskCheckRun
		if err := rows.Scan(
			&taskCheckRun.ID,
			&taskCheckRun.CreatorId,
			&taskCheckRun.CreatedTs,
			&taskCheckRun.UpdaterId,
			&taskCheckRun.UpdatedTs,
			&taskCheckRun.TaskId,
			&taskCheckRun.Name,
			&taskCheckRun.Status,
			&taskCheckRun.Type,
			&taskCheckRun.Comment,
			&taskCheckRun.Result,
			&taskCheckRun.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &taskCheckRun)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}
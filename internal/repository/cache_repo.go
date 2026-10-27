package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ivancabzon/task-service/internal/model"
)

const taskListTTL = 5 * time.Minute

type CacheRepo struct {
	rdb *redis.Client
}

func NewCacheRepo(rdb *redis.Client) *CacheRepo {
	return &CacheRepo{rdb: rdb}
}

// GetTasks — достаёт закешированный список. ok=false при промахе.
func (c *CacheRepo) GetTasks(ctx context.Context, f model.TaskFilter) (tasks []model.Task, ok bool, err error) {
	key := buildTaskListKey(f)
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis get: %w", err)
	}
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, false, nil
	}
	return tasks, true, nil
}

func (c *CacheRepo) SetTasks(ctx context.Context, f model.TaskFilter, tasks []model.Task) error {
	key := buildTaskListKey(f)
	data, err := json.Marshal(tasks)
	if err != nil {
		return fmt.Errorf("marshal tasks: %w", err)
	}
	if err := c.rdb.Set(ctx, key, data, taskListTTL).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func (c *CacheRepo) InvalidateTeamTasks(ctx context.Context, teamID int64) error {
	pattern := fmt.Sprintf("tasks:team:%d:*", teamID)

	var cursor uint64
	for {
		keys, next, err := c.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("redis scan: %w", err)
		}
		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("redis del: %w", err)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

func buildTaskListKey(f model.TaskFilter) string {
	status := "*"
	if f.Status != "" {
		status = f.Status
	}
	assignee := "*"
	if f.AssigneeID != nil {
		assignee = fmt.Sprintf("%d", *f.AssigneeID)
	}
	return fmt.Sprintf("tasks:team:%d:status:%s:assignee:%s:limit:%d:offset:%d",
		f.TeamID, status, assignee, f.Limit, f.Offset)
}

func buildTaskListKeyFromMap(teamID int64, filters map[string]any) string {
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	fmt.Fprintf(&b, "tasks:team:%d", teamID)
	for _, k := range keys {
		fmt.Fprintf(&b, ":%s:%v", k, filters[k])
	}
	return b.String()
}

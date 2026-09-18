package scheduler

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"quick-feishu/internal/config"
)

type Scheduler struct {
	Cron       *cron.Cron
	OnSnapshot func()
	OnReport   func()
}

func New(cfg *config.Config, loc *time.Location) *Scheduler {
	return &Scheduler{
		Cron: cron.New(cron.WithLocation(loc), cron.WithSeconds()),
	}
}

// RegisterSnapshot 注册每日快照任务（6 段表达式，含秒）
func (s *Scheduler) RegisterSnapshot(cronExpr string) error {
	_, err := s.Cron.AddFunc(cronExpr, func() {
		if s.OnSnapshot != nil {
			s.OnSnapshot()
		}
	})
	return err
}

// RegisterReport 注册每日日报任务（6 段表达式，含秒）
func (s *Scheduler) RegisterReport(cronExpr string) error {
	_, err := s.Cron.AddFunc(cronExpr, func() {
		if s.OnReport != nil {
			s.OnReport()
		}
	})
	return err
}

func (s *Scheduler) Start() { s.Cron.Start() }

func (s *Scheduler) Stop() { s.Cron.Stop() }

// TimeToCron 将 "HH:MM" 转为 6 段 cron 表达式（秒 分 时 日 月 周）
func TimeToCron(t string) (string, error) {
	return timeToCron(t)
}

// timeToCron 将 "HH:MM" 转为 cron 表达式
func timeToCron(t string) (string, error) {
	parsed, err := time.Parse("15:04", t)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0 %d %d * * *", parsed.Minute(), parsed.Hour()), nil
}

package app

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
	"quick-feishu/internal/api"
	"quick-feishu/internal/collector"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
	"quick-feishu/internal/report"
	"quick-feishu/internal/scheduler"
)

// loc 固定为北京时间（UTC+8）
var loc = time.FixedZone("CST", 8*3600)

// Today 返回北京时间的当天日期 YYYY-MM-DD
func Today() string { return time.Now().In(loc).Format("2006-01-02") }

// App 管理运行时状态：数据库、配置、API 客户端与定时任务。
// API 客户端可在账号配置变更后重建；定时任务可重启以应用新配置。
type App struct {
	mu         sync.Mutex
	DB         *gorm.DB
	Config     *config.Config
	ConfigPath string

	client *api.Client
	sched  *scheduler.Scheduler
}

func New(cfg *config.Config, gdb *gorm.DB, configPath string) *App {
	a := &App{DB: gdb, Config: cfg, ConfigPath: configPath}
	a.client = api.NewClient(cfg.Account.APIBase, cfg.Account.SystemToken, cfg.Account.UserID)
	return a
}

// Client 返回当前 API 客户端（可能已被重建）
func (a *App) Client() *api.Client {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.client
}

// RunSnapshot 采集并新增快照，保留同一天的每次采集记录。
func (a *App) RunSnapshot() (*collector.Result, error) {
	res := collector.Collect(a.Client())
	if err := collector.Save(a.DB, Today(), res); err != nil {
		return res, err
	}
	return res, nil
}

// RunReport 取最近两日快照生成并发送日报。
// 无快照时返回 (nil, error)；发送失败时返回 (log, error)。
func (a *App) RunReport() (*model.SendLog, error) {
	latest, err := db.LatestSnapshot(a.DB)
	if err != nil || latest == nil {
		return nil, fmt.Errorf("no snapshots yet")
	}
	var prev *model.Snapshot
	if p, e := db.SnapshotBefore(a.DB, addDays(latest.SnapshotDate, -1)); e == nil {
		prev = p
	}
	tmpl, _ := report.TemplateFromMap(a.Config.ReportTemplate)
	if tmpl == nil {
		tmpl = report.DefaultTemplate()
	}
	return report.ExecuteReport(a.DB, latest, prev, tmpl, a.Config.Feishu.WebhookURL, a.Config.Feishu.RetryTimes)
}

// Backfill 启动补采缺失的历史快照
func (a *App) Backfill() error {
	return scheduler.BackfillMissing(a.DB, a.Client(), Today(), func() *collector.Result {
		return collector.Collect(a.Client())
	})
}

// SaveConfig 保存设置（尊重环境变量覆盖），用户从页面保存时调用
func (a *App) SaveConfig(in *config.Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	env := config.EnvOverridden()
	if env["user_id"] || in.Account.UserID == "" {
		in.Account.UserID = a.Config.Account.UserID
	}
	if env["system_token"] || in.Account.SystemToken == "" {
		in.Account.SystemToken = a.Config.Account.SystemToken
	}
	if env["webhook_url"] || in.Feishu.WebhookURL == "" {
		in.Feishu.WebhookURL = a.Config.Feishu.WebhookURL
	}
	if in.Account.APIBase == "" {
		in.Account.APIBase = a.Config.Account.APIBase
	}
	*a.Config = *in
	return a.Config.Save(a.ConfigPath)
}

// SaveTemplate 保存日报模板
func (a *App) SaveTemplate(tmpl map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Config.ReportTemplate = tmpl
	return a.Config.Save(a.ConfigPath)
}

// StartScheduler 启动定时任务（首次启动调用）
func (a *App) StartScheduler() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.startLocked()
}

// Restart 重新从磁盘加载配置、重建 API 客户端并重启定时任务。
// 用于账号/令牌/时间变更后生效。
func (a *App) Restart() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if cfg, err := config.Load(a.ConfigPath); err == nil {
		*a.Config = *cfg
	}
	a.client = api.NewClient(a.Config.Account.APIBase, a.Config.Account.SystemToken, a.Config.Account.UserID)
	return a.startLocked()
}

// NextRuns 返回各定时任务的下次执行时间
func (a *App) NextRuns() []time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sched == nil {
		return nil
	}
	var out []time.Time
	for _, e := range a.sched.Cron.Entries() {
		out = append(out, e.Next)
	}
	return out
}

func (a *App) startLocked() error {
	if a.sched != nil {
		a.sched.Stop()
		a.sched = nil
	}
	sch := scheduler.New(a.Config, loc)
	sch.OnSnapshot = func() {
		if res, err := a.RunSnapshot(); err != nil {
			fmt.Println("snapshot error:", err)
		} else {
			fmt.Println("snapshot saved:", Today())
			for _, e := range res.Issues {
				fmt.Printf("  collect issue: kind=%s scope=%s token=%s status=%d detail=%s\n", e.Kind, e.Scope, e.TokenName, e.Status, e.Detail)
			}
		}
	}
	sch.OnReport = func() {
		log, err := a.RunReport()
		if err != nil {
			fmt.Println("report error:", err)
		} else {
			fmt.Printf("report sent, log id %d\n", log.ID)
		}
	}
	snapExpr, err := scheduler.TimeToCron(a.Config.Schedule.SnapshotTime)
	if err != nil {
		return fmt.Errorf("bad snapshot time: %w", err)
	}
	reportExpr, err := scheduler.TimeToCron(a.Config.Schedule.ReportTime)
	if err != nil {
		return fmt.Errorf("bad report time: %w", err)
	}
	if err := sch.RegisterSnapshot(snapExpr); err != nil {
		return err
	}
	if err := sch.RegisterReport(reportExpr); err != nil {
		return err
	}
	sch.Start()
	a.sched = sch
	return nil
}

func addDays(date string, n int) string {
	d, _ := time.Parse("2006-01-02", date)
	return d.AddDate(0, 0, n).Format("2006-01-02")
}

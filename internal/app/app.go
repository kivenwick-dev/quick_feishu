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
	mu            sync.Mutex
	db            *gorm.DB
	Config        *config.Config
	ConfigPath    string
	dataDir       string
	currentDBPath string

	client *api.Client
	sched  *scheduler.Scheduler
}

// openAccountDB 迁移（首次）并打开指定账号库；任何一步失败都会关闭已打开的连接。
func openAccountDB(dataDir, userID string) (*gorm.DB, string, error) {
	if err := db.MigrateLegacy(dataDir, userID); err != nil {
		return nil, "", err
	}
	path := db.Path(dataDir, userID)
	gdb, err := db.Open(path)
	if err != nil {
		return nil, "", err
	}
	if err := db.SeedDicts(gdb); err != nil {
		if sqlDB, cerr := gdb.DB(); cerr == nil {
			_ = sqlDB.Close()
		}
		return nil, "", err
	}
	return gdb, path, nil
}

// New 按 cfg.Account.UserID 迁移（首次）并打开对应账号库，返回 App。
func New(cfg *config.Config, dataDir, configPath string) (*App, error) {
	gdb, path, err := openAccountDB(dataDir, cfg.Account.UserID)
	if err != nil {
		return nil, err
	}
	a := &App{
		db:            gdb,
		Config:        cfg,
		ConfigPath:    configPath,
		dataDir:       dataDir,
		currentDBPath: path,
	}
	a.client = api.NewClient(cfg.Account.APIBase, cfg.Account.SystemToken, cfg.Account.UserID)
	return a, nil
}

// DB 返回当前账号库（切库后指向新库）。
func (a *App) DB() *gorm.DB {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.db
}

// Client 返回当前 API 客户端（可能已被重建）
func (a *App) Client() *api.Client {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.client
}

// conn 在单次加锁下返回当前客户端与数据库，避免操作中途切库导致跨账号串写。
func (a *App) conn() (*api.Client, *gorm.DB) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.client, a.db
}

// ConfigSnapshot 在锁内返回当前配置的副本，供只读使用，
// 避免与切库/保存配置并发读写同一 config.Config。
func (a *App) ConfigSnapshot() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return *a.Config
}

// reportDeps 是生成日报所需的、在单次加锁下取得的依赖快照。
type reportDeps struct {
	client     *api.Client
	gdb        *gorm.DB
	template   map[string]interface{}
	webhookURL string
	retryTimes int
}

// reportDeps 单次加锁返回数据库与日报配置快照，避免切库时把旧账号数据发到新账号 webhook。
func (a *App) reportDeps() reportDeps {
	a.mu.Lock()
	defer a.mu.Unlock()
	return reportDeps{
		client:     a.client,
		gdb:        a.db,
		template:   a.Config.ReportTemplate,
		webhookURL: a.Config.Feishu.WebhookURL,
		retryTimes: a.Config.Feishu.RetryTimes,
	}
}

// RunSnapshot 采集并新增快照，保留同一天的每次采集记录。
func (a *App) RunSnapshot() (*collector.Result, error) {
	client, gdb := a.conn()
	res := collector.Collect(client)
	if err := collector.SaveForAccount(gdb, Today(), res, client.UserID); err != nil {
		return res, err
	}
	return res, nil
}

// RunReport 先采集当前快照，再与上一条快照对比生成并发送日报。
// 无快照时返回 (nil, error)；发送失败时返回 (log, error)。
func (a *App) RunReport() (*model.SendLog, error) {
	deps := a.reportDeps()
	res := collector.Collect(deps.client)
	if err := collector.SaveForAccount(deps.gdb, Today(), res, deps.client.UserID); err != nil {
		return nil, err
	}
	latest, err := db.LatestSnapshot(deps.gdb)
	if err != nil || latest == nil {
		return nil, fmt.Errorf("no snapshots yet")
	}
	var prev *model.Snapshot
	if p, e := db.PreviousSnapshot(deps.gdb, latest); e == nil {
		prev = p
	}
	tmpl, _ := report.TemplateFromMap(deps.template)
	if tmpl == nil {
		tmpl = report.DefaultTemplate()
	}
	return report.ExecuteReport(deps.gdb, latest, prev, tmpl, deps.webhookURL, deps.retryTimes)
}

// Backfill 启动补采缺失的历史快照
func (a *App) Backfill() error {
	client, gdb := a.conn()
	return scheduler.BackfillMissing(gdb, client, Today(), func() *collector.Result {
		return collector.Collect(client)
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

// Restart 重载配置、按账号切库（如有变化）、重建 API 客户端并重启定时任务。
// 立即采集由调用方在 Restart 成功后执行，避免首次切到空库时 Backfill 与立即采集重复写入。
func (a *App) Restart() error {
	a.mu.Lock()
	_, err := a.restartLocked()
	a.mu.Unlock()
	return err
}

func (a *App) restartLocked() (bool, error) {
	if cfg, err := config.Load(a.ConfigPath); err == nil {
		*a.Config = *cfg
	}
	switched := false
	next := db.Path(a.dataDir, a.Config.Account.UserID)
	if next != a.currentDBPath {
		gdb, path, err := openAccountDB(a.dataDir, a.Config.Account.UserID)
		if err != nil {
			return false, err
		}
		if sqlDB, cerr := a.db.DB(); cerr == nil {
			_ = sqlDB.Close()
		}
		a.db = gdb
		a.currentDBPath = path
		switched = true
	}
	a.client = api.NewClient(a.Config.Account.APIBase, a.Config.Account.SystemToken, a.Config.Account.UserID)
	if err := a.startLocked(); err != nil {
		return switched, err
	}
	return switched, nil
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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"quick-feishu/internal/api"
	"quick-feishu/internal/collector"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
	"quick-feishu/internal/report"
	"quick-feishu/internal/scheduler"
	"quick-feishu/internal/server"
)

var loc = time.FixedZone("CST", 8*3600)

func today() string { return time.Now().In(loc).Format("2006-01-02") }

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"serve"}
	}
	switch args[0] {
	case "serve":
		serve()
	case "run-report":
		runReport()
	case "snapshot":
		runSnapshot()
	default:
		fmt.Println("usage: quick-feishu [serve|run-report|snapshot]")
		os.Exit(1)
	}
}

func baseDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	return filepath.Dir(exe)
}

func serve() {
	dir := baseDir()
	cfg, err := config.Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		fmt.Println("config error:", err)
		os.Exit(1)
	}
	gdb, err := db.Init(filepath.Join(dir, "data"))
	if err != nil {
		fmt.Println("db error:", err)
		os.Exit(1)
	}
	if err := db.SeedDicts(gdb); err != nil {
		fmt.Println("seed error:", err)
		os.Exit(1)
	}
	c := api.NewClient(cfg.Account.APIBase, cfg.Account.SystemToken, cfg.Account.UserID)

	// 启动补采
	if err := scheduler.BackfillMissing(gdb, c, today(), func() *collector.Result { return collector.Collect(c) }); err != nil {
		fmt.Println("backfill error:", err)
	}

	sch := scheduler.New(cfg, loc)
	sch.OnSnapshot = func() {
		res := collector.Collect(c)
		if err := collector.Save(gdb, today(), res); err != nil {
			fmt.Println("snapshot error:", err)
		} else {
			fmt.Println("snapshot saved:", today())
		}
	}
	sch.OnReport = func() {
		latest, _ := db.LatestSnapshot(gdb)
		if latest == nil {
			fmt.Println("no snapshots for report")
			return
		}
		var prev *model.Snapshot
		if p, perr := db.SnapshotBefore(gdb, addDays(latest.SnapshotDate, -1)); perr == nil {
			prev = p
		}
		tmpl, _ := report.TemplateFromMap(cfg.ReportTemplate)
		if tmpl == nil {
			tmpl = report.DefaultTemplate()
		}
		log, err := report.ExecuteReport(gdb, latest, prev, tmpl, cfg.Feishu.WebhookURL, cfg.Feishu.RetryTimes)
		if err != nil {
			fmt.Println("report error:", err)
		} else {
			fmt.Printf("report sent, log id %d\n", log.ID)
		}
	}

	snapExpr, err := scheduler.TimeToCron(cfg.Schedule.SnapshotTime)
	if err != nil {
		fmt.Println("bad snapshot time:", err)
		os.Exit(1)
	}
	reportExpr, err := scheduler.TimeToCron(cfg.Schedule.ReportTime)
	if err != nil {
		fmt.Println("bad report time:", err)
		os.Exit(1)
	}
	if err := sch.RegisterSnapshot(snapExpr); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := sch.RegisterReport(reportExpr); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	sch.Start()

	srv := server.New(cfg.App.Port)
	port, err := srv.Listen()
	if err != nil {
		fmt.Println("port error:", err)
		os.Exit(1)
	}
	handlers := &server.Handlers{DB: gdb, Config: cfg, ConfigPath: filepath.Join(dir, "config.yaml"), APIClient: c}
	srv.RegisterRoutes(handlers)
	url := fmt.Sprintf("http://localhost:%d/", port)
	fmt.Println("QuickFeishu running at", url)
	_ = server.OpenBrowser(url)
	if err := srv.Run(port); err != nil {
		fmt.Println("server error:", err)
		os.Exit(1)
	}
}

func runSnapshot() {
	dir := baseDir()
	cfg, err := config.Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		fmt.Println("config error:", err)
		os.Exit(1)
	}
	gdb, err := db.Init(filepath.Join(dir, "data"))
	if err != nil {
		fmt.Println("db error:", err)
		os.Exit(1)
	}
	c := api.NewClient(cfg.Account.APIBase, cfg.Account.SystemToken, cfg.Account.UserID)
	res := collector.Collect(c)
	if err := collector.Save(gdb, today(), res); err != nil {
		fmt.Println("save error:", err)
		os.Exit(1)
	}
	fmt.Println("snapshot saved:", today())
}

func runReport() {
	dir := baseDir()
	cfg, err := config.Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		fmt.Println("config error:", err)
		os.Exit(1)
	}
	gdb, err := db.Init(filepath.Join(dir, "data"))
	if err != nil {
		fmt.Println("db error:", err)
		os.Exit(1)
	}
	latest, _ := db.LatestSnapshot(gdb)
	if latest == nil {
		fmt.Println("no snapshots yet, run 'snapshot' first")
		os.Exit(1)
	}
	var prev *model.Snapshot
	if p, perr := db.SnapshotBefore(gdb, addDays(latest.SnapshotDate, -1)); perr == nil {
		prev = p
	}
	tmpl, _ := report.TemplateFromMap(cfg.ReportTemplate)
	if tmpl == nil {
		tmpl = report.DefaultTemplate()
	}
	log, err := report.ExecuteReport(gdb, latest, prev, tmpl, cfg.Feishu.WebhookURL, cfg.Feishu.RetryTimes)
	if err != nil {
		fmt.Println("report failed:", err)
		os.Exit(1)
	}
	fmt.Println("report sent, log id:", log.ID)
}

func addDays(date string, n int) string {
	d, _ := time.Parse("2006-01-02", date)
	return d.AddDate(0, 0, n).Format("2006-01-02")
}

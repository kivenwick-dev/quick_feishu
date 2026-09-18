package main

import (
	"fmt"
	"os"
	"path/filepath"

	"quick-feishu/internal/app"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/server"
)

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

func newApp() (*app.App, error) {
	dir := baseDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}
	gdb, err := db.Init(filepath.Join(dir, "data"))
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	if err := db.SeedDicts(gdb); err != nil {
		return nil, fmt.Errorf("seed error: %w", err)
	}
	return app.New(cfg, gdb, cfgPath), nil
}

func serve() {
	a, err := newApp()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// 启动补采
	if err := a.Backfill(); err != nil {
		fmt.Println("backfill error:", err)
	}
	// 启动定时任务
	if err := a.StartScheduler(); err != nil {
		fmt.Println("scheduler error:", err)
		os.Exit(1)
	}

	srv := server.New(a.Config.App.Port)
	port, err := srv.Listen()
	if err != nil {
		fmt.Println("port error:", err)
		os.Exit(1)
	}
	handlers := &server.Handlers{App: a, DB: a.DB, Config: a.Config, ConfigPath: a.ConfigPath}
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
	a, err := newApp()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	res, err := a.RunSnapshot()
	if err != nil {
		fmt.Println("save error:", err)
		os.Exit(1)
	}
	fmt.Println("snapshot saved:", app.Today())
	for _, e := range res.Issues {
		fmt.Printf("  issue: kind=%s scope=%s token=%s status=%d detail=%s\n", e.Kind, e.Scope, e.TokenName, e.Status, e.Detail)
	}
}

func runReport() {
	a, err := newApp()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log, err := a.RunReport()
	if err != nil {
		fmt.Println("report failed:", err)
		os.Exit(1)
	}
	fmt.Println("report sent, log id:", log.ID)
}

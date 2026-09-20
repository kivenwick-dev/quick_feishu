package main

import (
	"fmt"
	"os"
	"path/filepath"

	"quick-feishu/internal/app"
	"quick-feishu/internal/config"
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

// resolveDataDir 返回数据库等运行数据目录：优先 QF_DATA_DIR（容器里通常指向挂载卷），
// 否则回退到可执行文件目录下的 data/。
func resolveDataDir() string {
	if dir := os.Getenv("QF_DATA_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(baseDir(), "data")
}

// resolveConfigPath 返回配置文件路径：优先 QF_CONFIG_FILE（只读根文件系统时挂载单个文件），
// 否则回退到可执行文件目录下的 config.yaml。
func resolveConfigPath() string {
	if path := os.Getenv("QF_CONFIG_FILE"); path != "" {
		return path
	}
	return filepath.Join(baseDir(), "config.yaml")
}

func newApp() (*app.App, error) {
	cfgPath := resolveConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}
	a, err := app.New(cfg, resolveDataDir(), cfgPath)
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	return a, nil
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
	handlers := &server.Handlers{App: a}
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

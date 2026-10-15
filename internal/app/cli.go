package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func urlQuery(s string) string { return url.QueryEscape(s) }
func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return err
}
func Run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Print(`Memvault · 本地会话与记忆资产库

  setup      初始化记忆目录，检测来源并配置自动运行
  status     查看归档、来源和接入状态
  open       打开本地网页
  search     搜索会话与记忆
  show ID    阅读完整会话（--format markdown 可输出可读原文）
  resume ID  用指定 agent 接续（--agent codex|claude|opencode）
  export     导出自包含档案（--out archive.zip）
  import ZIP 导入带校验清单的档案
  repair     检查或重建（--action verify|rebuild|integrations|scan）
  uninstall  移除本机接入与登录启动，保留所有档案

  setup --root <目录> --yes       使用指定目录并完成配置
  serve                          前台运行服务
  mcp                            MCP 标准输入输出入口
  version                        程序版本
  licenses                       第三方许可证
`)
		return nil
	}
	command := args[0]
	rest := args[1:]
	if command == "licenses" {
		fmt.Print(thirdPartyNotices)
		return nil
	}
	if command == "version" {
		fmt.Println(Version)
		return nil
	}
	if command == "setup" {
		return setup(rest)
	}
	if command == "hook" {
		flags := flag.NewFlagSet("hook", flag.ContinueOnError)
		agent := flags.String("agent", "", "来源")
		_ = flags.String("integration", "", "")
		if err := flags.Parse(rest); err != nil {
			return err
		}
		return Hook(*agent, os.Stdin)
	}
	c, err := loadConfig()
	if err != nil {
		return fail("尚未配置，先运行 memvault setup：%w", err)
	}
	if command == "serve" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		return Serve(ctx, c)
	}
	if command == "mcp" {
		return MCP(c, os.Stdin, os.Stdout)
	}
	if command == "uninstall" {
		items, err := Configure(c, true)
		_, _ = request(context.Background(), c, "POST", "shutdown", object{})
		serviceErr := serviceInstall(c, true)
		_ = printJSON(object{"integrations": items, "service_error": errorText(serviceErr), "archive_preserved": c.Root})
		return err
	}
	c, err = ensureService(c)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	switch command {
	case "status":
		b, err := request(ctx, c, "GET", "status", nil)
		if err != nil {
			return err
		}
		var v object
		_ = json.Unmarshal(b, &v)
		v["integrations"] = IntegrationStatus(c)
		return printJSON(v)
	case "open":
		address := "http://" + c.Address + "/#token=" + c.Token
		fmt.Println("正在打开本地记忆库")
		return openBrowser(address)
	case "search":
		if len(rest) == 0 {
			return fail("请输入要搜索的内容")
		}
		b, err := request(ctx, c, "GET", "search?q="+urlQuery(strings.Join(rest, " ")), nil)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	case "show":
		if len(rest) == 0 {
			return fail("缺少会话编号")
		}
		id := rest[0]
		b, err := request(ctx, c, "GET", "sessions/"+url.PathEscape(id)+"?limit=500", nil)
		if err != nil {
			return err
		}
		if len(rest) > 1 && strings.Contains(strings.Join(rest[1:], " "), "markdown") {
			var v struct {
				Session Session `json:"session"`
			}
			_ = json.Unmarshal(b, &v)
			fmt.Printf("# %s\n\n", v.Session.Title)
			for offset := 0; ; offset += 500 {
				if offset > 0 {
					b, err = request(ctx, c, "GET", fmt.Sprintf("sessions/%s?offset=%d&limit=500", id, offset), nil)
					if err != nil {
						return err
					}
					_ = json.Unmarshal(b, &v)
				}
				for _, m := range v.Session.Messages {
					fmt.Printf("## %s · %s\n\n%s\n\n", m.Role, m.Time, m.Text)
				}
				if offset+len(v.Session.Messages) >= v.Session.Count {
					break
				}
			}
			return nil
		}
		fmt.Println(string(b))
		return nil
	case "resume":
		if len(rest) == 0 {
			return fail("缺少会话编号")
		}
		flags := flag.NewFlagSet("resume", flag.ContinueOnError)
		agent := flags.String("agent", "codex", "")
		dry := flags.Bool("dry-run", false, "")
		if err := flags.Parse(rest[1:]); err != nil {
			return err
		}
		b, err := request(ctx, c, "POST", "resume", object{"session": rest[0], "agent": *agent, "launch": !*dry})
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	case "export":
		flags := flag.NewFlagSet("export", flag.ContinueOnError)
		out := flags.String("out", "memvault-archive.zip", "")
		if err := flags.Parse(rest); err != nil {
			return err
		}
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://"+c.Address+"/api/v1/export", nil)
		req.Header.Set("Authorization", "Bearer "+c.Token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fail("导出失败：%s", resp.Status)
		}
		file, err := os.CreateTemp(filepath.Dir(*out), ".memvault-export-*")
		if err != nil {
			return err
		}
		name := file.Name()
		defer os.Remove(name)
		_, err = io.Copy(file, resp.Body)
		file.Close()
		if err != nil {
			return err
		}
		if err = os.Rename(name, *out); err != nil {
			return err
		}
		fmt.Println("已导出：", *out)
		return nil
	case "import":
		if len(rest) != 1 {
			return fail("请指定一个档案 ZIP")
		}
		file, err := os.Open(rest[0])
		if err != nil {
			return err
		}
		defer file.Close()
		req, _ := http.NewRequestWithContext(ctx, "POST", "http://"+c.Address+"/api/v1/import", file)
		req.Header.Set("Authorization", "Bearer "+c.Token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			return fail("导入失败：%s", b)
		}
		fmt.Println("档案已校验并导入")
		return nil
	case "repair":
		flags := flag.NewFlagSet("repair", flag.ContinueOnError)
		action := flags.String("action", "verify", "")
		if err := flags.Parse(rest); err != nil {
			return err
		}
		path := "repair"
		if *action == "scan" {
			path = "scan"
		}
		b, err := request(ctx, c, "POST", path, object{"action": *action})
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	return fail("未知命令 %s；运行 memvault help 查看用法", command)
}
func errorText(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
func setup(args []string) error {
	flags := flag.NewFlagSet("setup", flag.ContinueOnError)
	cwd, _ := os.Getwd()
	root := flags.String("root", cwd, "记忆位置")
	yes := flags.Bool("yes", false, "使用检测结果")
	noIntegrations := flags.Bool("no-integrations", false, "不修改 agent 配置")
	noService := flags.Bool("no-service", false, "不安装登录启动")
	noStart := flags.Bool("no-start", false, "只初始化，不启动服务")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*yes {
		answer, err := readPrompt("记忆位置 [" + *root + "]：")
		if err != nil && err != io.EOF {
			return err
		}
		if answer != "" {
			*root = answer
		}
		answer, err = readPrompt("导入可读取的历史，并配置三个 agent 与登录启动？[Y/n]：")
		if err != nil && err != io.EOF {
			return err
		}
		if strings.EqualFold(answer, "n") {
			return fail("已取消，未修改配置")
		}
	}
	abs, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	if err = Initialize(abs); err != nil {
		return err
	}
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	binary, _ = filepath.EvalSymlinks(binary)
	c, oldErr := loadConfig()
	if oldErr != nil {
		c = Config{Device: randomID(), Token: randomID(), Sources: defaultSources(), ContextBudget: 16000, Address: "127.0.0.1:27941"}
	}
	oldRoot := c.Root
	c.Root = abs
	c.Binary = binary
	if oldRoot != "" && oldRoot != abs {
		previous := c
		previous.Root = oldRoot
		if err = migrateRoot(previous, c); err != nil {
			return err
		}
	}
	if (!*noStart || !*noService) && oldRoot == abs {
		if err = stopServer(c); err != nil {
			return err
		}
	}
	if err = saveConfig(c); err != nil {
		return err
	}
	var integrations []Integration
	if !*noIntegrations {
		integrations, err = Configure(c, false)
		if err != nil {
			return err
		}
	}
	var serviceErr error
	if !*noService {
		serviceErr = serviceInstall(c, false)
	}
	result := object{"root": abs, "integrations": integrations, "service_installed": !*noService && serviceErr == nil, "service_error": errorText(serviceErr)}
	if !*noStart {
		if c, err = ensureService(c); err != nil {
			result["startup_error"] = err.Error()
		} else {
			result["web"] = "使用 memvault open 打开"
			result["history_import"] = "后台正在导入；可正常使用"
		}
	}
	return printJSON(result)
}
func IntegrationStatus(c Config) []Integration {
	var out []Integration
	for _, agent := range []string{"codex", "claude", "opencode"} {
		item := Integration{Agent: agent, Configured: false, State: "not_configured"}
		var configPath string
		switch agent {
		case "codex":
			configPath = filepath.Join(first(os.Getenv("CODEX_HOME"), filepath.Join(home(), ".codex")), "hooks.json")
		case "claude":
			configPath = filepath.Join(first(os.Getenv("CLAUDE_CONFIG_DIR"), filepath.Join(home(), ".claude")), "settings.json")
		case "opencode":
			configPath = filepath.Join(first(os.Getenv("XDG_CONFIG_HOME"), filepath.Join(home(), ".config")), "opencode", "plugins", "memvault.mjs")
		}
		if b, err := os.ReadFile(configPath); err == nil && (strings.Contains(string(b), "--integration "+marker) || agent == "opencode" && strings.Contains(string(b), "MemvaultPlugin")) {
			item.Configured = true
			item.State = "configured_not_observed"
		}
		var receipt object
		if readJSON(filepath.Join(stateDir(), "receipts", agent+".json"), &receipt) == nil {
			item.State = "hook_observed"
		}
		if agent == "codex" && item.Configured && item.State != "hook_observed" {
			item.State = "waiting_hook_review"
		}
		if agent == "claude" {
			if _, err := execLookPath("claude"); err != nil {
				item.State = "client_not_installed"
			}
		}
		out = append(out, item)
	}
	return out
}

// Migrate through the same self-contained transfer format. The old vault remains a recoverable copy.
func migrateRoot(old, next Config) error {
	if pathInside(old.Root, next.Root) || pathInside(next.Root, old.Root) {
		return fail("新旧记忆目录不能互相包含")
	}
	if _, err := os.Stat(filepath.Join(old.Root, "memvault.json")); err != nil {
		entries, _ := os.ReadDir(filepath.Join(next.Root, "journal"))
		if len(entries) == 0 {
			return fail("旧目录不可用，新目录也没有已同步档案；未切换")
		}
	} else {
		c, err := ensureService(old)
		if err != nil {
			return err
		}
		req, _ := http.NewRequest("GET", "http://"+c.Address+"/api/v1/export", nil)
		req.Header.Set("Authorization", "Bearer "+c.Token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fail("旧记忆库未能完成导出；未切换")
		}
		f, err := os.CreateTemp(stateDir(), "migration-*.zip")
		if err != nil {
			return err
		}
		defer os.Remove(f.Name())
		_, err = io.Copy(f, resp.Body)
		if err == nil {
			err = f.Sync()
		}
		f.Close()
		if err != nil {
			return err
		}
		target, err := OpenEngine(next)
		if err != nil {
			return err
		}
		err = target.Import(f.Name())
		if err == nil {
			issues := target.Verify()
			if len(issues) > 0 {
				err = fail("迁移校验失败：%s", strings.Join(issues, "；"))
			}
		}
		target.Close()
		if err != nil {
			return err
		}
	}
	_, _ = request(context.Background(), old, "POST", "shutdown", object{})
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		_, err := request(ctx, old, "GET", "status", nil)
		cancel()
		if err != nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fail("旧后台尚未停止，档案已复制，但本机配置没有切换")
}

func launchSetup(c Config, root string) error {
	if !filepath.IsAbs(root) {
		return fail("请提供新记忆目录的完整路径")
	}
	log, err := os.OpenFile(filepath.Join(stateDir(), "migration.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	cmd := exec.Command(c.Binary, "setup", "--root", root, "--yes")
	cmd.Stdout = log
	cmd.Stderr = log
	detach(cmd)
	if err = cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func stopServer(c Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := request(ctx, c, "POST", "shutdown", object{})
	cancel()
	if err != nil {
		return nil
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
		_, err = request(ctx, c, "GET", "status", nil)
		cancel()
		if err != nil {
			return nil
		}
	}
	return fail("后台尚未停止，配置保持原样；请稍后重试")
}

// 用户管理 CLI。直连数据库，不启动 HTTP 服务。
//
// 这是生产环境的逃生通道：即使 APP_ENV=prod 的启动校验（弱口令扫描）拦住了服务，
// 本工具依然可用，因为校验只在 HTTP 服务启动时执行。
//
//	go run scripts/admin_user/main.go list
//	go run scripts/admin_user/main.go create   --username alice --password 'Str0ng-Passw0rd' --role user
//	go run scripts/admin_user/main.go reset-password --username alice --password 'An0ther-Str0ng-Pw'
//	go run scripts/admin_user/main.go rename   --from admin --to myuser
//	go run scripts/admin_user/main.go disable  --username alice
//	go run scripts/admin_user/main.go enable   --username alice
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"backend-go/internal/config"
	"backend-go/internal/db"
	"backend-go/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cfg := config.Load()
	driver, dsn := cfg.EffectiveDB()
	d := db.Init(driver, dsn)
	if err := d.AutoMigrate(&db.User{}, &db.Session{}); err != nil {
		log.Fatalf("建表失败: %v", err)
	}
	svc := service.NewUserService(d, cfg)

	switch os.Args[1] {
	case "list":
		cmdList(svc)
	case "create":
		cmdCreate(svc)
	case "reset-password":
		cmdResetPassword(svc)
	case "rename":
		cmdRename(svc)
	case "disable":
		cmdSetStatus(svc, 0)
	case "enable":
		cmdSetStatus(svc, 1)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `用户管理 CLI

子命令:
  list                                              列出全部账号
  create        --username U --password P [--role admin|user] [--display-name D] [--must-change]
  reset-password --username U --password P [--must-change]
  rename        --from OLD --to NEW
  disable       --username U
  enable        --username U

注意: 口令通过命令行参数传递会进入 shell 历史。生产环境建议用引号包裹后执行,
并在执行后清理历史记录 (history -d), 或改用交互式终端。
`)
}

func cmdList(svc *service.UserService) {
	users, err := svc.ListUsers()
	if err != nil {
		log.Fatalf("读取用户列表失败: %v", err)
	}
	if len(users) == 0 {
		fmt.Println("（无账号。执行 create 子命令创建，或让服务在 users 表为空时自动引导）")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\t用户名\t显示名\t角色\t状态\t需改密\t最后登录\t创建时间")
	for _, u := range users {
		status := "启用"
		if u.Status != 1 {
			status = "禁用"
		}
		mustChange := "否"
		if u.MustChangePassword {
			mustChange = "是"
		}
		last := "-"
		if u.LastLoginAt != nil {
			last = u.LastLoginAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			u.ID, u.Username, u.DisplayName, u.Role, status, mustChange, last,
			u.CreatedAt.Format("2006-01-02 15:04"))
	}
	w.Flush()
}

func cmdCreate(svc *service.UserService) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	username := fs.String("username", "", "用户名（必填）")
	password := fs.String("password", "", "口令（必填）")
	role := fs.String("role", "user", "角色: admin | user")
	displayName := fs.String("display-name", "", "显示名（缺省同用户名）")
	mustChange := fs.Bool("must-change", false, "首次登录强制改密")
	fs.Parse(os.Args[2:])

	requireFlag("username", *username)
	requireFlag("password", *password)

	u, err := svc.CreateUser(*username, *password, *role, *displayName, *mustChange)
	if err != nil {
		log.Fatalf("创建失败: %v", err)
	}
	fmt.Printf("已创建: %s (id=%d, role=%s)\n", u.Username, u.ID, u.Role)
}

func cmdResetPassword(svc *service.UserService) {
	fs := flag.NewFlagSet("reset-password", flag.ExitOnError)
	username := fs.String("username", "", "用户名（必填）")
	password := fs.String("password", "", "新口令（必填）")
	mustChange := fs.Bool("must-change", true, "下次登录强制改密（默认开启）")
	fs.Parse(os.Args[2:])

	requireFlag("username", *username)
	requireFlag("password", *password)

	u, err := svc.ResetPassword(*username, *password, *mustChange)
	if err != nil {
		log.Fatalf("重置失败: %v", err)
	}
	fmt.Printf("已重置 %s 的口令，并踢出其全部会话（强制改密: %v）\n", u.Username, *mustChange)
}

func cmdRename(svc *service.UserService) {
	fs := flag.NewFlagSet("rename", flag.ExitOnError)
	from := fs.String("from", "", "原用户名（必填）")
	to := fs.String("to", "", "新用户名（必填）")
	fs.Parse(os.Args[2:])

	requireFlag("from", *from)
	requireFlag("to", *to)

	u, err := svc.RenameUser(*from, *to)
	if err != nil {
		log.Fatalf("改名失败: %v", err)
	}
	fmt.Printf("已改名: %s -> %s (id=%d)\n", *from, u.Username, u.ID)
	fmt.Println("提示: APP_ENV=prod 时用户名不得为保留字，且需确认 BOOTSTRAP_ADMIN_USERNAME 与新名一致。")
}

func cmdSetStatus(svc *service.UserService, status int8) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	username := fs.String("username", "", "用户名（必填）")
	fs.Parse(os.Args[2:])
	requireFlag("username", *username)

	u, err := svc.SetStatus(*username, status)
	if err != nil {
		log.Fatalf("操作失败: %v", err)
	}
	action := "禁用"
	if status == 1 {
		action = "启用"
	}
	fmt.Printf("已%s: %s (id=%d)\n", action, u.Username, u.ID)
}

func requireFlag(name, value string) {
	if value == "" {
		fmt.Fprintf(os.Stderr, "缺少必填参数 --%s\n\n", name)
		usage()
		os.Exit(2)
	}
}

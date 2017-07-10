package main

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli"
)

const (
	version = "v0.0.1"

	buffsize = 1500
)

func main() {
	app := cli.NewApp()
	app.Name = "kcpn"
	app.Description = "又一款VPN软件，基于UDP流量并使用KCP协议加速"
	app.Usage = "kcpn -l[isten] 0.0.0.0 -c[onfig] /path/to/kcpn/config"
	app.Version = version
	app.Author = "im帥 <iris-me@live.com>"
	app.Copyright = time.Now().Format("2006") + "© Prissh"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "listen, l",
			Value: "0.0.0.0",
			Usage: "kcpn服务器绑定IP，例如：0.0.0.0 或 192.168.1.1",
		},
		cli.StringFlag{
			Name:  "config,c",
			Value: "config.json",
			Usage: "kcpn配置文件路径，默认为当前目录中的config.json文件",
		},
	}
	app.Action = func(ctx *cli.Context) error {
		config := Config{}
		config.Mode = "normal"
		config.Crypt = "salsa20"
		config.NoComp = true
		config.Listen = ctx.String("listen")
		config.AutoExpire = 0
		config.ScavengeTTL = 600
		config.MTU = 1350
		config.SndWnd = 1024
		config.RcvWnd = 1024
		config.DataShard = 10
		config.ParityShard = 3
		config.DSCP = 0
		config.AckNodelay = false
		config.NoDelay = 0
		config.Interval = 40
		config.Resend = 0
		config.NoCongestion = 0
		config.SockBuf = 4194304
		config.KeepAlive = 10
		//读取配置文件
		err := parseJSONConfig(&config, ctx.String("config"))
		if err != nil {
			fmt.Printf("读取配置文件发生错误：%v, 将以默认配置运行...\n", err)
		} else {
			fmt.Printf("读取配置文件成功...\n")
		}
		switch config.Mode {
		case "normal":
			config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 0, 30, 2, 1
		case "fast":
			config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 0, 20, 2, 1
		case "fast2":
			config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 1, 20, 2, 1
		case "fast3":
			config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 1, 10, 2, 1
		}

		s := &Server{}
		s.cfg = &config
		s.Run()

		return nil
	}
	app.Run(os.Args)
}

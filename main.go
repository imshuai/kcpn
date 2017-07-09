package main

import (
	"errors"
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
	app.Usage = "kcpn -m[ode] server|client -c[onfig] /path/to/kcpn/config"
	app.Version = version
	app.Author = "im帥 <iris-me@live.com>"
	app.Copyright = time.Now().Format("2006") + "© Prissh"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "mode, m",
			Value: "",
			Usage: "kcpn运行模式,\"server\"或者\"client\"",
		},
		cli.StringFlag{
			Name:  "remote, r",
			Value: "",
			Usage: "kcpn服务器ip地址，例如：123.222.111.22",
		},
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
		cli.IntFlag{
			Name:  "autoexpire",
			Value: 0,
			Usage: "set auto expiration time(in seconds) for a single UDP connection, 0 to disable",
		},
		cli.IntFlag{
			Name:  "scavengettl",
			Value: 600,
			Usage: "set how long an expired connection can live(in sec), -1 to disable",
		},
		cli.IntFlag{
			Name:   "mtu",
			Value:  1350,
			Usage:  "set maximum transmission unit for UDP packets",
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "sndwnd",
			Value:  1024,
			Usage:  "set send window size(num of packets)",
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "rcvwnd",
			Value:  1024,
			Usage:  "set receive window size(num of packets)",
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "datashard",
			Value:  10,
			Usage:  "set reed-solomon erasure coding - datashard",
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "parityshard",
			Value:  3,
			Usage:  "set reed-solomon erasure coding - parityshard",
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "dscp",
			Value:  0,
			Usage:  "set DSCP(6bit)",
			Hidden: true,
		},
		cli.BoolFlag{
			Name:   "acknodelay",
			Usage:  "flush ack immediately when a packet is received",
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "nodelay",
			Value:  0,
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "interval",
			Value:  40,
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "resend",
			Value:  0,
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "nc",
			Value:  0,
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "sockbuf",
			Value:  4194304, // socket buffer size in bytes
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "keepalive",
			Value:  10, // nat keepalive interval in seconds
			Hidden: true,
		},
	}
	app.Action = func(ctx *cli.Context) error {
		config := Config{}
		config.Mode = "normal"
		config.Crypt = "salsa20"
		config.NoComp = true
		config.Remote = ctx.String("remote")
		config.Listen = ctx.String("listen")
		config.AutoExpire = ctx.Int("autoexpire")
		config.ScavengeTTL = ctx.Int("scavengettl")
		config.MTU = ctx.Int("mtu")
		config.SndWnd = ctx.Int("sndwnd")
		config.RcvWnd = ctx.Int("rcvwnd")
		config.DataShard = ctx.Int("datashard")
		config.ParityShard = ctx.Int("parityshard")
		config.DSCP = ctx.Int("dscp")
		config.AckNodelay = ctx.Bool("acknodelay")
		config.NoDelay = ctx.Int("nodelay")
		config.Interval = ctx.Int("interval")
		config.Resend = ctx.Int("resend")
		config.NoCongestion = ctx.Int("nc")
		config.SockBuf = ctx.Int("sockbuf")
		config.KeepAlive = ctx.Int("keepalive")
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
		//判断运行模式
		switch ctx.String("mode") {
		case "server":
			c := &Client{}
			c.Run()
		case "client":
			s := &Server{}
			s.Run()
		default:
			return errors.New("错误的运行模式, 运行模式只能是\"server\"或者\"client\"")
		}
		return nil
	}
	app.Run(os.Args)
}

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/songgao/water"
	"github.com/urfave/cli"
)

const (
	version = "v0.0.1"

	buffsize = 1500
)

func initClient() {
	fmt.Println("开始初始化虚拟网卡...")
	ifce, err := water.New(water.Config{
		DeviceType: water.TAP,
		PlatformSpecificParams: water.PlatformSpecificParams{
			ComponentID: "tap0901",
		},
	})
	if err != nil {
		fmt.Printf("虚拟网卡初始化失败，发生错误:%v\n", err)
		os.Exit(1)
	}
	defer func() {
		err := ifce.Close()
		if err != nil {
			fmt.Printf("虚拟网卡关闭失败，发生错误:%v\n", err)
		} else {
			fmt.Printf("虚拟网卡关闭成功...")
		}
	}()
	cmd := exec.Command("netsh", "interface", "ip", "set", "address", "name="+ifce.Name(), "source=static", "addr=10.1.0.10", "mask=255.255.255.0", "gateway=none")
	err = cmd.Run()
	if err != nil {
		fmt.Printf("netsh命令执行发生错误：%v\n", err)
		os.Exit(1)
	}
	fmt.Println("虚拟网卡IP地址设置成功...")
}

func initServer() {

}

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
			Name:  "mtu",
			Value: 1350,
			Usage: "set maximum transmission unit for UDP packets",
		},
		cli.IntFlag{
			Name:  "sndwnd",
			Value: 1024,
			Usage: "set send window size(num of packets)",
		},
		cli.IntFlag{
			Name:  "rcvwnd",
			Value: 1024,
			Usage: "set receive window size(num of packets)",
		},
		cli.IntFlag{
			Name:  "datashard",
			Value: 10,
			Usage: "set reed-solomon erasure coding - datashard",
		},
		cli.IntFlag{
			Name:  "parityshard",
			Value: 3,
			Usage: "set reed-solomon erasure coding - parityshard",
		},
		cli.IntFlag{
			Name:  "dscp",
			Value: 0,
			Usage: "set DSCP(6bit)",
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
		config.Remote = c.String("remote")
		config.Listen = c.String("listen")
		config.MTU = c.Int("mtu")
		config.SndWnd = c.Int("sndwnd")
		config.RcvWnd = c.Int("rcvwnd")
		config.DataShard = c.Int("datashard")
		config.ParityShard = c.Int("parityshard")
		config.DSCP = c.Int("dscp")
		config.AckNodelay = c.Bool("acknodelay")
		config.NoDelay = c.Int("nodelay")
		config.Interval = c.Int("interval")
		config.Resend = c.Int("resend")
		config.NoCongestion = c.Int("nc")
		config.SockBuf = c.Int("sockbuf")
		config.KeepAlive = c.Int("keepalive")
		err := parseJSONConfig(&config, c.String("config"))
		if err != nil {
			fmt.Printf("读取配置文件发生错误：%v, 将以默认配置运行...\n")
		} else {
			fmt.Printf("读取配置文件成功...\n")
		}
		switch ctx.String("mode") {
		case "server":
			initServer()
		case "client":
			initClient()
		default:
			return errors.New("错误的运行模式, 运行模式只能是\"server\"或者\"client\"")
		}
	}
	app.Run(os.Args)
}

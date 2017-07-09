package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/songgao/water"
)

//Server 定义服务器类
type Server struct {
	ifce *water.Interface
	cfg  *Config
	pool *ippool
}

func ipToInt(ipstring string) int {
	ipSegs := strings.Split(ipstring, ".")
	ipInt := 0
	var pos uint = 24
	for _, ipSeg := range ipSegs {
		tempInt, _ := strconv.Atoi(ipSeg)
		tempInt = tempInt << pos
		ipInt = ipInt | tempInt
		pos -= 8
	}
	return ipInt
}

func ipIntToIP(ipInt int) net.IP {
	var ipSegs net.IP
	for i := 0; i < 4; i++ {
		tempInt := ipInt & 0xFF
		ipSegs[4-i-1] = byte(tempInt)
		ipInt = ipInt >> 8
	}
	return ipSegs
}

func (s *Server) handleRequset(conn net.Conn) {
	defer conn.Close()
	rcvBuffer := make([]byte, 4096)
	n, err := conn.Read(rcvBuffer[0:])
	if err != nil {
		fmt.Printf("从客户端读取请求数据时发生错误：%v\n", err)
		return
	}
	if n != len(reqLocalIP) {
		err = errors.New("数据长度与预期不符")
		fmt.Printf("从客户端读取请求数据时发生错误：%v\n", err)
		return
	}
	for i := 0; i < 2; i++ {
		if rcvBuffer[i] != reqLocalIP[i] {
			err = errors.New("数据与预期不符")
			fmt.Printf("从客户端读取请求数据时发生错误：%v\n", err)
			return
		}
	}
	tIP, err := s.pool.Pop()
	if err != nil {
		return
	}
	n, err = conn.Write([]byte{0x01, 0x00, tIP[12], tIP[13], tIP[14], tIP[15]})
	if err != nil {
		return
	}
	if n != 6 {
		return
	}
	n, err = conn.Read(rcvBuffer[0:])
	if err != nil {
		return
	}
	if n != 2 {
		return
	}
	if rcvBuffer[0] != 0x01 || rcvBuffer[1] != 0x01 {
		return
	}
}

func (s *Server) runTCPservice() {
	listener, err := net.Listen("tcp4", ":2342")
	if err != nil {
		fmt.Printf("listen on tcp4://0.0.0.0:2342 fail with error:%v\n", err)
		os.Exit(1)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("accept new connection fail with error:%v\n", err)
			os.Exit(1)
		}
		go s.handleRequset(conn)
	}
}

func (s *Server) initVirtualAdapter() error {
	fmt.Println("开始初始化虚拟网卡...")
	ifce, err := water.New(water.Config{
		DeviceType: water.TAP,
		PlatformSpecificParams: water.PlatformSpecificParams{
			ComponentID: "tap0901",
		},
	})
	if err != nil {
		fmt.Printf("虚拟网卡初始化失败，发生错误:%v\n", err)
		return err
	}
	defer func() {
		err := ifce.Close()
		if err != nil {
			fmt.Printf("虚拟网卡关闭失败，发生错误:%v\n", err)
		} else {
			fmt.Printf("虚拟网卡关闭成功...")
		}
	}()

	cmd := exec.Command("netsh", "interface", "ip", "set", "address", "name="+ifce.Name(), "source=static", "addr="+c.localIP.String(), "mask=255.255.255.0", "gateway=none")
	err = cmd.Run()
	if err != nil {
		fmt.Printf("设置虚拟网卡IP命令执行发生错误：%v\n", err)
		return err
	}
	fmt.Println("虚拟网卡IP地址设置成功...")
	return nil
}

//Run 服务器运行
func (s *Server) Run() {

}

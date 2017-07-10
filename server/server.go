package main

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"github.com/golang/snappy"
	"github.com/songgao/water"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
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

func runCMD(args ...string) error {
	cmd := exec.Command("/sbin/ip", args...)
	err := cmd.Run()
	return err
}

func (s *Server) initVirtualAdapter() error {
	fmt.Println("开始初始化虚拟网卡...")
	ifce, err := water.New(water.Config{
		DeviceType: water.TAP,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: "kcpn0",
		},
	})
	if err != nil {
		fmt.Printf("虚拟网卡初始化失败，发生错误:%v\n", err)
		return err
	}
	//	defer func() {
	//		err := ifce.Close()
	//		if err != nil {
	//			fmt.Printf("虚拟网卡关闭失败，发生错误:%v\n", err)
	//		} else {
	//			fmt.Printf("虚拟网卡关闭成功...")
	//		}
	//	}()

	err = runCMD("addr", "add", "173.10.10.1/24", "dev", ifce.Name())
	if err != nil {
		fmt.Printf("设置虚拟网卡IP命令执行发生错误：%v\n", err)
		return err
	}
	err = runCMD("link", "set", "dev", ifce.Name(), "up")
	if err != nil {
		fmt.Printf("设置虚拟网卡IP命令执行发生错误：%v\n", err)
		return err
	}
	fmt.Println("虚拟网卡IP地址设置成功...")
	s.ifce = ifce
	return nil
}

//Run 服务器运行
func (s *Server) Run() {
	rand.Seed(int64(time.Now().Nanosecond()))
	err := s.initVirtualAdapter()
	if err != nil {
		os.Exit(1)
	}
	pass := pbkdf2.Key([]byte("kcpn-"+version), []byte("im帥"), 4096, 32, sha1.New)
	var block kcp.BlockCrypt
	switch s.cfg.Crypt {
	case "tea":
		block, _ = kcp.NewTEABlockCrypt(pass[:16])
	case "xor":
		block, _ = kcp.NewSimpleXORBlockCrypt(pass)
	case "none":
		block, _ = kcp.NewNoneBlockCrypt(pass)
	case "aes-128":
		block, _ = kcp.NewAESBlockCrypt(pass[:16])
	case "aes-192":
		block, _ = kcp.NewAESBlockCrypt(pass[:24])
	case "blowfish":
		block, _ = kcp.NewBlowfishBlockCrypt(pass)
	case "twofish":
		block, _ = kcp.NewTwofishBlockCrypt(pass)
	case "cast5":
		block, _ = kcp.NewCast5BlockCrypt(pass[:16])
	case "3des":
		block, _ = kcp.NewTripleDESBlockCrypt(pass[:24])
	case "xtea":
		block, _ = kcp.NewXTEABlockCrypt(pass[:16])
	case "salsa20":
		block, _ = kcp.NewSalsa20BlockCrypt(pass)
	default:
		s.cfg.Crypt = "aes"
		block, _ = kcp.NewAESBlockCrypt(pass)
	}
	lis, err := kcp.ListenWithOptions(s.cfg.Listen, block, s.cfg.DataShard, s.cfg.ParityShard)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := lis.SetDSCP(s.cfg.DSCP); err != nil {
		log.Println("SetDSCP:", err)
	}
	if err := lis.SetReadBuffer(s.cfg.SockBuf); err != nil {
		log.Println("SetReadBuffer:", err)
	}
	if err := lis.SetWriteBuffer(s.cfg.SockBuf); err != nil {
		log.Println("SetWriteBuffer:", err)
	}
	for {
		if conn, err := lis.AcceptKCP(); err == nil {
			log.Println("remote address:", conn.RemoteAddr())
			conn.SetStreamMode(true)
			conn.SetWriteDelay(true)
			conn.SetNoDelay(s.cfg.NoDelay, s.cfg.Interval, s.cfg.Resend, s.cfg.NoCongestion)
			conn.SetMtu(s.cfg.MTU)
			conn.SetWindowSize(s.cfg.SndWnd, s.cfg.RcvWnd)
			conn.SetACKNoDelay(s.cfg.AckNodelay)

			if s.cfg.NoComp {
				go handleMux(conn, s)
			} else {
				go handleMux(newCompStream(conn), s)
			}
		} else {
			log.Printf("%+v", err)
		}
	}
}

type compStream struct {
	conn net.Conn
	w    *snappy.Writer
	r    *snappy.Reader
}

func (c *compStream) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *compStream) Write(p []byte) (n int, err error) {
	n, err = c.w.Write(p)
	err = c.w.Flush()
	return n, err
}

func (c *compStream) Close() error {
	return c.conn.Close()
}

func newCompStream(conn net.Conn) *compStream {
	c := new(compStream)
	c.conn = conn
	c.w = snappy.NewBufferedWriter(conn)
	c.r = snappy.NewReader(conn)
	return c
}

// handle multiplex-ed connection
func handleMux(conn io.ReadWriteCloser, s *Server) {
	// stream multiplex
	smuxConfig := smux.DefaultConfig()
	smuxConfig.MaxReceiveBuffer = s.cfg.SockBuf
	smuxConfig.KeepAliveInterval = time.Duration(s.cfg.KeepAlive) * time.Second

	mux, err := smux.Server(conn, smuxConfig)
	if err != nil {
		log.Println(err)
		return
	}
	defer mux.Close()
	for {
		p1, err := mux.AcceptStream()
		if err != nil {
			log.Println(err)
			return
		}
		go handleClient(p1, s.ifce)
	}
}

func handleClient(p1, p2 io.ReadWriteCloser) {
	log.Println("stream opened")
	defer log.Println("stream closed")
	defer p1.Close()
	//	defer p2.Close()

	// start tunnel
	p1die := make(chan struct{})
	go func() { io.Copy(p1, p2); close(p1die) }()

	p2die := make(chan struct{})
	go func() { io.Copy(p2, p1); close(p2die) }()

	// wait for tunnel termination
	select {
	case <-p1die:
	case <-p2die:
	}
}

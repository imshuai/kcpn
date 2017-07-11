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
	"time"

	"github.com/songgao/water"
	kcp "github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

//Client 定义客户端类
type Client struct {
	ifce    *water.Interface
	cfg     *Config
	localIP net.IP
}

//RequestLocalIP 向远程服务器请求本地IP地址
func (c *Client) RequestLocalIP() error {
	fmt.Println("向服务器申请IP...")
	conn, err := net.Dial("tcp4", c.cfg.Remote+":2342")
	if err != nil {
		fmt.Printf("连接至服务器发生错误：%v\n", err)
		return err
	}
	defer conn.Close()
	rcvBuffer := make([]byte, 4096)
	n, err := conn.Write(reqLocalIP)
	if err != nil {
		fmt.Printf("向服务器发送请求数据时发生错误：%v\n", err)
		return err
	}
	if n != len(reqLocalIP) {
		err = errors.New("数据长度与预期不符")
		fmt.Printf("向服务器发送请求数据时发生错误：%v\n", err)
		return err
	}
	n, err = conn.Read(rcvBuffer[0:])
	if err != nil {
		fmt.Printf("从服务器读取数据时发生错误：%v\n", err)
		return err
	}
	if rcvBuffer[0] != 0x01 {
		err = errors.New("错误应答码")
		fmt.Printf("从服务器读取数据时发生错误：%v\n", err)
		return err
	}
	localIP := net.IPv4(rcvBuffer[2], rcvBuffer[3], rcvBuffer[4], rcvBuffer[5])
	fmt.Printf("向服务器接申请IP成功：%s\n", localIP.String())
	c.localIP = localIP
	conn.Write(reqLocalIPComplete)
	return nil
}

//InitVirtualAdapter 初始化虚拟网卡
func (c *Client) initVirtualAdapter() error {
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
	//	defer func() {
	//		err := ifce.Close()
	//		if err != nil {
	//			fmt.Printf("虚拟网卡关闭失败，发生错误:%v\n", err)
	//		} else {
	//			fmt.Printf("虚拟网卡关闭成功...")
	//		}
	//	}()
	//	err = c.RequestLocalIP()
	//	if err != nil {
	//		return err
	//	}
	cmd := exec.Command("netsh", "interface", "ip", "set", "address", "name="+ifce.Name(), "source=static", "addr=173.10.10.2", "mask=255.255.255.0", "gateway=none")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		fmt.Printf("设置虚拟网卡IP命令执行发生错误：%v\n", err)
		return err
	}
	fmt.Println("虚拟网卡IP地址设置成功...")
	c.ifce = ifce
	return nil
}

//Run 开始执行
func (c *Client) Run() {
	rand.Seed(int64(time.Now().Nanosecond()))
	err := c.initVirtualAdapter()
	if err != nil {
		os.Exit(1)
	}
	pass := pbkdf2.Key([]byte("kcpn-"+version), []byte("im帥"), 4096, 32, sha1.New)
	var block kcp.BlockCrypt
	switch c.cfg.Crypt {
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
		c.cfg.Crypt = "aes"
		block, _ = kcp.NewAESBlockCrypt(pass)
	}
	//smuxConfig := smux.DefaultConfig()
	//smuxConfig.MaxReceiveBuffer = c.cfg.SockBuf

	//createConn := func() (*smux.Session, error) {
	createConn := func() (*kcp.UDPSession, error) {
		kcpconn, err := kcp.DialWithOptions(c.cfg.Remote, block, c.cfg.DataShard, c.cfg.ParityShard)
		if err != nil {
			return nil, errors.New("连接至远程服务器失败，发生错误：" + err.Error())
		}
		kcpconn.SetStreamMode(true)
		kcpconn.SetWriteDelay(true)
		kcpconn.SetNoDelay(c.cfg.NoDelay, c.cfg.Interval, c.cfg.Resend, c.cfg.NoCongestion)
		kcpconn.SetWindowSize(c.cfg.SndWnd, c.cfg.RcvWnd)
		kcpconn.SetMtu(c.cfg.MTU)
		kcpconn.SetACKNoDelay(c.cfg.AckNodelay)
		if err := kcpconn.SetDSCP(c.cfg.DSCP); err != nil {
			log.Println("SetDSCP:", err)
		}
		if err := kcpconn.SetReadBuffer(c.cfg.SockBuf); err != nil {
			log.Println("SetReadBuffer:", err)
		}
		if err := kcpconn.SetWriteBuffer(c.cfg.SockBuf); err != nil {
			log.Println("SetWriteBuffer:", err)
		}
		return kcpconn, nil

		//		var session *smux.Session
		//		if c.cfg.NoComp {
		//			session, err = smux.Client(kcpconn, smuxConfig)
		//		} else {
		//			session, err = smux.Client(newCompStream(kcpconn), smuxConfig)
		//		}
		//		if err != nil {
		//			return nil, errors.New("建立会话失败，发生错误：" + err.Error())
		//		}
		//		return session, nil
	}

	//handleClient(conn, c.ifce)

	// wait until a connection is ready
	//waitConn := func() *smux.Session {
	waitConn := func() *kcp.UDPSession {
		for {
			if session, err := createConn(); err == nil {
				return session
			}
			time.Sleep(time.Second)
		}
	}

	//	numconn := uint16(c.cfg.Conn)
	//	muxes := make([]struct {
	//		session *smux.Session
	//		ttl     time.Time
	//	}, numconn)

	//	for k := range muxes {
	//		sess, err := createConn()
	//		if err != nil {
	//			return
	//		}
	//		muxes[k].session = sess
	//		muxes[k].ttl = time.Now().Add(time.Duration(c.cfg.AutoExpire) * time.Second)
	//	}

	//	chScavenger := make(chan *smux.Session, 128)
	//	go scavenger(chScavenger)
	//	rr := uint16(0)
	for {

		//		idx := rr % numconn

		//		// do auto expiration && reconnection
		//		if muxes[idx].session.IsClosed() || (c.cfg.AutoExpire > 0 && time.Now().After(muxes[idx].ttl)) {
		//			chScavenger <- muxes[idx].session
		//			muxes[idx].session = waitConn()
		//			muxes[idx].ttl = time.Now().Add(time.Duration(c.cfg.AutoExpire) * time.Second)
		//		}
		session := waitConn()
		handleClient(session, c.ifce)
		//		rr++
	}
}

//func handleClient(sess *smux.Session, p1 io.ReadWriteCloser) {
func handleClient(p2 *kcp.UDPSession, p1 io.ReadWriteCloser) {
	//	p2, err := sess.OpenStream()
	//	if err != nil {
	//		return
	//	}

	go log.Println("stream opened")
	defer log.Println("stream closed")
	defer p2.Close()

	// start tunnel
	p2die := make(chan struct{})
	go func() {
		io.Copy(p2, p1)
		close(p2die)
	}()
	p1die := make(chan struct{})
	go func() {
		io.Copy(p1, p2)
		close(p1die)
	}()

	// wait for tunnel termination
	select {
	case <-p1die:
	case <-p2die:
	}
}

//type compStream struct {
//	conn net.Conn
//	w    *snappy.Writer
//	r    *snappy.Reader
//}

//func (c *compStream) Read(p []byte) (n int, err error) {
//	return c.r.Read(p)
//}

//func (c *compStream) Write(p []byte) (n int, err error) {
//	n, err = c.w.Write(p)
//	err = c.w.Flush()
//	return n, err
//}

//func (c *compStream) Close() error {
//	return c.conn.Close()
//}

//func newCompStream(conn net.Conn) *compStream {
//	c := new(compStream)
//	c.conn = conn
//	c.w = snappy.NewBufferedWriter(conn)
//	c.r = snappy.NewReader(conn)
//	return c
//}

//type scavengeSession struct {
//	session *smux.Session
//	ttl     time.Time
//}

//const (
//	maxScavengeTTL = 10 * time.Minute
//)

//func scavenger(ch chan *smux.Session) {
//	ticker := time.NewTicker(30 * time.Second)
//	defer ticker.Stop()
//	var sessionList []scavengeSession
//	for {
//		select {
//		case sess := <-ch:
//			sessionList = append(sessionList, scavengeSession{sess, time.Now()})
//		case <-ticker.C:
//			var newList []scavengeSession
//			for k := range sessionList {
//				s := sessionList[k]
//				if s.session.NumStreams() == 0 || s.session.IsClosed() || time.Since(s.ttl) > maxScavengeTTL {
//					log.Println("session scavenged")
//					s.session.Close()
//				} else {
//					newList = append(newList, sessionList[k])
//				}
//			}
//			sessionList = newList
//		}
//	}
//}

package main

import (
	"crypto/tls"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func slave(ServerTarget string, LocalS5port string) {
	var RemoteConn net.Conn
	var err error
	go CreateForwardSocks(LocalS5port)
	fmt.Println("init success")

	conf := &tls.Config{
		//不验证证书的合法性
		InsecureSkipVerify: true,
	}

	for {
		for {
			RemoteConn, err = tls.Dial("tcp", ServerTarget, conf)
			if err == nil {
				break
			}
			//RemoteConn, err = net.Dial("tcp", listenTarget)
			//if err == nil {
			//	break
			//}
		}

		//Forward to socks
		localsocks := "127.0.0.1:" + LocalS5port
		Localsockscon, err := net.Dial("tcp", localsocks)
		if err != nil {
			Localsockscon.Close()
			return
		}
		fmt.Println(time.Now().String()[:19], "connet:", RemoteConn.RemoteAddr().String())
		Socks5Forward(RemoteConn, Localsockscon)
	}
}

func CreateForwardSocks(LocalS5port string) {
	fmt.Println("LocalS5port start..." + LocalS5port)
	LocalS5port = ":" + LocalS5port
	server, err := net.Listen("tcp", LocalS5port)
	if err != nil {
		fmt.Printf("LOCAL Listen failed: %v\n", err)
		return
	}

	for {
		client, err := server.Accept()
		if err != nil {
			fmt.Printf("Accept failed: %v\n", err)
			continue
		}
		go process(client)
	}
}

func process(client net.Conn) {

	//Auth
	if err := Socks5Auth(client); err != nil {
		fmt.Println("Auth error:", err)
		client.Close()
		return
	}

	//Connect
	target, err := Socks5Connect(client)
	if err != nil {
		fmt.Println("Connect error:", err)
		client.Close()
		return
	}
	//Forward
	Socks5Forward(client, target)
}

func Socks5Auth(client net.Conn) (err error) {
	buf := make([]byte, 256)

	// 读取 VER 和 NMETHODS
	n, err := io.ReadFull(client, buf[:2])
	if n != 2 {
		return errors.New("reading header: " + err.Error())
	}

	ver, nMethods := int(buf[0]), int(buf[1])
	if ver != 5 {
		return errors.New("invalid version 5")
	}

	// 读取 METHODS 列表
	n, err = io.ReadFull(client, buf[:nMethods])
	if n != nMethods {
		return errors.New("reading methods: " + err.Error())
	}

	//无需认证
	n, err = client.Write([]byte{0x05, 0x00})
	if n != 2 || err != nil {
		return errors.New("write rsp: " + err.Error())
	}

	return nil
}

func Socks5Connect(client net.Conn) (net.Conn, error) {
	buf := make([]byte, 256)

	//前四个 VAR CMD RSV ATYP
	n, err := io.ReadFull(client, buf[:4])
	if n != 4 {
		return nil, errors.New("read header: " + err.Error())
	}

	//前四个 VAR CMD RSV ATYP
	ver, cmd, _, atyp := buf[0], buf[1], buf[2], buf[3]
	if ver != 5 || cmd != 1 {
		return nil, errors.New("invalid ver/cmd")
	}

	addr := ""
	//ATYP	地址类型
	switch atyp {
	case 1:
		n, err = io.ReadFull(client, buf[:4])
		if n != 4 {
			return nil, errors.New("invalid IPv4: " + err.Error())
		}
		addr = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])

	case 3:
		n, err = io.ReadFull(client, buf[:1])
		if n != 1 {
			return nil, errors.New("invalid hostname: " + err.Error())
		}
		addrLen := int(buf[0])

		n, err = io.ReadFull(client, buf[:addrLen])
		if n != addrLen {
			return nil, errors.New("invalid hostname: " + err.Error())
		}
		addr = string(buf[:addrLen])

	case 4:
		return nil, errors.New("no supported IPv6")

	default:
		return nil, errors.New("invalid atyp")
	}

	n, err = io.ReadFull(client, buf[:2])
	if n != 2 {
		return nil, errors.New("read port: " + err.Error())
	}
	port := binary.BigEndian.Uint16(buf[:2])

	destAddrPort := fmt.Sprintf("%s:%d", addr, port)
	dest, err := net.Dial("tcp", destAddrPort)
	if err != nil {
		return nil, errors.New("dial dst: " + err.Error())
	}

	n, err = client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		dest.Close()
		return nil, errors.New("write rsp: " + err.Error())
	}

	return dest, nil
}

func Socks5Forward(client, target net.Conn) {
	forward := func(src, dest net.Conn) {
		defer src.Close()
		defer dest.Close()
		io.Copy(src, dest)
	}
	go forward(client, target)
	go forward(target, client)
}

func main() {
	ServerTarget := flag.String("t", "", "ServerTarget ex:-t 1.2.3.4:8888")
	LocalS5port := flag.String("local", "1080", "loacl s5 port ex:-local 1080")
	flag.Parse()
	if *ServerTarget == "" {
		log.Println("[Info] 请填server地址,ex: 1.2.3.4:8888")
		os.Exit(1)
	} else {
		fmt.Println("ServerTarget:" + *ServerTarget)
		fmt.Println("LocalS5port:" + *LocalS5port)
		slave(*ServerTarget, *LocalS5port)
	}
}

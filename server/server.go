package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func Server(port, s5port string) {

	//证书模块
	cert, err := tls.LoadX509KeyPair("server.pem", "server.key")
	if err != nil {
		log.Println(err)
		return
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	//证书模块:建立连接
	lis_ip_port := ":" + port
	lis, err := tls.Listen("tcp", lis_ip_port, config)
	if err != nil {
		log.Println(err)
		return
	}
	defer lis.Close()
	fmt.Println("Server Listen :", port)

	// 没有tls
	//lis, err := net.ListenTCP("tcp", &net.TCPAddr{net.ParseIP(ip), port, ""})
	//ErrHandler(err)
	//defer lis.Close()
	//fmt.Println("Listen :", port)

	s5port_str := "0.0.0.0" + ":" + s5port
	s5lis, err := net.Listen("tcp", s5port_str)
	//s5lis, err := net.ListenTCP("tcp", &net.TCPAddr{net.ParseIP(ip), s5port, ""})
	if err != nil {
		panic(err)
	}
	defer s5lis.Close()
	fmt.Println("localsocks5 port:", s5port)

	NetHandle(lis, s5lis)
}

func NetHandle(listen net.Listener, s5listen net.Listener) {
	for {
		s5conn, err := s5listen.Accept()
		if err != nil {
			fmt.Println(time.Now().String()[:19], "接受客户端连接error:", err.Error())
			continue
		}
		fmt.Println(time.Now().String()[:19], "用户客户端连接来自:", s5conn.RemoteAddr().String())
		defer s5conn.Close()

		conn, err := listen.Accept()
		if err != nil {
			fmt.Println(time.Now().String()[:19], "接受客户端连接error:", err.Error())
			continue
		}
		fmt.Println(time.Now().String()[:20], "客户端连接来自:", conn.RemoteAddr().String())
		defer conn.Close()

		go Forwardhandle(conn, s5conn)
	}
}

func Forwardhandle(sconn net.Conn, dconn net.Conn) {
	defer sconn.Close()
	defer dconn.Close()
	ExitChan := make(chan bool, 1)
	go func(sconn net.Conn, dconn net.Conn, Exit chan bool) {
		io.Copy(dconn, sconn)
		ExitChan <- true
	}(sconn, dconn, ExitChan)

	go func(sconn net.Conn, dconn net.Conn, Exit chan bool) {
		io.Copy(sconn, dconn)
		ExitChan <- true
	}(sconn, dconn, ExitChan)
	<-ExitChan
	dconn.Close()
}

func main() {
	ServerListen := flag.String("l", "8888", "ServerListen ex: -l 8888")
	User_5S_5port := flag.String("s5", "2080", "user s5 port ex: -s5 2080")
	flag.Parse()
	if *ServerListen == "" {
		log.Println("[Info] 请填server地址Listener, ex: 8888")
		os.Exit(1)
	} else {
		fmt.Println("ServerListen:" + *ServerListen)
		fmt.Println("User_5S_5port:" + *User_5S_5port)
		Server(*ServerListen, *User_5S_5port)
	}
}

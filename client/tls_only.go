package main

import (
	"crypto/tls"
	"log"
	"net"
)

func main() {

	conf := &tls.Config{
		//不验证证书的合法性
		InsecureSkipVerify: true,
	}
	var conn net.Conn
	var err error
	conn, err = tls.Dial("tcp", "127.0.0.1:3000", conf)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	n, err := conn.Write([]byte("hello\n"))
	if err != nil {
		log.Println(n, err)
		return
	}
	buf := make([]byte, 100)
	n, err = conn.Read(buf)
	if err != nil {
		log.Println(n, err)
		return
	}
	println(string(buf[:n]))
}

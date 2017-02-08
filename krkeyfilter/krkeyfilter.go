package main

import (
	"bytes"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("not enough arguments")
	}
	var host, port string
	host = os.Args[1]
	if len(os.Args) >= 3 {
		port = os.Args[2]
	} else {
		port = "22"
	}

	remoteConn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		log.Fatal("err connecting to remote:", err.Error())
	}

	go func() {
		for {
			buf := make([]byte, 1<<15)
			n, err := remoteConn.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				byteBuf := bytes.NewBuffer(buf[:n])
				wroteN, err := byteBuf.WriteTo(os.Stdout)
				if wroteN != int64(n) {
					log.Println("not all bytes written")
				}
				if err != nil {
					log.Println("err writing remote to stdout", err.Error())
					return
				}
			}
		}
	}()

	go func() {
		for {
			buf := make([]byte, 1<<15)
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				byteBuf := bytes.NewBuffer(buf[:n])
				wroteN, err := byteBuf.WriteTo(remoteConn)
				if wroteN != int64(n) {
					log.Println("not all bytes written")
				}
				if err != nil {
					log.Println("err writing stdin to remote", err.Error())
					return
				}
			}
		}
	}()

	select {}
}

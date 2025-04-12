package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	for{
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		// conn.SetReadDeadline(time.Time{})
		go func (conn net.Conn){
			fmt.Println("Incoming connection")
			defer conn.Close()
			buf := make([]byte, 1024)
			// conn.SetDeadline(time.Now().Add(10* time.Second))
			n, err := conn.Read(buf)
			if err != nil && err != io.EOF {
				fmt.Println("Error receiving data from client: ", err.Error())
				return
			}
			fmt.Println("data from client", string(buf[:n]))
			if _, err = io.WriteString(conn, "HTTP/1.1 200 OK\r\n\r\n"); err != nil {
				fmt.Println("Error sending data to client: ", err.Error())
				return
			}
		}(conn)
	}
	

}

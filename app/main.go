package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Starting http server...")

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
		
		go handleConnection(conn)
	}
	

}

func clean(input []byte)string{
	output := string(input[:max(len(input)-2, 0)])
	return output
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	headers := make(map[string][]string)
	reader := bufio.NewReader(conn)

	// Read request line
	fmt.Println()
	reqL, err := reader.ReadBytes('\n')
	if err != nil{
		fmt.Errorf("Error reading request line from client request %s", err.Error())
		return
	}
	rq := clean(reqL)
	requestLine := strings.Split(rq, " ")
	if len(requestLine) != 3{
		fmt.Printf("Invalid request line received %s\n", rq)
		return		
	}
	
	method, path, protocol := requestLine[0], requestLine[1], requestLine[2]
	fmt.Println(method, path, protocol)
	// Read headers
	fmt.Println()
	for {
		
		h, err := reader.ReadBytes('\n')
		if err != nil{
			fmt.Printf("Error reading headers from client request %s\n", err.Error())
			return
		}
		if bytes.Equal(h, []byte{'\r', '\n'}){
			break
		}
		cleanheader := strings.TrimSpace(clean(h))
		hdr := strings.SplitN(cleanheader, ":", 2)
		if len(hdr) != 2{
			fmt.Printf("Invalid header received %s", string(h))
			continue
		}
		headers[hdr[0]] = strings.Split(strings.TrimSpace(hdr[1]), ",")
	}
	for k, v := range headers{
		fmt.Printf("%s : " , k)
		for _, s := range v{
			fmt.Printf("%s", s)
		}
		fmt.Println()
	}

	// Read body if it exists
	fmt.Println()
	value, exists := headers["Content-Length"]
	if exists{
		cl, err := strconv.Atoi(value[0]) 
		if err == nil && cl > 0{
			body := make([]byte, cl)
			_, err = io.ReadFull(reader, body)
			if err != nil{
				fmt.Printf("Error reading body from client request %v", err)
			}		
			fmt.Println(string(body[:]))
		}else{
			fmt.Printf("Invalid Content-Length header received %v", err)
		}
	}
	fmt.Println()
	// send response back to client
	rsp := ""
	if strings.EqualFold(path, "/"){
		rsp  = "HTTP/1.1 200 OK\r\n\r\n"
	}else if strings.HasPrefix(path, "/echo/"){
		op := strings.SplitN(path, "/", 3)
		rsp  = fmt.Sprintf(
		"HTTP/1.1 200 OK" +
		"\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: %d\r\n\r\n" +
		"%s", len(op[2]), op[2],
	)
	}else if strings.EqualFold(path, "/user-agent"){
		if v, exists := headers["User-Agent"]; exists && len(v) > 0{
			rsp  = fmt.Sprintf(
				"HTTP/1.1 200 OK" +
				"\r\n" +
				"Content-Type: text/plain\r\n" +
				"Content-Length: %d\r\n\r\n" +
				"%s", len(v[0]), v[0],
			)
		}

	}else{
			rsp  = "HTTP/1.1 404 Not Found\r\n\r\n"
	}
	

	_, err = conn.Write([]byte(rsp))
	if err != nil{
		fmt.Errorf("Error reading headers from client request %s", err.Error())
		return
	}	
}

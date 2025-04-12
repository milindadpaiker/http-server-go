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

func handleConnection(conn net.Conn) {
	status := "200 OK"
	contentLength := 0
	contentType := "text/plain"
	body := ""
	defer conn.Close()
	
	//Read client request
	req, err := NewRequest(conn)
	if err != nil{
		status = HTTPStatus[400]
	}
	
	
	//Path based routing and response handling
 	if strings.EqualFold(req.Path, "/"){
		status = HTTPStatus[200]
	}else if strings.HasPrefix(req.Path, "/echo/"){
		echoResponse := strings.SplitN(req.Path, "/", 3)
		contentLength = len(echoResponse[2])
		body = echoResponse[2]
	}else if strings.EqualFold(req.Path, "/user-agent"){
		if uagent, exists := req.Headers["User-Agent"]; exists && len(uagent) > 0{
			contentLength = len(uagent[0])
			body = uagent[0]			
		}
	}else{
		status = HTTPStatus[404]
	}

	//send response
	rsp  := fmt.Sprintf(
		"HTTP/1.1 %s\r\n" +
		"Content-Type: %s\r\n" +
		"Content-Length: %d\r\n\r\n" +
		"%s", status, contentType, contentLength,body,
	)	
	fmt.Println("<==")
	fmt.Println(rsp)
	_, err = conn.Write([]byte(rsp))
	if err != nil{
		fmt.Printf("Error reading headers from client request %s", err.Error())
		return
	}	
}


func clean(input []byte)string{
	output := string(input[:max(len(input)-2, 0)])
	return output
}

var HTTPStatus = map[int]string{
    200:  "200 OK",
    404: "404 Not Found",
	400: "400 Bad Request",
}
type Request struct{
	Headers map[string][]string
	Path string
	Method string
	Protocol string
	Body []byte
}

func NewRequest(conn net.Conn) (*Request, error) {

	reader := bufio.NewReader(conn)	
	r := &Request{
		Headers: make(map[string][]string),
		Body : nil,
	}
	// Read request line
	reqL, err := reader.ReadBytes('\n')
	if err != nil{
		fmt.Printf("Error reading request line from client request %s", err.Error())
		return nil, err
	}
	rq := clean(reqL)
	requestLine := strings.Split(rq, " ")
	if len(requestLine) != 3{
		fmt.Printf("Invalid request line received %s\n", rq)
		return nil, err
	}
	
	r.Method, r.Path, r.Protocol = requestLine[0], requestLine[1], requestLine[2]
	
	// // Read headers
	for {
		
		h, err := reader.ReadBytes('\n')
		if err != nil{
			fmt.Printf("Error reading headers from client request %s\n", err.Error())
			return nil, err
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
		r.Headers[hdr[0]] = strings.Split(strings.TrimSpace(hdr[1]), ",")
	}

	value, exists := r.Headers["Content-Length"]
	if exists{
		cl, err := strconv.Atoi(value[0]) 
		if err == nil && cl > 0{
			r.Body = make([]byte, cl)
			_, err = io.ReadFull(reader, r.Body)
			if err != nil{
				fmt.Printf("Error reading body from client request %v", err)
			}		
		}else{
			fmt.Printf("Invalid Content-Length header received %v", err)
		}
	}	
	fmt.Println("==>")
	fmt.Println(r.Method, r.Path, r.Protocol)
	for k, v := range r.Headers{
		fmt.Printf("%s : " , k)
		for _, s := range v{
			fmt.Printf("%s", s)
		}
		fmt.Println()
	}	
	fmt.Println(string(r.Body))	
	return r, nil
}



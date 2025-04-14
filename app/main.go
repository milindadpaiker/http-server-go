package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.

	RootDir = flag.String("directory", ".", "root directory to serve files from")
	flag.Parse()
	fmt.Println("Root directory set to", *RootDir)
	fmt.Println("Starting http server...")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		fmt.Println("*********Milind")
		go handleConnection(conn)
	}

}

func clean(input []byte) string {
	output := string(input[:max(len(input)-2, 0)])
	return output
}

var RootDir *string
var HTTPStatus = map[int]string{
	200: "200 OK",
	201: "201 Created",
	404: "404 Not Found",
	400: "400 Bad Request",
	500: "500 Internal Server Error",
}

type Request struct {
	Headers  map[string][]string
	Path     string
	Method   string
	Protocol string
	Body     []byte
}

type Response struct {
	Headers map[string][]string
	Body    []byte
	Status  string
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {

		// var contentLength int64
		// contentType := "text/plain"
		// var body []byte
		// defer conn.Close()
		fmt.Println(conn.RemoteAddr().String())
		rsp := &Response{
			Headers: map[string][]string{
				"Content-Type": {"text/plain"},
			},
			Body:   nil,
			Status: "200 OK",
		}
		//Read client request
		reader := bufio.NewReader(conn)
		req, err := NewRequest(reader)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Println("Milind timeout")
				return // Timeout, close connection
			}
			if errors.Is(err, io.EOF) {
				fmt.Println("Milind EOF")
				return // Client closed connection
			}
			rsp.Status = HTTPStatus[400]
		}

		//Path based routing and response handling
		if strings.EqualFold(req.Path, "/") {
			rsp.Status = HTTPStatus[200]
		} else if strings.HasPrefix(req.Path, "/echo/") {
			// time.Sleep(10 * time.Second)
			echoResponse := strings.SplitN(req.Path, "/", 3)
			// contentLength = int64(len(echoResponse[2]))
			rsp.Body = []byte(echoResponse[2])
		} else if strings.HasPrefix(req.Path, "/files/") {
			fileReq := strings.SplitN(req.Path, "/", 3)
			fileName := fileReq[2]
			fullPath := filepath.Join(*RootDir, fileName)
			if req.Method == "GET" {
				_, err := os.Stat(fullPath)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						//return 404 if file does not exist,
						rsp.Status = HTTPStatus[404]
					} else {
						rsp.Status = HTTPStatus[400]
					}
				} else {
					//else return file content

					rsp.Headers["Content-Type"] = []string{"application/octet-stream"}
					var err error
					rsp.Body, err = os.ReadFile(fullPath)
					if err != nil {
						fmt.Printf("Failed to read file %s. Error: %s", fullPath, err.Error())
						rsp.Status = HTTPStatus[500]
					}
				}
			} else if req.Method == "POST" {
				rsp.Status = HTTPStatus[201]
				err := os.WriteFile(fullPath, req.Body, 0644)
				if err != nil {
					fmt.Printf("Failed to write file %s. Error: %s", fullPath, err.Error())
					rsp.Status = HTTPStatus[500]
				}
			}
		} else if strings.EqualFold(req.Path, "/user-agent") {
			if uagent, exists := req.Headers["User-Agent"]; exists && len(uagent) > 0 {
				// contentLength = int64(len(uagent[0]))
				rsp.Body = []byte(uagent[0])
			}
		} else {
			rsp.Status = HTTPStatus[404]
		}
		//compress response body if gzip is one of the supported client comporession for this request
		//update body, contentLength and Content-Encoding accordingly

		handleCompression(req, rsp)
		err = sendResponse(rsp, conn)
		if err != nil {
			fmt.Println("Error sending response:", err)
			return
		}

	}

}

func handleCompression(req *Request, rsp *Response) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if cmp, exists := req.Headers["Accept-Encoding"]; exists {
		for _, val := range cmp {

			if strings.TrimSpace(val) == "gzip" {
				if _, err := gzipWriter.Write(rsp.Body); err != nil {
					fmt.Println("Error writing to gzip writer:", err)
					rsp.Status = HTTPStatus[500]
					break
				}

				if err := gzipWriter.Close(); err != nil {
					fmt.Println("Error closing gzip writer:", err)
					break
				}
				rsp.Body = buf.Bytes()
				rsp.Headers["Content-Encoding"] = []string{"gzip"}
				break
			}
		}
	}
}
func sendResponse(rsp *Response, conn net.Conn) error {
	r1 := fmt.Sprintf(
		"HTTP/1.1 %s\r\n"+
			// "Content-Type: %s\r\n" +
			"Content-Length: %d\r\n", rsp.Status, len(rsp.Body),
	)

	for k, values := range rsp.Headers {
		r1 += k + ": " + strings.Join(values, ",") + "\r\n"
	}
	r1 += "\r\n"
	fmt.Println("<==")
	fmt.Println(r1)

	_, err := conn.Write([]byte(r1))
	if err != nil {
		fmt.Printf("Error sending rsp headers to client %s", err.Error())
		return err
	}
	fmt.Println(string(rsp.Body))

	_, err = conn.Write(rsp.Body)
	if err != nil {
		fmt.Printf("Error sending rsp body to client %s", err.Error())
		return err
	}
	return nil
}

func NewRequest(reader *bufio.Reader) (*Request, error) {

	r := &Request{
		Headers: make(map[string][]string),
		Body:    nil,
	}
	// Read request line
	reqL, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Printf("Error reading request line from client request %s", err.Error())
		return nil, err
	}
	rq := clean(reqL)
	requestLine := strings.Split(rq, " ")
	if len(requestLine) != 3 {
		fmt.Printf("Invalid request line received %s\n", rq)
		return nil, err
	}

	r.Method, r.Path, r.Protocol = requestLine[0], requestLine[1], requestLine[2]

	// // Read headers
	for {

		h, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("Error reading headers from client request %s\n", err.Error())
			return nil, err
		}
		if bytes.Equal(h, []byte{'\r', '\n'}) {
			break
		}
		cleanheader := strings.TrimSpace(clean(h))
		hdr := strings.SplitN(cleanheader, ":", 2)
		if len(hdr) != 2 {
			fmt.Printf("Invalid header received %s", string(h))
			continue
		}
		r.Headers[hdr[0]] = strings.Split(strings.TrimSpace(hdr[1]), ",")
	}

	value, exists := r.Headers["Content-Length"]
	if exists {
		cl, err := strconv.Atoi(value[0])
		if err == nil && cl > 0 {
			r.Body = make([]byte, cl)
			_, err = io.ReadFull(reader, r.Body)
			if err != nil {
				fmt.Printf("Error reading body from client request %v", err)
			}
		} else {
			fmt.Printf("Invalid Content-Length header received %v", err)
		}
	}
	fmt.Println("==>")
	fmt.Println(r.Method, r.Path, r.Protocol)
	for k, v := range r.Headers {
		fmt.Printf("%s : ", k)
		for _, s := range v {
			fmt.Printf("%s", s)
		}
		fmt.Println()
	}
	fmt.Println(string(r.Body))
	return r, nil
}

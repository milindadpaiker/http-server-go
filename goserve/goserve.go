package goserve

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

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

var HTTPStatus = map[int]string{
	200: "200 OK",
	201: "201 Created",
	404: "404 Not Found",
	400: "400 Bad Request",
	500: "500 Internal Server Error",
}

type Handler func(*Request, *Response)
type Server struct {
	RootDir         string
	ExactHandlers   map[string]Handler
	wildcardHandler map[string]Handler
}

func NewServer(rootDir string) *Server {
	s := &Server{
		RootDir:         rootDir,
		ExactHandlers:   make(map[string]Handler),
		wildcardHandler: make(map[string]Handler),
	}
	return s
}

func (s *Server) ListenAndServe(bindAddress string) error {
	l, err := net.Listen("tcp", bindAddress)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		fmt.Println("=======Milind =============")
		go s.handleConnection(conn)
	}
}
func (s *Server) Handle(method, path string, h Handler) {
	if strings.HasSuffix(path, "/*") {
		basepath := strings.TrimSuffix(path, "*")
		s.wildcardHandler[method+":"+basepath] = h
	}
	s.ExactHandlers[method+":"+path] = h
}

// In-built handlers
func (s *Server) defaultHandler(req *Request, rsp *Response) {
	rsp.Status = HTTPStatus[404]
}

func (s *Server) EchotHandler(req *Request, rsp *Response) {
	echoResponse := strings.SplitN(req.Path, "/", 3)
	rsp.Body = []byte(echoResponse[2])
}

func (s *Server) UserAgentHandler(req *Request, rsp *Response) {
	if strings.EqualFold(req.Path, "/user-agent") {
		if uagent, exists := req.Headers["User-Agent"]; exists && len(uagent) > 0 {
			// contentLength = int64(len(uagent[0]))
			rsp.Body = []byte(uagent[0])
		}
	}
}

func (s *Server) FileHandler(req *Request, rsp *Response) {
	fileReq := strings.SplitN(req.Path, "/", 3)
	fileName := fileReq[2]
	fullPath := filepath.Join(*&s.RootDir, fileName)
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
}

func (s *Server) getHandler(req *Request) Handler {
	p := req.Method + ":" + req.Path
	//handle exact handler first
	if Handler, exists := s.ExactHandlers[p]; exists {
		return Handler
	}
	for key, handler := range s.wildcardHandler {
		a := strings.Split(key, ":")
		method, path := a[0], a[1]
		if req.Method == method && strings.HasPrefix(req.Path, path) {
			return handler
		}
	}
	return s.defaultHandler
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {

		rsp := &Response{
			Headers: map[string][]string{
				"Content-Type": {"text/plain"},
			},
			Body:   nil,
			Status: "200 OK",
		}
		//Read client request
		reader := bufio.NewReader(conn)
		req, err := s.parseRequest(reader)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return // Timeout, close connection
			}
			if errors.Is(err, io.EOF) {
				return // Client closed connection
			}
			rsp.Status = HTTPStatus[400]
			err = s.sendResponse(rsp, conn)
			if err != nil {
				fmt.Println("Error sending response:", err)
				return
			}
		}
		handler := s.getHandler(req)
		handler(req, rsp)
		// s.processRequest(req, rsp)
		s.handleCompression(req, rsp)
		if s.shouldClose(req) {
			rsp.Headers["Connection"] = []string{"close"}
		} else {
			rsp.Headers["Connection"] = []string{"keep-alive"}
		}
		err = s.sendResponse(rsp, conn)
		if err != nil {
			fmt.Println("Error sending response:", err)
			return
		}
		if s.shouldClose(req) {
			fmt.Println("Closing...")
			return
		}

	}
}

func (s *Server) parseRequest(reader *bufio.Reader) (*Request, error) {

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

func clean(input []byte) string {
	output := string(input[:max(len(input)-2, 0)])
	return output
}

func (s *Server) shouldClose(req *Request) bool {
	if value, exists := req.Headers["Connection"]; exists {
		return strings.EqualFold(value[0], "Close")
	}
	return false
}
func (s *Server) handleCompression(req *Request, rsp *Response) {
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
func (s *Server) sendResponse(rsp *Response, conn net.Conn) error {
	r1 := fmt.Sprintf(
		"HTTP/1.1 %s\r\n"+
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

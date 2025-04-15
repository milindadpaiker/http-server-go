package main

import (
	"flag"
	"fmt"

	"github.com/milindadpaiker/http-server-go/goserve"
)

func main() {

	RootDir := flag.String("directory", ".", "root directory to serve files from")
	flag.Parse()
	s := goserve.NewServer(*RootDir)
	s.Handle("GET", "/", func(req *goserve.Request, rsp *goserve.Response) {
		rsp.Status = goserve.HTTPStatus[200]
	})
	s.Handle("GET", "/custom", func(req *goserve.Request, rsp *goserve.Response) {
		rsp.Status = goserve.HTTPStatus[200]
		rsp.Body = []byte("Custom Response")
		rsp.Headers["Custom-Header"] = []string{"ABC-Milind"}
	})
	s.Handle("GET", "/echo/*", s.EchotHandler)
	s.Handle("GET", "/files/*", s.FileHandler)
	s.Handle("POST", "/files/*", s.FileHandler)
	s.Handle("GET", "/user-agent", s.UserAgentHandler)

	s.Handle("POST", "/api/data", func(req *goserve.Request, rsp *goserve.Response) {
		// Custom logic
		rsp.Status = "201 Created"
		rsp.Body = []byte("Data received")
	})
	if err := s.ListenAndServe(":4221"); err != nil {
		fmt.Println("Server error:", err)
	}
}

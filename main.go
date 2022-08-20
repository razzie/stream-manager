package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/razzie/beepboop"
)

var (
	Port         int
	Username     string
	Password     string
	StreamTarget string
)

func init() {
	flag.IntVar(&Port, "port", 8080, "HTTP port")
	flag.StringVar(&Username, "user", "", "Username for basic http auth")
	flag.StringVar(&Password, "pass", "", "Password for basic http auth")
	flag.StringVar(&StreamTarget, "target", "rtsp://localhost", "Remote stream server to publish to (rtsp/rtsps/etc)")
	flag.Parse()

	log.SetOutput(os.Stdout)
}

func main() {
	sm := NewStreamManager(StreamTarget)

	srv := beepboop.NewServer()
	srv.AddMiddlewares(AuthMiddleware(Username, Password))
	srv.AddPages(sm.Pages()...)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", Port), srv))
}

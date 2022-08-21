package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/razzie/beepboop"
)

var (
	Port         int
	Username     string
	Password     string
	StreamTarget string
	Chroot       string
)

func init() {
	flag.IntVar(&Port, "port", 8080, "HTTP port")
	flag.StringVar(&Username, "user", "", "Username for basic http auth")
	flag.StringVar(&Password, "pass", "", "Password for basic http auth")
	flag.StringVar(&StreamTarget, "target", "rtsp://localhost", "Remote stream server to publish to (rtsp/rtsps/etc)")
	flag.StringVar(&Chroot, "chroot", "", "Chroot directory")
	flag.Parse()

	log.SetOutput(os.Stdout)

	if len(Chroot) > 0 {
		if err := syscall.Chroot(Chroot); err != nil {
			panic(err)
		}
		if err := os.Chdir("/"); err != nil {
			panic(err)
		}
	}
}

func main() {
	sm := NewStreamManager(StreamTarget)

	srv := beepboop.NewServer()
	srv.AddMiddlewares(AuthMiddleware(Username, Password))
	srv.AddPages(sm.Pages()...)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", Port), srv))
}

package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"

	"github.com/go-redis/redis/v8"
	"github.com/razzie/beepboop"
)

//go:embed favicon.png
var favicon []byte

var (
	Port         int
	Username     string
	Password     string
	StreamTarget string
	Chroot       string
	RedisConnStr string
)

func init() {
	flag.IntVar(&Port, "port", 8080, "HTTP port")
	flag.StringVar(&Username, "user", "", "Username for basic http auth")
	flag.StringVar(&Password, "pass", "", "Password for basic http auth")
	flag.StringVar(&StreamTarget, "target", "rtsp://localhost", "Remote stream server to publish to (rtsp/rtsps/etc)")
	flag.StringVar(&Chroot, "chroot", "", "Chroot directory")
	flag.StringVar(&RedisConnStr, "redis", "", "Redis connection string (redis://user:pass@host:port)")
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

	if ffmpegPath, err := exec.LookPath("ffmpeg"); err != nil {
		panic(err)
	} else {
		log.Println("ffmpeg path:", ffmpegPath)
	}

	if ffprobePath, err := exec.LookPath("ffprobe"); err != nil {
		panic(err)
	} else {
		log.Println("ffprobe path:", ffprobePath)
	}
}

func main() {
	log.Println("stream-manager start")

	var opt *redis.Options
	var err error
	if len(RedisConnStr) > 0 {
		opt, err = redis.ParseURL(RedisConnStr)
		if err != nil {
			log.Println("failed to parse redis url:", err)
		}
	}

	sm := NewStreamManager(StreamTarget, opt)

	srv := beepboop.NewServer()
	srv.FaviconPNG = favicon
	srv.AddMiddlewares(AuthMiddleware(Username, Password))
	srv.AddPages(sm.Pages()...)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", Port), srv))
}

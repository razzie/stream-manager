package main

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

var namePattern = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

type Stream struct {
	Name            string
	Source          string
	Target          string
	StartPosition   time.Duration
	VideoChannel    int
	AudioChannel    int
	SubtitleChannel int
	ReadRate        int
	runner          atomic.Pointer[StreamRunner]
}

func NewStream(name, source, target string, startpos time.Duration, video, audio, subtitle, readrate int) (*Stream, error) {
	if !namePattern.MatchString(name) {
		return nil, fmt.Errorf("invalid name: %s", name)
	}
	if len(source) == 0 {
		return nil, fmt.Errorf("no source")
	}
	if len(target) == 0 {
		return nil, fmt.Errorf("no target")
	}
	stream := &Stream{
		Name:            name,
		Source:          source,
		Target:          target,
		StartPosition:   startpos,
		VideoChannel:    video,
		AudioChannel:    audio,
		SubtitleChannel: subtitle,
		ReadRate:        readrate,
	}
	return stream, nil
}

func (stream *Stream) Start() error {
	runner := NewStreamRunner(stream)
	if !stream.runner.CompareAndSwap(nil, runner) {
		for {
			old := stream.runner.Load()
			if old.IsRunning() {
				return fmt.Errorf("stream already started")
			}
			if stream.runner.CompareAndSwap(old, runner) {
				break
			}
		}
	}
	return runner.Start()
}

func (stream *Stream) Status() string {
	if runner := stream.runner.Load(); runner != nil {
		if runner.IsRunning() {
			return "Running"
		} else if err := runner.Err(); err != nil {
			return "Error: " + err.Error()
		}
	}
	return "Stopped"
}

func (stream *Stream) Close() error {
	if runner := stream.runner.Swap(nil); runner != nil {
		runner.Close() // ignore exit status 1 error
	}
	return nil
}

type StreamRunner struct {
	Stream  *Stream
	cmd     *exec.Cmd
	ctx     context.Context
	cancel  context.CancelFunc
	errChan chan error
	errBuf  strings.Builder
	done    atomic.Bool
}

func NewStreamRunner(stream *Stream) *StreamRunner {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs(stream)...)
	return &StreamRunner{
		Stream:  stream,
		cmd:     cmd,
		ctx:     ctx,
		cancel:  cancel,
		errChan: make(chan error, 1),
	}
}

func (runner *StreamRunner) Start() error {
	runner.cmd.Stdout = &runner.errBuf
	runner.cmd.Stderr = &runner.errBuf
	if err := runner.cmd.Start(); err != nil {
		return err
	}
	go func() {
		runner.errChan <- runner.cmd.Wait()
		runner.done.Store(true)
	}()
	return nil
}

func (runner *StreamRunner) IsRunning() bool {
	return !runner.done.Load()
}

func (runner *StreamRunner) Err() error {
	if runner.done.Load() {
		errStr := runner.errBuf.String()
		if len(errStr) > 128 {
			errStr = "..." + errStr[len(errStr)-128:]
		}
		if len(errStr) > 0 {
			return fmt.Errorf("%s", errStr)
		}
	}
	return nil
}

func (runner *StreamRunner) Close() error {
	runner.cancel()
	return <-runner.errChan
}

func ffmpegArgs(stream *Stream) (args []string) {
	args = append(args,
		"-hide_banner", "-loglevel", "error",
		"-copyts", "-start_at_zero", "-preset", "ultrafast")
	readrate := float32(stream.ReadRate) / 100.
	if readrate < 1. {
		readrate = 1.
	}
	args = append(args, "-readrate", fmt.Sprint(readrate))
	if stream.StartPosition > 0 {
		startpos := fmt.Sprint(stream.StartPosition.Seconds())
		args = append(args, "-ss", startpos, "-i", stream.Source, "-ss", startpos)
	} else {
		args = append(args, "-i", stream.Source)
	}
	if stream.VideoChannel >= 0 {
		args = append(args, "-c:v", "libx264", "-map", fmt.Sprintf("0:v:%d", stream.VideoChannel))
	}
	if stream.AudioChannel >= 0 {
		args = append(args, "-c:a", "aac", "-map", fmt.Sprintf("0:a:%d", stream.AudioChannel))
	}
	if stream.SubtitleChannel >= 0 {
		escapedSource := strings.ReplaceAll(stream.Source, ":", "\\:")
		args = append(args, "-vf", fmt.Sprintf("subtitles='%s':stream_index=%d", escapedSource, stream.SubtitleChannel))
	}
	target := stream.Target
	if !strings.HasSuffix(target, "/") {
		target += "/"
	}
	args = append(args, "-f", "rtsp", "-rtsp_transport", "tcp", "-auth_type", "digest", target+stream.Name)
	return
}

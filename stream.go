package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"
)

type Stream struct {
	Name          string
	Source        string
	Target        string
	StartPosition string
	Audio         int
	Subtitle      int
	runner        atomic.Pointer[StreamRunner]
}

func NewStream(name, source, target, startpos string, audio, subs int) (*Stream, error) {
	// TODO: validate values
	stream := &Stream{
		Name:          name,
		Source:        source,
		Target:        target,
		StartPosition: startpos,
		Audio:         audio,
		Subtitle:      subs,
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
		if idx := strings.LastIndex(errStr, "\n"); idx != -1 {
			errStr = errStr[idx+1:]
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
	if len(stream.StartPosition) > 0 {
		args = append(args, "-ss", stream.StartPosition)
	}
	args = append(args,
		"-hide_banner", "-loglevel", "error",
		"-copyts", "-re",
		"-i", stream.Source,
		"-c:v", "libx264", "-preset", "ultrafast")
	if len(stream.StartPosition) > 0 {
		args = append(args, "-ss", stream.StartPosition)
	}
	if stream.Subtitle >= 0 {
		args = append(args, "-vf", fmt.Sprintf("subtitles=%s:stream_index=%d", stream.Source, stream.Subtitle))
	}
	args = append(args,
		"-c:a", "aac", "-map", fmt.Sprintf("0:a:%d", stream.Audio), "-map", "0:v:0",
		"-f", "rtsp", "-rtsp_transport", "tcp", "-auth_type", "digest",
		stream.Target+"/"+stream.Name)
	return
}

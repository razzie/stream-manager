package main

import (
	"context"
	"fmt"
	"os/exec"
	"sync/atomic"
)

type Stream struct {
	Name          string
	Source        string
	Target        string
	StartPosition string
	Audio         int
	Subtitle      int
	runner        atomic.Value
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
		return fmt.Errorf("stream already started")
	}
	return runner.Start()
}

func (stream *Stream) Status() string {
	runner := stream.runner.Load()
	if runner != nil {
		runner := runner.(*StreamRunner)
		if err := runner.ctx.Err(); err != nil {
			return "Error: " + err.Error()
		}
		return "Running"
	}
	return "Pending"
}

func (stream *Stream) Close() error {
	runner := stream.runner.Swap(nil)
	if runner != nil {
		runner := runner.(*StreamRunner)
		return runner.Close()
	}
	return nil
}

type StreamRunner struct {
	Stream *Stream
	cmd    *exec.Cmd
	ctx    context.Context
	cancel context.CancelFunc
}

func NewStreamRunner(stream *Stream) *StreamRunner {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs(stream)...)
	return &StreamRunner{
		Stream: stream,
		cmd:    cmd,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (runner *StreamRunner) Start() error {
	return runner.cmd.Start()
}

func (runner *StreamRunner) Close() error {
	runner.cancel()
	return runner.cmd.Wait()
}

func ffmpegArgs(stream *Stream) (args []string) {
	if len(stream.StartPosition) > 0 {
		args = append(args, "-ss", stream.StartPosition)
	}
	args = append(args,
		"-hide_banner", "-copyts", "-re",
		"-i", stream.Source,
		"-c:v", "libx264", "-preset", "ultrafast")
	if len(stream.StartPosition) > 0 {
		args = append(args, "-ss", stream.StartPosition)
	}
	if stream.Subtitle >= 0 {
		args = append(args, fmt.Sprintf("subtitles=%s:stream_index=%d", stream.Source, stream.Subtitle))
	}
	args = append(args,
		"-c:a", "aac", "-map", fmt.Sprintf("0:a:%d", stream.Audio),
		"-f", "rtsp", "-rtsp_transport", "tcp", "-auth_type", "digest",
		stream.Target+"/"+stream.Name)
	return
}

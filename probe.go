package main

import (
	"context"
	"os/exec"
)

func Probe(ctx context.Context, source string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", source)
	return cmd.Output()
}

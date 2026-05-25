package service

import (
	"context"
	"fmt"
	"os/exec"
)

type VoiceProcessor struct{}

func NewVoiceProcessor() *VoiceProcessor {
	return &VoiceProcessor{}
}

func (p *VoiceProcessor) CheckFFmpeg() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}
	return nil
}

func (p *VoiceProcessor) MakeSqueaky(ctx context.Context, inputPath, outputPath string) error {
	args := []string{
		"-y",
		"-i", inputPath,
		"-filter:a", "asetrate=48000*1.55,aresample=48000,atempo=0.85",
		"-c:a", "libopus",
		"-b:a", "64k",
		"-vbr", "on",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run ffmpeg: %w: %s", err, output)
	}
	return nil
}

func (p *VoiceProcessor) MakeSerious(ctx context.Context, inputPath, outputPath string) error {
	args := []string{
		"-y",
		"-i", inputPath,
		"-filter:a", "asetrate=48000*0.78,aresample=48000,atempo=1.2",
		"-c:a", "libopus",
		"-b:a", "64k",
		"-vbr", "on",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run ffmpeg: %w: %s", err, output)
	}
	return nil
}

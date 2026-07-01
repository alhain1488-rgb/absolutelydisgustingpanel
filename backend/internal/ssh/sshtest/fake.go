// Package sshtest provides a scriptable fake Runner for unit tests.
package sshtest

import (
	"context"
	"strings"
)

// Response is a canned result for a matched command.
type Response struct {
	Stdout string
	Stderr string
	Err    error
}

// FakeRunner implements ssh.Runner. It matches commands by substring and
// records everything it was asked to run/upload.
type FakeRunner struct {
	// Responses maps a substring to match against the command → response.
	Responses map[string]Response
	// Default is returned when no substring matches.
	Default Response

	Commands []string
	Uploads  []Upload
}

// Upload records a file upload.
type Upload struct {
	Path    string
	Content []byte
	Mode    uint32
}

// NewFakeRunner builds an empty FakeRunner.
func NewFakeRunner() *FakeRunner {
	return &FakeRunner{Responses: map[string]Response{}}
}

// On registers a response for commands containing substr.
func (f *FakeRunner) On(substr string, resp Response) *FakeRunner {
	f.Responses[substr] = resp
	return f
}

// Run implements ssh.Runner.
func (f *FakeRunner) Run(_ context.Context, cmd string) (string, string, error) {
	f.Commands = append(f.Commands, cmd)
	for substr, resp := range f.Responses {
		if strings.Contains(cmd, substr) {
			return resp.Stdout, resp.Stderr, resp.Err
		}
	}
	return f.Default.Stdout, f.Default.Stderr, f.Default.Err
}

// Upload implements ssh.Runner.
func (f *FakeRunner) Upload(_ context.Context, path string, content []byte, mode uint32) error {
	f.Uploads = append(f.Uploads, Upload{Path: path, Content: append([]byte(nil), content...), Mode: mode})
	return nil
}

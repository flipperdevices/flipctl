package runner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/flipperdevices/flipctl/internal/plugin"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type EventCallback func(stream, chunk string)

type Result struct {
	Argv   []string `json:"argv"`
	Stdout string   `json:"stdout"`
	Stderr string   `json:"stderr"`
	Output string   `json:"output"`
	Parsed any      `json:"parsed"`
	Error  string   `json:"error,omitempty"`
}

func Build(a plugin.App, vals map[string]string) ([]string, error) {
	v := map[string]string{}
	for _, f := range a.Fields {
		val := vals[f.ID]
		if val == "" {
			val = f.Default
		}
		if f.Required && val == "" {
			return nil, fmt.Errorf("%s required", f.ID)
		}
		if f.Pattern != "" {
			ok, _ := regexp.MatchString(f.Pattern, val)
			if !ok {
				return nil, fmt.Errorf("%s invalid", f.ID)
			}
		}
		if f.Type == "int" && val != "" {
			n, e := strconv.Atoi(val)
			if e != nil || (f.Min != 0 && n < f.Min) || (f.Max != 0 && n > f.Max) {
				return nil, fmt.Errorf("%s out of range", f.ID)
			}
		}
		v[f.ID] = val
	}
	argv := []string{a.Command.Executable}
	for _, t := range a.Command.Args {
		if strings.HasPrefix(t, "{{") && strings.HasSuffix(t, "}}") {
			argv = append(argv, v[strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{{"), "}}"))])
		} else {
			argv = append(argv, t)
		}
	}
	return argv, nil
}
func Run(ctx context.Context, a plugin.App, vals map[string]string) (Result, error) {
	return RunWithCallback(ctx, a, vals, nil)
}
func RunWithCallback(ctx context.Context, a plugin.App, vals map[string]string, cb EventCallback) (Result, error) {
	argv, e := Build(a, vals)
	if e != nil {
		return Result{}, e
	}
	to := time.Duration(a.Command.TimeoutSeconds) * time.Second
	if to == 0 {
		to = 5 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	cmd := exec.CommandContext(cctx, argv[0], argv[1:]...)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=", "LANG=C", "LC_ALL=C"}
	cmd.Dir = "."
	var stdout, stderr bytes.Buffer
	if cb == nil {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		e = cmd.Run()
	} else {
		var wg sync.WaitGroup
		outPipe, pe := cmd.StdoutPipe()
		if pe != nil {
			return Result{}, pe
		}
		errPipe, pe := cmd.StderrPipe()
		if pe != nil {
			return Result{}, pe
		}
		if e = cmd.Start(); e == nil {
			wg.Add(2)
			go captureStream(&wg, outPipe, &stdout, "stdout", cb)
			go captureStream(&wg, errPipe, &stderr, "stderr", cb)
			e = cmd.Wait()
			wg.Wait()
		}
	}
	out, errout := limit(stdout.String(), 64000), limit(stderr.String(), 16000)
	r := Result{Argv: argv, Stdout: out, Stderr: errout, Output: limit(out+errout, 64000)}
	if errors.Is(cctx.Err(), context.DeadlineExceeded) {
		r.Error = "timeout"
		return r, cctx.Err()
	}
	if errors.Is(cctx.Err(), context.Canceled) {
		r.Error = "canceled"
		return r, cctx.Err()
	}
	if e != nil {
		r.Error = e.Error()
	}
	return r, e
}
func captureStream(wg *sync.WaitGroup, r io.Reader, buf *bytes.Buffer, stream string, cb EventCallback) {
	defer wg.Done()
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 4096), 1024*1024)
	for s.Scan() {
		line := s.Text() + "\n"
		buf.WriteString(line)
		cb(stream, line)
	}
}
func limit(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

package runner

import (
	"context"
	"github.com/flipperdevices/flipctl/internal/plugin"
	"testing"
	"time"
)

func TestBuildRejectsInvalidAndUsesTemplates(t *testing.T) {
	a := plugin.App{ID: "x", Fields: []plugin.Field{{ID: "target", Required: true, Pattern: `^[a-z]+$`}}, Command: plugin.Command{Executable: "echo", Args: []string{"{{target}}"}}}
	if _, err := Build(a, map[string]string{"target": "bad;"}); err == nil {
		t.Fatal("expected invalid")
	}
	argv, err := Build(a, map[string]string{"target": "ok"})
	if err != nil || argv[1] != "ok" {
		t.Fatalf("%v %#v", err, argv)
	}
}

func TestBuildRealToolsAllowExternalTargets(t *testing.T) {
	apps, err := plugin.LoadDir("../../plugins-real")
	if err != nil {
		t.Fatal(err)
	}
	nmap, ok := plugin.ByID(apps, "network.nmap")
	if !ok {
		t.Fatal("missing nmap")
	}
	argv, err := Build(nmap, map[string]string{"target": "ya.ru"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/usr/bin/nmap", "-sT", "-Pn", "--stats-every", "1s", "-p", "80", "ya.ru"}
	if len(argv) != len(want) {
		t.Fatalf("argv %#v", argv)
	}
	for i := range want {
		if argv[i] != want[i] {
			t.Fatalf("argv %#v", argv)
		}
	}
	ping, ok := plugin.ByID(apps, "network.ping")
	if !ok {
		t.Fatal("missing ping")
	}
	argv, err = Build(ping, map[string]string{"target": "ya.ru", "count": "1"})
	if err != nil {
		t.Fatal(err)
	}
	want = []string{"/bin/ping", "-n", "-c", "1", "-W", "1", "ya.ru"}
	if len(argv) != len(want) {
		t.Fatalf("argv %#v", argv)
	}
	for i := range want {
		if argv[i] != want[i] {
			t.Fatalf("argv %#v", argv)
		}
	}
}

func TestRunWithCallbackStreamsBeforeCompletion(t *testing.T) {
	a := plugin.App{ID: "x", Command: plugin.Command{Executable: "/bin/sh", Args: []string{"-c", "printf 'one\\n'; sleep 0.2; printf 'two\\n'"}, TimeoutSeconds: 2}}
	seen := make(chan string, 2)
	start := time.Now()
	res, err := RunWithCallback(context.Background(), a, nil, func(stream, chunk string) { seen <- chunk })
	if err != nil {
		t.Fatal(err)
	}
	if res.Stdout != "one\ntwo\n" {
		t.Fatalf("bad stdout %q", res.Stdout)
	}
	select {
	case first := <-seen:
		if first != "one\n" || time.Since(start) > time.Second {
			t.Fatalf("not streamed promptly: %q", first)
		}
	default:
		t.Fatal("missing streamed callback")
	}
}

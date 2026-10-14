package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// keepLogs is how many run logs stay in ~/.iesedoble-ai/logs.
const keepLogs = 20

// fileLogger copies every message and progress step to a run log, then passes
// it on to the caller's logger.
type fileLogger struct {
	next Logger
	f    *os.File
}

func (l fileLogger) write(kind, msg string) {
	fmt.Fprintf(l.f, "%s %-5s %s\n", time.Now().Format("15:04:05"), kind, msg)
}

func (l fileLogger) Info(msg string) { l.write("info", msg); l.next.Info(msg) }
func (l fileLogger) OK(msg string)   { l.write("ok", msg); l.next.OK(msg) }
func (l fileLogger) Warn(msg string) { l.write("warn", msg); l.next.Warn(msg) }

// Step implements Progress, forwarding to the caller's logger when it shows progress.
func (l fileLogger) Step(done, total int, label string) {
	l.write("step", fmt.Sprintf("%d/%d %s", done, total, label))
	if p, ok := l.next.(Progress); ok {
		p.Step(done, total, label)
	}
}

// openRunLog starts the log of a run that writes (not dry runs). It returns
// the logger to use and a function that records the outcome and closes the
// file. When the log cannot be created, the run goes on without it.
func openRunLog(opts Options, action string, log Logger) (Logger, func(err error)) {
	if opts.DryRun {
		return log, func(error) {}
	}
	home := opts.Home
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	dir := filepath.Join(stateDirFor(home), "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return log, func(error) {}
	}
	now := time.Now()
	file := filepath.Join(dir, now.UTC().Format("2006-01-02T15-04-05.000Z")+"-"+action+".log")
	f, err := os.Create(file)
	if err != nil {
		return log, func(error) {}
	}
	models := map[string]any{}
	resolved := withDefaultModels(opts.Models)
	for _, t := range opts.Targets {
		models[t] = resolved[t]
	}
	selection, _ := json.MarshalIndent(struct {
		Targets    []string
		Components Components
		Models     map[string]any
	}{opts.Targets, opts.Components, models}, "", "  ")
	fmt.Fprintf(f, toolName+" %s %s\n%s %s/%s\n%s\n\n", Version, action,
		now.Format(time.RFC3339), runtime.GOOS, runtime.GOARCH, selection)
	pruneLogs(dir)
	fl := fileLogger{next: log, f: f}
	return fl, func(err error) {
		if err != nil {
			fl.write("error", err.Error())
		} else {
			fl.write("done", action+" finished")
		}
		_ = f.Close()
		log.Info("Log: " + file)
	}
}

// pruneLogs keeps the newest keepLogs files in dir; names sort by date.
func pruneLogs(dir string) {
	entries, _ := os.ReadDir(dir)
	var logs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			logs = append(logs, e.Name())
		}
	}
	sort.Strings(logs)
	for len(logs) > keepLogs {
		_ = os.Remove(filepath.Join(dir, logs[0]))
		logs = logs[1:]
	}
}

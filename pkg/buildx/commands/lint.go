package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/depot/cli/pkg/buildx/build"
	"github.com/docker/buildx/builder"
	"github.com/docker/buildx/util/progress"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	gateway "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/solver/pb"
	"github.com/morikuni/aec"
	"github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

var LintFailed = errors.New("linting failed")

type LintFailure int

const (
	LintFailureIgnore LintFailure = iota
	LintFailureWarn
	LintFailureError
)

func NewLintFailureMode(cliFlag string) LintFailure {
	switch strings.ToLower(cliFlag) {
	case "error":
		return LintFailureError
	case "warn":
		return LintFailureWarn
	default:
		return LintFailureIgnore
	}

}

type Linter struct {
	FailureMode LintFailure
	Clients     []*client.Client
	BuildxNodes []builder.Node

	mu     sync.Mutex
	Issues map[string][]client.VertexWarning
}

func NewLinter(failureMode LintFailure, clients []*client.Client, nodes []builder.Node) *Linter {
	return &Linter{
		FailureMode: failureMode,
		Clients:     clients,
		BuildxNodes: nodes,
		Issues:      make(map[string][]client.VertexWarning),
	}
}

func (l *Linter) Handle(ctx context.Context, target string, driverIndex int, dockerfile *build.DockerfileInputs, printer progress.Writer) error {
	if l.FailureMode == LintFailureIgnore {
		return nil
	}

	// This prevents more than one platform architecture from running linting.
	{
		l.mu.Lock()
		if _, ok := l.Issues[target]; ok {
			l.mu.Unlock()
			return nil
		}
		l.mu.Unlock()
	}

	var warnings []client.VertexWarning
	if driverIndex > len(l.Clients) {
		return nil // TODO:?
	}
	if l.Clients[driverIndex] == nil {
		return nil // TODO:?
	}
	if len(l.BuildxNodes[driverIndex].Platforms) == 0 {
		return nil // TODO:?
	}

	lintName := "[lint]"
	if target != defaultTargetName {
		lintName = fmt.Sprintf("[%s lint]", target)
	}
	dgst := digest.Canonical.FromString(identity.NewID())
	tm := time.Now()
	printer.Write(&client.SolveStatus{
		Vertexes: []*client.Vertex{
			{
				Digest:  dgst,
				Name:    lintName,
				Started: &tm,
			},
		},
	})

	output, err := NewContainer(ctx, l.Clients[driverIndex], l.BuildxNodes[driverIndex].Platforms[0], dockerfile)
	if err != nil {
		fmt.Println(err)
		// TODO? return err?
	}

	doneTm := time.Now()
	statuses := make([]*client.VertexStatus, 0, len(output.Messages))
	for _, lint := range output.Lints() {
		status := &client.VertexStatus{
			Vertex:    dgst,
			ID:        fmt.Sprintf("%s %s:%d %s: %s", strings.ToUpper(lint.Level), lint.File, lint.Line, lint.Code, lint.Message),
			Timestamp: tm,
			Started:   &tm,
			Completed: &doneTm,
		}
		statuses = append(statuses, status)
		warning := client.VertexWarning{
			Vertex: dgst,
			Level:  2,
			Short:  []byte(lint.Message),
			SourceInfo: &pb.SourceInfo{
				Filename: lint.File,
				Data:     dockerfile.Content,
			},
			Range: []*pb.Range{
				{
					Start: pb.Position{
						Line:      int32(lint.Line),
						Character: int32(lint.Column),
					},
				},
			},
			URL: fmt.Sprintf("https://github.com/hadolint/hadolint/wiki/%s", lint.Code),
		}
		warnings = append(warnings, warning)
	}
	printer.Write(&client.SolveStatus{
		Vertexes: []*client.Vertex{
			{
				Digest:    dgst,
				Name:      lintName,
				Started:   &tm,
				Completed: &doneTm,
			},
		},
		Statuses: statuses,
	})

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.Issues == nil {
		l.Issues = make(map[string][]client.VertexWarning)
	}
	l.Issues[target] = warnings

	if l.FailureMode == LintFailureError && len(warnings) > 0 {
		return LintFailed
	}

	return nil
}

func NewContainer(ctx context.Context, c *client.Client, platform ocispecs.Platform, dockerfile *build.DockerfileInputs) (CaptureOutput, error) {
	output := CaptureOutput{}
	_, err := c.Build(ctx, client.SolveOpt{}, "buildx", func(ctx context.Context, c gateway.Client) (*gateway.Result, error) {
		hadolint := llb.Image("hadolint/hadolint:latest-alpine").
			Platform(platform).
			File(
				llb.Mkfile(dockerfile.Filename, 0664, dockerfile.Content),
				llb.WithCustomName("[internal] lint"),
			)

		def, err := hadolint.Marshal(ctx, llb.Platform(platform))
		if err != nil {
			return nil, err
		}
		imgRef, err := c.Solve(ctx, gateway.SolveRequest{
			Definition: def.ToPB(),
		})
		if err != nil {
			return nil, err
		}

		containerCtx, containerCancel := context.WithCancel(ctx)
		defer containerCancel()
		bkContainer, err := c.NewContainer(containerCtx, gateway.NewContainerRequest{
			Mounts: []gateway.Mount{
				{
					Dest:      "/",
					MountType: pb.MountType_BIND,
					Ref:       imgRef.Ref,
				},
			},
			Platform: &pb.Platform{Architecture: platform.Architecture, OS: platform.OS},
		})
		if err != nil {
			return nil, err
		}

		defer func() {
			// TODO: you could return the error here if we want to fail the build because of linting errors.
			if err != nil {
				fmt.Printf("release err: %v\n", err)
			}
		}()

		proc, err := bkContainer.Start(ctx, gateway.StartRequest{
			Args:   []string{"/bin/hadolint", dockerfile.Filename, "-f", "json"},
			Stdout: &output,
		})
		if err != nil {
			_ = bkContainer.Release(ctx)
			return nil, err
		}
		_ = proc.Wait()

		output.Err = bkContainer.Release(ctx)

		return nil, nil
	}, nil)
	return output, err
}

type CaptureOutput struct {
	Messages []string
	Err      error
}

func (o *CaptureOutput) Write(p []byte) (n int, err error) {
	msg := string(p)
	msgs := strings.Split(msg, "\n")
	for i := range msgs {
		if msgs[i] == "" {
			continue
		}
		o.Messages = append(o.Messages, msgs[i])
	}

	return len(p), nil
}

func (o *CaptureOutput) Close() error {
	return nil
}

func (o *CaptureOutput) Lints() []Lint {
	var allLints []Lint
	for _, msg := range o.Messages {
		var lints []Lint
		if err := json.Unmarshal([]byte(msg), &lints); err != nil {
			continue
		}
		allLints = append(allLints, lints...)
	}
	return allLints
}

type Lint struct {
	Code    string `json:"code"`
	Column  int    `json:"column"`
	File    string `json:"file"`
	Level   string `json:"level"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

func (l *Linter) Print(w io.Writer, mode string) {
	// Copied from printWarnings with a few modifications for errors.
	if l.FailureMode == LintFailureIgnore {
		return
	}

	if mode == progress.PrinterModeQuiet {
		return
	}

	numIssues := 0
	for _, targetIssues := range l.Issues {
		numIssues += len(targetIssues)
	}
	if numIssues == 0 {
		return
	}

	fmt.Fprintf(w, "\n ")
	sb := &bytes.Buffer{}
	if numIssues == 1 {
		fmt.Fprintf(sb, "1 linter issue found")
	} else {
		fmt.Fprintf(sb, "%d linter issues found", numIssues)
	}

	color := aec.GreenF
	if l.FailureMode == LintFailureError {
		color = aec.RedF
	} else if l.FailureMode == LintFailureWarn {
		color = aec.YellowF
	}

	fmt.Fprintf(sb, ":\n")
	fmt.Fprint(w, aec.Apply(sb.String(), color))

	for target, issues := range l.Issues {
		if target == defaultTargetName {
			target = ""
		} else {
			target = fmt.Sprintf("[%s] ", target)
		}

		for _, issue := range issues {
			fmt.Fprintf(w, "%s%s:%d %s\n", target, issue.SourceInfo.Filename, issue.Range[0].Start.Line, issue.Short)

			for _, d := range issue.Detail {
				fmt.Fprintf(w, "%s\n", d)
			}
			if issue.URL != "" {
				fmt.Fprintf(w, "  More info: %s\n", issue.URL)
			}
			if issue.SourceInfo != nil && issue.Range != nil {
				Print(w, &issue, color)
			}
			fmt.Fprintf(w, "\n")

		}
	}
}

func Print(w io.Writer, issue *client.VertexWarning, color aec.ANSI) {
	si := issue.SourceInfo
	if si == nil {
		return
	}
	lines := strings.Split(string(si.Data), "\n")

	start, end, ok := getStartEndLine(issue.Range)
	if !ok {
		return
	}
	if start > len(lines) || start < 1 {
		return
	}
	if end > len(lines) {
		end = len(lines)
	}

	pad := 2
	if end == start {
		pad = 4
	}

	var p int
	for {
		if p >= pad {
			break
		}
		if start > 1 {
			start--
			p++
		}
		if end != len(lines) {
			end++
			p++
		}
		p++
	}

	fmt.Fprint(w, "\n  --------------------\n")
	for i := start; i <= end; i++ {
		pfx := "   "
		if containsLine(issue.Range, i) {
			pfx = aec.Apply(">>>", color)
		}
		fmt.Fprintf(w, "   %3d | %s %s\n", i, pfx, lines[i-1])
	}
	fmt.Fprintf(w, "  --------------------\n")
}

func containsLine(rr []*pb.Range, l int) bool {
	for _, r := range rr {
		e := r.End.Line
		if e < r.Start.Line {
			e = r.Start.Line
		}
		if r.Start.Line <= int32(l) && e >= int32(l) {
			return true
		}
	}
	return false
}

func getStartEndLine(rr []*pb.Range) (start int, end int, ok bool) {
	first := true
	for _, r := range rr {
		e := r.End.Line
		if e < r.Start.Line {
			e = r.Start.Line
		}
		if first || int(r.Start.Line) < start {
			start = int(r.Start.Line)
		}
		if int(e) > end {
			end = int(e)
		}
		first = false
	}
	return start, end, !first
}
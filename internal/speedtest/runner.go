package speedtest

import (
	"context"
	"fmt"
	"math"
	"time"

	lib "github.com/showwin/speedtest-go/speedtest"
)

// Result is a single speedtest outcome (success or failure).
type Result struct {
	OK             bool
	Error          string
	DownloadMbps   *float64
	UploadMbps     *float64
	PingMS         *float64
	JitterMS       *float64
	ServerID       string
	ServerName     string
	ServerLocation string
	TestedAt       string
}

// Runner runs a speedtest. Tests inject a mock.
type Runner interface {
	Run(ctx context.Context, serverID string) Result
}

// LibRunner uses showwin/speedtest-go against speedtest.net.
type LibRunner struct{}

func NewLibRunner() *LibRunner {
	return &LibRunner{}
}

func (r *LibRunner) Run(ctx context.Context, serverID string) Result {
	testedAt := utcNowISO()
	client := lib.New()

	var server *lib.Server
	var err error
	if serverID != "" {
		server, err = client.FetchServerByIDContext(ctx, serverID)
		if err != nil {
			return Result{OK: false, Error: err.Error(), TestedAt: testedAt}
		}
	} else {
		list, err := client.FetchServerListContext(ctx)
		if err != nil {
			return Result{OK: false, Error: err.Error(), TestedAt: testedAt}
		}
		targets, err := list.FindServer([]int{})
		if err != nil {
			return Result{OK: false, Error: err.Error(), TestedAt: testedAt}
		}
		if len(targets) == 0 {
			return Result{OK: false, Error: "no speedtest servers available", TestedAt: testedAt}
		}
		server = targets[0]
	}

	if err := server.PingTestContext(ctx, nil); err != nil {
		return failResult(server, testedAt, fmt.Sprintf("ping: %v", err))
	}
	if err := server.DownloadTestContext(ctx); err != nil {
		return failResult(server, testedAt, fmt.Sprintf("download: %v", err))
	}
	if err := server.UploadTestContext(ctx); err != nil {
		return failResult(server, testedAt, fmt.Sprintf("upload: %v", err))
	}

	dl := round1(server.DLSpeed.Mbps())
	ul := round1(server.ULSpeed.Mbps())
	ping := round1(float64(server.Latency) / float64(time.Millisecond))
	jitter := round1(float64(server.Jitter) / float64(time.Millisecond))
	if server.Context != nil {
		server.Context.Reset()
	}
	return Result{
		OK:             true,
		DownloadMbps:   &dl,
		UploadMbps:     &ul,
		PingMS:         &ping,
		JitterMS:       &jitter,
		ServerID:       server.ID,
		ServerName:     serverSponsorName(server),
		ServerLocation: serverLocation(server),
		TestedAt:       testedAt,
	}
}

func failResult(server *lib.Server, testedAt, errMsg string) Result {
	r := Result{OK: false, Error: errMsg, TestedAt: testedAt}
	if server != nil {
		r.ServerID = server.ID
		r.ServerName = serverSponsorName(server)
		r.ServerLocation = serverLocation(server)
	}
	return r
}

func serverSponsorName(s *lib.Server) string {
	if s.Sponsor != "" && s.Sponsor != "?" {
		return s.Sponsor
	}
	return s.Name
}

func serverLocation(s *lib.Server) string {
	parts := []string{}
	if s.Name != "" && s.Name != "?" {
		parts = append(parts, s.Name)
	}
	if s.Country != "" && s.Country != "?" {
		parts = append(parts, s.Country)
	}
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += ", " + parts[i]
	}
	return out
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func utcNowISO() string {
	return time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
}


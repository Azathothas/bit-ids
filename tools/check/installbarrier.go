// install-barrier - hold a lane's capture until every route of its adapter
// has installed, across the separate runners a dispatch spreads them over.
//
// CI-11. It was scripts/ci/install-barrier.sh, and it is Go because the
// Windows lane of `capture-client.yml` waits at it too and a Windows runner has
// no `sh`. The reasons the shell half gave are kept here.
//
// ⛔ WHY IT EXISTS. A run manifest says a build was acquired twice before it was
// observed (`E-MAN-03`), and `capture-client` run 37 broke that without anyone
// lying: its release lane observed from 09:36:22 to 09:37:07, on one runner,
// while the source route was still compiling on another until 09:43:40. The
// operator settled on 2026-10-14 that the workflow makes the order true rather
// than the record bending to it. `CI-09`.
//
// ⭐ THE RUNNERS SHARE NOTHING BUT THE COMMIT, so the commit is where they meet.
// Each lane posts one completed check run naming its run, its adapter and its
// route, with conclusion `success` once it installed and `failure` when it could
// not; each then reads the commit's check runs until every route of its adapter
// has arrived. The run attempt is in the name, so a barrier from an earlier
// dispatch of the same commit can never satisfy this one. ⚠ The Windows lane
// arrives under `<adapter>-windows`, so its routes meet each other and not the
// Linux lanes', whose pair is a different manifest.
//
// ⚠ IT NEVER FAILS A LANE. `wait` always ends 0 once it could run and writes its
// verdict instead: `passed`, `failed` naming the routes that could not install,
// or `timeout`. `assemble-capture` is what refuses a manifest whose order is not
// true, by the timestamps, whatever the verdict says.
//
// -- ⚠ WHERE THIS DIFFERS FROM THE SHELL HALF -------------------------------
//
//   - JSON is Go's, so the host needs no `jq`.
//   - The token reaches `curl` through a header FILE (`-H @file`), made in a
//     directory only this user can read and removed after the call, rather than
//     on `curl`'s command line, where any process on the host could list it.
//   - The bound is this process's: 20 seconds around a `curl --max-time 15`, as
//     `timeout 20` gave it.
//
// Usage:
//
//	bit-check install-barrier arrive <adapter> <route> <success|failure>
//	bit-check install-barrier wait <adapter> <routes-json> <record>
//
// Environment: GH_TOKEN, GITHUB_API_URL, GITHUB_REPOSITORY, GITHUB_SHA, and
// BARRIER_RUN, the run's id and attempt. BIT_IDS_BARRIER_TIMEOUT (seconds,
// default 1500) and BIT_IDS_BARRIER_INTERVAL (seconds, default 15) bound the
// wait; BIT_IDS_BARRIER_RETRY_WAIT (seconds, default 2, doubling) spaces an
// arrival's four attempts.
//
// Exit codes: 0 posted, or waited and wrote a verdict; 2 could not run.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func installBarrier(args []string) int {
	say := func(f string, a ...any) { fmt.Fprintf(os.Stderr, "install-barrier: "+f+"\n", a...) }
	env := map[string]string{}
	for _, k := range []string{"GH_TOKEN", "GITHUB_API_URL", "GITHUB_REPOSITORY", "GITHUB_SHA", "BARRIER_RUN"} {
		env[k] = os.Getenv(k)
		if env[k] == "" {
			say("%s is not set", k)
			return 2
		}
	}
	curl, err := exec.LookPath("curl")
	if err != nil {
		say("curl is not on this host")
		return 2
	}
	api := env["GITHUB_API_URL"] + "/repos/" + env["GITHUB_REPOSITORY"]
	nameOf := func(adapter, route string) string {
		return fmt.Sprintf("install-barrier (%s %s-%s)", env["BARRIER_RUN"], adapter, route)
	}
	if len(args) < 1 {
		say("a mode is required, arrive or wait")
		return 2
	}
	b := barrierAPI{curl: curl, token: env["GH_TOKEN"]}

	switch args[0] {
	case "arrive":
		if len(args) != 4 {
			say("arrive takes <adapter> <route> <success|failure>")
			return 2
		}
		adapter, route, conclusion := args[1], args[2], args[3]
		if conclusion != "success" && conclusion != "failure" {
			say("a conclusion is success or failure, not [%s]", conclusion)
			return 2
		}
		body, _ := json.Marshal(map[string]any{
			"name": nameOf(adapter, route), "head_sha": env["GITHUB_SHA"],
			"status": "completed", "conclusion": conclusion,
			"output": map[string]string{"title": "install " + conclusion, "summary": "install " + conclusion},
		})
		// ⛔ ONE ATTEMPT WAS A LANE LOST FOR THE WHOLE RUN. capture-client run
		// 39588980862's Linux source lane installed, posted once, got no `id` back
		// and said only that; both Linux lanes then waited the full 1500 seconds
		// and recorded `timeout`. So the post is tried four times, 2, 4 and 8
		// seconds apart, and each refusal prints what GitHub said. ⚠ A retry after
		// a post that landed but lost its answer posts the same name twice, which
		// `wait` reads as one: it takes the first. `CLIENT-06`.
		wait := envSeconds("BIT_IDS_BARRIER_RETRY_WAIT", 2)
		for attempt := 1; ; attempt++ {
			out, _ := b.call("POST", api+"/check-runs", body)
			var created struct {
				ID      json.Number `json:"id"`
				Message string      `json:"message"`
			}
			if json.Unmarshal(out, &created) == nil && created.ID != "" {
				break
			}
			said := created.Message
			if said == "" {
				said = "no answer GitHub could be read from"
			}
			say("the check run for %s %s was not created on attempt %d: %s", adapter, route, attempt, said)
			if attempt == 4 {
				return 2
			}
			time.Sleep(time.Duration(wait) * time.Second)
			wait *= 2
		}
		fmt.Printf("install-barrier: %s arrived, %s\n", nameOf(adapter, route), conclusion)
		return 0

	case "wait":
		if len(args) != 4 {
			say("wait takes <adapter> <routes-json> <record>")
			return 2
		}
		adapter, record := args[1], args[3]
		var routes []string
		if json.Unmarshal([]byte(args[2]), &routes) != nil || len(routes) == 0 {
			say("no routes in [%s]", args[2])
			return 2
		}
		timeout := envSeconds("BIT_IDS_BARRIER_TIMEOUT", 1500)
		interval := envSeconds("BIT_IDS_BARRIER_INTERVAL", 15)

		start := time.Now()
		verdict := "timeout"
		var failed, missing []string
		for {
			arrived := 0
			failed, missing = nil, nil
			for _, route := range routes {
				q := uriEscape(nameOf(adapter, route))
				out, _ := b.call("GET", api+"/commits/"+env["GITHUB_SHA"]+"/check-runs?check_name="+q+"&per_page=10", nil)
				var runs struct {
					CheckRuns []struct {
						Conclusion *string `json:"conclusion"`
					} `json:"check_runs"`
				}
				c := ""
				if json.Unmarshal(out, &runs) == nil && len(runs.CheckRuns) > 0 && runs.CheckRuns[0].Conclusion != nil {
					c = *runs.CheckRuns[0].Conclusion
				}
				switch c {
				case "success":
					arrived++
				case "failure":
					failed = append(failed, route)
				default:
					missing = append(missing, route)
				}
			}
			if len(failed) > 0 {
				verdict = "failed"
				break
			}
			if len(missing) == 0 {
				verdict = "passed"
				break
			}
			if int(time.Since(start)/time.Second) >= timeout {
				break
			}
			time.Sleep(time.Duration(interval) * time.Second)
		}
		text := fmt.Sprintf("bit-ids/install-barrier/1\nrun=%s\nadapter=%s\nroutes=%s\nverdict=%s\nfailed_routes=%s\nmissing_routes=%s\nwaited_seconds=%d\nended_at=%s\n",
			env["BARRIER_RUN"], adapter, strings.Join(routes, ","), verdict, strings.Join(failed, " "),
			strings.Join(missing, " "), int(time.Since(start)/time.Second), utcNow())
		if err := os.WriteFile(record, []byte(text), 0o644); err != nil {
			say("cannot write %s", record)
			return 2
		}
		fmt.Print(text)
		return 0
	}
	say("unknown mode [%s]", args[0])
	return 2
}

type barrierAPI struct{ curl, token string }

// call is one request, bounded, with the token in a header file and the body,
// when there is one, in another.
func (b barrierAPI) call(method, url string, body []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "install-barrier-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	headers := filepath.Join(dir, "headers")
	if err := os.WriteFile(headers, []byte("Authorization: Bearer "+b.token+"\nAccept: application/vnd.github+json\n"), 0o600); err != nil {
		return nil, err
	}
	args := []string{"-sS", "--max-time", "15", "-X", method, url, "-H", "@" + headers}
	if body != nil {
		bodyFile := filepath.Join(dir, "body.json")
		if err := os.WriteFile(bodyFile, body, 0o600); err != nil {
			return nil, err
		}
		args = append(args, "--data-binary", "@"+bodyFile)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, b.curl, args...)
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	return out.Bytes(), err
}

// uriEscape is jq's `@uri`: every byte but the unreserved ones percent-encoded,
// so a space is `%20` and never `+`.
func uriEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.IndexByte("-_.~", c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

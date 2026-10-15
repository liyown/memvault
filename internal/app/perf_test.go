package app

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"testing"
	"time"
)

// Reproducible local index workload; separate from archive/recovery acceptance tests.
// MEMVAULT_PERF=1 go test ./internal/app -run TestMillionMessageSearch -v -timeout 20m
func TestMillionMessageSearch(t *testing.T) {
	if os.Getenv("MEMVAULT_PERF") != "1" {
		t.Skip("opt-in: one million indexed messages")
	}
	e := testEngine(t)
	const sessions = 10000
	const perSession = 100
	started := time.Now()
	for batch := 0; batch < sessions; batch += 100 {
		tx, err := e.DB.Begin()
		must(t, err)
		ss, err := tx.Prepare("INSERT INTO sessions VALUES(?,?,?,?,?,?,?)")
		must(t, err)
		ms, err := tx.Prepare("INSERT INTO messages VALUES(?,?,?,?,?,?,?,?,?,?)")
		must(t, err)
		fs, err := tx.Prepare("INSERT INTO search VALUES(?,?,?,?,?,?,?,?,?)")
		must(t, err)
		for i := batch; i < batch+100; i++ {
			id := fmt.Sprintf("%032x", i+1)
			title := fmt.Sprintf("会话 %d", i)
			s := Session{ID: id, Agent: "codex", NativeID: id, Title: title, Updated: "2026-10-15T00:00:00Z"}
			_, err = ss.Exec(id, s.Agent, id, title, "", s.Updated, encoded(s))
			must(t, err)
			for j := 0; j < perSession; j++ {
				mid := fmt.Sprintf("m-%d", j)
				topic := []string{"记忆归档 archive", "旅行计划 travel", "软件开发 software", "阅读笔记 reading", "音乐创作 music", "健康习惯 habit", "学习英语 language", "财务记录 finance", "设计讨论 design", "生活安排 daily"}[i%10]
				text := fmt.Sprintf("%s：用户第 %d 次讨论。conversation memory portable 本地保存与跨平台接续；记录结论和下一步。reference%04d", topic, j, i%1000)
				m := Message{ID: mid, Role: "user", Text: text, Order: int64(j), Complete: true}
				_, err = ms.Exec(id, mid, hash(encoded(m)), j, m.Role, text, "2026-10-15T00:00:00Z", 0, 1, encoded(m))
				must(t, err)
				_, err = fs.Exec(terms(text), text, id, "", mid, title, "codex", "user", "2026-10-15T00:00:00Z")
				must(t, err)
			}
		}
		ss.Close()
		ms.Close()
		fs.Close()
		must(t, tx.Commit())
		if batch%2000 == 0 {
			t.Logf("indexed %d messages", (batch+100)*perSession)
		}
	}
	t.Logf("dataset: %d sessions, %d messages, seed %s; %s/%s CPUs=%d", sessions, sessions*perSession, time.Since(started), runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	var durations []time.Duration
	for round := 0; round < 20; round++ {
		for _, q := range []string{"记忆归档", "portable", "reference0042", "软件开发 software", "本地保存", "design"} {
			start := time.Now()
			hits, err := e.Search(SearchQuery{Query: q, Limit: 30})
			must(t, err)
			if len(hits) != 30 {
				t.Fatal("unexpected result count", q, len(hits))
			}
			d := time.Since(start)
			durations = append(durations, d)
			t.Logf("search %q: %s", q, d)
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95 := durations[(len(durations)*95-1)/100]
	start := time.Now()
	list, total, err := e.Sessions("", 0, 50)
	must(t, err)
	if len(list) != 50 || total != sessions {
		t.Fatal("pagination incorrect")
	}
	_, err = e.Session(list[0].ID, 0, 100)
	must(t, err)
	t.Logf("hot search samples=%d P95=%s; list + first 100 messages=%s", len(durations), p95, time.Since(start))
	done := make(chan error, 1)
	go func() {
		for n := 0; n < 20; n++ {
			s := Session{ID: fmt.Sprintf("import-%d", n), Agent: "claude", NativeID: fmt.Sprint(n), Title: "并发导入", Updated: utc()}
			for i := 0; i < 100; i++ {
				s.Messages = append(s.Messages, Message{ID: fmt.Sprint(i), Role: "user", Text: "portable archive 并发导入保持搜索可用", Order: int64(i), Complete: true})
			}
			if err := e.captureSession(s); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	var concurrent []time.Duration
	for i := 0; i < 120; i++ {
		start := time.Now()
		_, err = e.Search(SearchQuery{Query: "portable", Limit: 30})
		must(t, err)
		concurrent = append(concurrent, time.Since(start))
	}
	must(t, <-done)
	sort.Slice(concurrent, func(i, j int) bool { return concurrent[i] < concurrent[j] })
	t.Logf("search during 2000-message import P95=%s", concurrent[(len(concurrent)*95-1)/100])

	if p95 > 500*time.Millisecond {
		t.Errorf("release target missed: search P95=%s exceeds 500ms", p95)
	}
	if receipt := os.Getenv("MEMVAULT_PERF_BROWSER_RECEIPT"); receipt != "" {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		must(t, err)
		server := &http.Server{Handler: NewServer(e, ln.Addr().String()).Handler()}
		go server.Serve(ln)
		defer server.Close()
		must(t, atomicWrite(receipt+".connection", encoded(object{"url": "http://" + ln.Addr().String(), "token": e.Config.Token}), 0600))
		deadline := time.Now().Add(60 * time.Second)
		verified := false
		for time.Now().Before(deadline) {
			b, err := os.ReadFile(receipt)
			if err == nil {
				var result struct {
					InteractiveMS float64 `json:"interactive_ms"`
				}
				must(t, json.Unmarshal(b, &result))
				t.Logf("million-message UI interactive: %.1f ms", result.InteractiveMS)
				if result.InteractiveMS > 3000 {
					t.Error("release target missed: UI >3s")
				}
				verified = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !verified {
			t.Error("browser measurement receipt not received")
		}
	}
}

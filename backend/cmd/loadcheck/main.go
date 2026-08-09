// Command loadcheck — WS connection load harness (M3 criterion: 100
// concurrent WebSocket connections stable). Each connection gets its own
// interview + ticket (single-active-connection rule), connects, expects
// interview.start + question, pings/pongs, disconnects. No LLM involved
// (no answers) — this exercises the WS/session layer, not the LLM path.
//
// Usage:
//
//	TEST_DATABASE_URL=postgres://intivai_app:intivai_app@localhost:5433/intivai?sslmode=disable \
//	  BASE_URL=http://localhost:8081 CONNS=100 go run ./cmd/loadcheck
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
	"net"
	"net/url"
)

type connResult struct {
	ok      bool
	stage   string
	latency time.Duration
}

func main() {
	conns := 100
	if v := os.Getenv("CONNS"); v != "" {
		fmt.Sscanf(v, "%d", &conns)
	}
	base := os.Getenv("BASE_URL")
	if base == "" {
		base = "http://localhost:8081"
	}
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		log.Fatal("TEST_DATABASE_URL required")
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}

	orgSlug := "load" + time.Now().Format("150405")
	fmt.Println("setup: register")
	org, adminEmail, adminPassword, err := registerOrg(base, orgSlug)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("setup: login")
	token, err := login(base, orgSlug, adminEmail, adminPassword)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("setup: seed")
	appID := seedApp(ctx, pool, org)
	fmt.Println("setup: done")

	start := time.Now()
	var wg sync.WaitGroup
	results := make(chan connResult, conns)
	for i := 0; i < conns; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results <- runConn(base, token, org, appID, i)
		}(i)
	}
	wg.Wait()
	close(results)

	var okCount, failCount int
	var total time.Duration
	for r := range results {
		total += r.latency
		if r.ok {
			okCount++
		} else {
			failCount++
			log.Printf("FAIL conn stage=%s", r.stage)
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("connections: ok=%d fail=%d total=%d\n", okCount, failCount, conns)
	fmt.Printf("elapsed=%s avg_conn=%s\n", elapsed, total/time.Duration(conns))
	if failCount > 0 || okCount != conns {
		log.Fatalf("load check FAILED: %d/%d connections failed", failCount, conns)
	}
	fmt.Println("LOAD CHECK PASSED: 100 concurrent WS connections stable")
}

func runConn(base, token, org string, appID uuid.UUID, i int) connResult {
	connStart := time.Now()
	stage := "create"
	ivID, invite, err := createInterview(base, token, appID)
	if err != nil {
		return connResult{stage: stage, latency: time.Since(connStart)}
	}
	stage = "ticket"
	ticket, err := issueTicket(base, ivID, invite)
	if err != nil {
		return connResult{stage: stage, latency: time.Since(connStart)}
	}
	stage = "ws"
	wsURL := "ws" + strings.TrimPrefix(base, "http") + "/api/v1/candidate/interviews/" + ivID.String() + "/chat"
	log.Printf("dial %s ticketlen=%d", wsURL, len(ticket))
	h := http.Header{
		"Authorization": {"Bearer " + ticket},
		"Origin":        {"http://localhost:3000"}, // matches INTIVAI_ALLOWED_ORIGINS
	}
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, h)
	if err != nil {
		if resp != nil {
			buf := make([]byte, 128)
			n, _ := resp.Body.Read(buf)
			log.Printf("dial %d %v body=%s", resp.StatusCode, err, string(buf[:n]))
			rawProbe(wsURL, ticket)
		} else {
			log.Printf("dial err=%v", err)
		}
		return connResult{stage: stage, latency: time.Since(connStart)}
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	stage = "start"
	if _, err := readFrame(conn); err != nil {
		return connResult{stage: stage, latency: time.Since(connStart)}
	}
	stage = "question"
	if _, err := readFrame(conn); err != nil {
		return connResult{stage: stage, latency: time.Since(connStart)}
	}
	stage = "ping"
	if err := conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
		return connResult{stage: stage, latency: time.Since(connStart)}
	}
	stage = "pong"
	f, err := readFrame(conn)
	if err != nil || f["type"] != "pong" {
		return connResult{stage: stage, latency: time.Since(connStart)}
	}
	return connResult{ok: true, latency: time.Since(connStart)}
}

func readFrame(conn *websocket.Conn) (map[string]any, error) {
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// --- API helpers ---

func registerOrg(base, slug string) (string, string, string, error) {
	email := "admin@" + slug + ".io"
	password := "loadsecret123"
	body := fmt.Sprintf(`{"name":"Load Test","slug":%q,"admin_email":%q,"admin_password":%q}`, slug, email, password)
	resp, err := http.Post(base+"/api/v1/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return "", "", "", fmt.Errorf("register status %d", resp.StatusCode)
	}
	var out struct {
		Data struct {
			OrgID string `json:"org_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", "", err
	}
	return out.Data.OrgID, email, password, nil
}

func login(base, slug, email, password string) (string, error) {
	body := fmt.Sprintf(`{"org_slug":%q,"email":%q,"password":%q}`, slug, email, password)
	resp, err := http.Post(base+"/api/v1/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Data.Token == "" {
		return "", fmt.Errorf("login: empty token (status %d)", resp.StatusCode)
	}
	return out.Data.Token, nil
}

func createInterview(base, token string, appID uuid.UUID) (uuid.UUID, string, error) {
	body := fmt.Sprintf(`{"application_id":%q,"question_count":2}`, appID.String())
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/interviews", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return uuid.Nil, "", err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			InterviewID uuid.UUID `json:"interview_id"`
			Token       string    `json:"invitation_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return uuid.Nil, "", err
	}
	if resp.StatusCode != 201 || out.Data.InterviewID == uuid.Nil {
		return uuid.Nil, "", fmt.Errorf("create interview status %d", resp.StatusCode)
	}
	return out.Data.InterviewID, out.Data.Token, nil
}

func issueTicket(base string, ivID uuid.UUID, invite string) (string, error) {
	body := fmt.Sprintf(`{"interview_id":%q,"invitation_token":%q}`, ivID.String(), invite)
	resp, err := http.Post(base+"/api/v1/candidate/interviews/"+ivID.String()+"/ticket", "application/json", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Ticket string `json:"ticket"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Data.Ticket == "" {
		return "", fmt.Errorf("ticket empty (status %d)", resp.StatusCode)
	}
	return out.Data.Ticket, nil
}

// seedApp inserts job + candidate + passed application for the org (bypasses
// the CV pipeline — the harness targets the WS layer).
func seedApp(ctx context.Context, pool *gorm.DB, org string) uuid.UUID {
	jobID, candID, appID := uuid.New(), uuid.New(), uuid.New()
	err := db.RunInTx(ctx, pool, org, func(tctx context.Context) error {
		tx, _ := db.TxFrom(tctx)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO jobs (id, org_id, title, description, status, created_at) VALUES ($1,$2,$3,$4,'active',NOW())`, []any{jobID, org, "Go Engineer", "Go"}},
			{`INSERT INTO candidates (id, org_id, name, email, status, created_at) VALUES ($1,$2,$3,$4,'extracted',NOW())`, []any{candID, org, "Load", "load@x.io"}},
			{`INSERT INTO applications (id, org_id, candidate_id, job_id, status, cv_score, passed_screening, created_at) VALUES ($1,$2,$3,$4,'passed',80,true,NOW())`, []any{appID, org, candID, jobID}},
		} {
			if err := tx.Exec(q.sql, q.args...).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return appID
}

// rawProbe sends a hand-written upgrade request (no gorilla) and prints the
// raw response — isolates library-vs-server issues.
func rawProbe(wsURL, ticket string) {
	u, err := url.Parse(wsURL)
	if err != nil {
		log.Printf("raw: %v", err)
		return
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	conn, err := net.DialTimeout("tcp", host, 3*time.Second)
	if err != nil {
		log.Printf("raw dial: %v", err)
		return
	}
	defer conn.Close()
	req := "GET " + u.RequestURI() + " HTTP/1.1\r\n" +
		"Host: " + u.Host + "\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: websocket\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Authorization: Bearer " + ticket + "\r\n\r\n"
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte(req)); err != nil {
		log.Printf("raw write: %v", err)
		return
	}
	buf := make([]byte, 512)
	n, _ := conn.Read(buf)
	log.Printf("RAW RESPONSE:\n%s", string(buf[:n]))
}

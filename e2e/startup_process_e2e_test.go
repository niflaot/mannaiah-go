package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestStartupProcessE2E verifies full startup behavior by running the API command as an external process.
func TestStartupProcessE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("resolve repository root")
	repoRoot := resolveRepositoryRoot(t)

	tracer.Step("reserve startup port")
	port := reserveFreePort(t)
	address := fmt.Sprintf("127.0.0.1:%d", port)

	tracer.Step("build startup process command")
	command := exec.Command("go", "run", "./module/core/cmd/api")
	command.Dir = repoRoot
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Env = append(os.Environ(),
		fmt.Sprintf("CORE_HOST=%s", "127.0.0.1"),
		fmt.Sprintf("CORE_PORT=%d", port),
		"CORE_ENVIRONMENT=development",
		"LOG_FORMAT=json",
		"LOG_LEVEL=info",
		"DB_DRIVER=sqlite",
		"DB_DSN=file::memory:?cache=shared",
		"STORAGE_ENABLED=true",
		"STORAGE_ENDPOINT=http://127.0.0.1:9000",
		"STORAGE_REGION=us-east-1",
		"STORAGE_BUCKET_NAME=mannaiah-e2e",
		"LOGTO_ISSUER=https://issuer.example",
		"LOGTO_AUDIENCE=https://api.mannaiah.e2e",
		"DEV_AUTH_TOKEN=dev-bypass-token",
		"DEV_AUTH_SCOPE=contacts:manage assets:create",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	tracer.Step("start startup process")
	if err := command.Start(); err != nil {
		t.Fatalf("command.Start() error = %v", err)
	}

	tracer.Step("wait for status endpoint")
	waitForEndpointReady(t, "http://"+address+"/status")

	client := &http.Client{Timeout: 2 * time.Second}

	tracer.Step("request status endpoint")
	statusCode, statusBody := doJSONHTTP(t, client, http.MethodGet, "http://"+address+"/status", "", nil)
	if statusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", statusCode, http.StatusOK)
	}
	if statusBody["status"] != "ok" {
		t.Fatalf("status payload = %v, want %q", statusBody["status"], "ok")
	}

	tracer.Step("request aggregated openapi endpoint")
	openapiCode, openapiBody := doJSONHTTP(t, client, http.MethodGet, "http://"+address+"/openapi.json", "", nil)
	if openapiCode != http.StatusOK {
		t.Fatalf("openapi code = %d, want %d", openapiCode, http.StatusOK)
	}
	if openapiBody["openapi"] == nil {
		t.Fatalf("expected openapi field in response")
	}
	paths, ok := openapiBody["paths"].(map[string]any)
	if !ok || paths["/woo/sync/contacts"] == nil {
		t.Fatalf("expected /woo/sync/contacts path in aggregated openapi")
	}
	if paths["/products"] == nil || paths["/products/{id}"] == nil {
		t.Fatalf("expected /products paths in aggregated openapi")
	}
	if paths["/variations"] == nil || paths["/variations/{id}"] == nil {
		t.Fatalf("expected /variations paths in aggregated openapi")
	}
	if paths["/assets"] == nil || paths["/assets/{id}"] == nil {
		t.Fatalf("expected /assets paths in aggregated openapi")
	}
	if paths["/check-auth"] == nil {
		t.Fatalf("expected /check-auth path in aggregated openapi")
	}

	tracer.Step("create contact through development auth bypass")
	createCode, createBody := doJSONHTTP(t, client, http.MethodPost, "http://"+address+"/contacts", "dev-bypass-token", map[string]any{
		"email":     "process@example.com",
		"legalName": "Process Inc",
	})
	if createCode != http.StatusCreated {
		t.Fatalf("create code = %d, want %d; stderr=%s", createCode, http.StatusCreated, stderr.String())
	}
	contactID, _ := createBody["id"].(string)
	if strings.TrimSpace(contactID) == "" {
		t.Fatalf("expected contact id in create response")
	}

	tracer.Step("list contacts through development auth bypass")
	listCode, listBody := doJSONHTTP(t, client, http.MethodGet, "http://"+address+"/contacts?page=1&limit=10", "dev-bypass-token", nil)
	if listCode != http.StatusOK {
		t.Fatalf("list code = %d, want %d", listCode, http.StatusOK)
	}
	if listBody["meta"] == nil {
		t.Fatalf("expected list meta payload")
	}

	tracer.Step("shutdown startup process")
	if err := syscall.Kill(-command.Process.Pid, syscall.SIGINT); err != nil {
		t.Fatalf("syscall.Kill(SIGINT) error = %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- command.Wait()
	}()

	select {
	case err := <-waitDone:
		if err != nil && !isExpectedInterruptExit(err) {
			t.Fatalf("command.Wait() error = %v; stdout=%s; stderr=%s", err, stdout.String(), stderr.String())
		}
	case <-time.After(8 * time.Second):
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		t.Fatalf("startup process shutdown timeout")
	}

	tracer.Step("assert startup process trace logs")
	tracer.AssertStepCount(10)
}

// TestStartupProcessRequiresStorage verifies startup fails fast when storage is disabled.
func TestStartupProcessRequiresStorage(t *testing.T) {
	tracer := newStepTracer(t)
	repoRoot := resolveRepositoryRoot(t)

	command := exec.Command("go", "run", "./module/core/cmd/api")
	command.Dir = repoRoot
	command.Env = append(os.Environ(),
		"CORE_HOST=127.0.0.1",
		"CORE_PORT=8199",
		"CORE_ENVIRONMENT=development",
		"LOG_FORMAT=json",
		"LOG_LEVEL=info",
		"DB_DRIVER=sqlite",
		"DB_DSN=file::memory:?cache=shared",
		"STORAGE_ENABLED=false",
		"LOGTO_ISSUER=https://issuer.example",
		"LOGTO_AUDIENCE=https://api.mannaiah.e2e",
		"DEV_AUTH_TOKEN=dev-bypass-token",
		"DEV_AUTH_SCOPE=contacts:manage",
	)

	var stderr bytes.Buffer
	command.Stderr = &stderr

	tracer.Step("start process without storage")
	if err := command.Start(); err != nil {
		t.Fatalf("command.Start() error = %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- command.Wait()
	}()

	select {
	case err := <-waitDone:
		if err == nil {
			t.Fatalf("expected startup failure when storage is disabled")
		}
		if !strings.Contains(strings.ToLower(stderr.String()), "storage is mandatory") {
			t.Fatalf("stderr = %q, want storage mandatory error", stderr.String())
		}
	case <-time.After(8 * time.Second):
		_ = command.Process.Kill()
		t.Fatalf("startup did not fail within timeout")
	}
}

// resolveRepositoryRoot resolves the repository root path from the root E2E package directory.
func resolveRepositoryRoot(t *testing.T) string {
	t.Helper()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	return filepath.Dir(workingDirectory)
}

// reserveFreePort reserves an available local TCP port and returns it.
func reserveFreePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	address := listener.Addr().(*net.TCPAddr)
	return address.Port
}

// waitForEndpointReady blocks until the endpoint responds with HTTP 200.
func waitForEndpointReady(t *testing.T, url string) {
	t.Helper()

	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(8 * time.Second)

	for time.Now().Before(deadline) {
		response, err := client.Get(url)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(120 * time.Millisecond)
	}

	t.Fatalf("timeout waiting for %s", url)
}

// doJSONHTTP executes HTTP requests and decodes JSON payload responses.
func doJSONHTTP(t *testing.T, client *http.Client, method string, url string, token string, body map[string]any) (int, map[string]any) {
	t.Helper()

	requestPayload := []byte(nil)
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		requestPayload = encoded
	}

	request, err := http.NewRequest(method, url, bytes.NewReader(requestPayload))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("client.Do() error = %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	result := map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("json.NewDecoder().Decode() error = %v", err)
	}

	return response.StatusCode, result
}

// isExpectedInterruptExit reports whether process exits are expected after an interrupt signal.
func isExpectedInterruptExit(err error) bool {
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}

	if status, ok := exitError.ProcessState.Sys().(syscall.WaitStatus); ok {
		if status.Signal() == syscall.SIGINT || status.Signal() == syscall.SIGTERM {
			return true
		}
		if status.ExitStatus() == 1 {
			return true
		}
	}

	return false
}

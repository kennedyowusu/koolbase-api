package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func denoPath() string {
	if p := os.Getenv("DENO_PATH"); p != "" {
		return p
	}
	return "/root/.deno/bin/deno"
}

const functionsDir = "/var/koolbase/functions"

type ExecutionInput struct {
	Request  map[string]interface{} `json:"request"`
	Env      map[string]string      `json:"env"`
	DB       DBContext               `json:"db"`
	TestMode bool                   `json:"test_mode"`
}

type DBContext struct {
	ProjectID string `json:"project_id"`
	APIKey    string `json:"api_key"`
	BaseURL   string `json:"base_url"`
}

type ExecutionResult struct {
	Status     int
	Body       map[string]interface{}
	Output     string
	Error      string
	DurationMs int
}

// SyncFunctionToDisk writes function code to disk at deploy time
func SyncFunctionToDisk(fn *Function) error {
	dir := filepath.Join(functionsDir, fn.ProjectID, fn.Name, fmt.Sprintf("v%d", fn.Version))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	wrapper := buildWrapper(fn.Code)
	return os.WriteFile(filepath.Join(dir, "index.ts"), []byte(wrapper), 0644)
}

func FunctionFilePath(fn *Function) string {
	return filepath.Join(functionsDir, fn.ProjectID, fn.Name,
		fmt.Sprintf("v%d", fn.Version), "index.ts")
}

// buildWrapper wraps user code with Koolbase runtime context.
// Contract: developer must export `async function handler(ctx) { ... }`
func buildWrapper(userCode string) string {
	return fmt.Sprintf(`
// ── Koolbase Function Runtime ──────────────────────────────────────
const __ctxRaw = Deno.env.get("__KOOLBASE_CTX") ?? "{}";
const __ctxData = JSON.parse(__ctxRaw);
const __testMode = __ctxData.test_mode === true;

const __realDB = {
  insert: async (collection, data) => {
    const res = await fetch(__ctxData.db.base_url + "/v1/sdk/db/insert", {
      method: "POST",
      headers: { "Content-Type": "application/json", "x-api-key": __ctxData.db.api_key },
      body: JSON.stringify({ collection, data }),
    });
    return res.json();
  },
  find: async (collection, filters = {}, limit = 20) => {
    const res = await fetch(__ctxData.db.base_url + "/v1/sdk/db/query", {
      method: "POST",
      headers: { "Content-Type": "application/json", "x-api-key": __ctxData.db.api_key },
      body: JSON.stringify({ collection, filters, limit }),
    });
    return res.json();
  },
  update: async (id, data) => {
    const res = await fetch(__ctxData.db.base_url + "/v1/sdk/db/records/" + id, {
      method: "PATCH",
      headers: { "Content-Type": "application/json", "x-api-key": __ctxData.db.api_key },
      body: JSON.stringify({ data }),
    });
    return res.json();
  },
  delete: async (id) => {
    await fetch(__ctxData.db.base_url + "/v1/sdk/db/records/" + id, {
      method: "DELETE",
      headers: { "x-api-key": __ctxData.db.api_key },
    });
  },
};

const __mockDB = {
  insert: async (collection, data) => ({ __test: true, simulated: "insert", collection, data }),
  find: async (collection, filters, limit) => ({ __test: true, simulated: "find", collection, records: [] }),
  update: async (id, data) => ({ __test: true, simulated: "update", id, data }),
  delete: async (id) => ({ __test: true, simulated: "delete", id }),
};

const ctx = {
  request: __ctxData.request ?? {},
  env: __ctxData.env ?? {},
  db: __testMode ? __mockDB : __realDB,
};
// ── End Runtime ────────────────────────────────────────────────────

%s

// Execute — requires exported handler function
if (typeof handler !== "function") {
  console.error("Koolbase: function must export 'async function handler(ctx) { ... }'");
  Deno.exit(1);
}

const __result = await handler(ctx);
const __output = __result !== undefined ? JSON.stringify(__result) : JSON.stringify({ ok: true });
console.log("__KOOLBASE_RESULT__" + __output);
`, userCode)
}

// Execute runs an already-deployed function from disk
func Execute(fn *Function, input ExecutionInput) *ExecutionResult {
	start := time.Now()
	result := &ExecutionResult{}

	var filePath string
	if input.TestMode {
		// Write temp file with mock DB for test execution
		tmpDir := filepath.Join(functionsDir, fn.ProjectID, fn.Name, fmt.Sprintf("v%d-test", fn.Version))
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			result.Error = "failed to create test dir: " + err.Error()
			result.Status = 500
			return result
		}
		tmpPath := filepath.Join(tmpDir, "index.ts")
		if err := os.WriteFile(tmpPath, []byte(buildWrapper(fn.Code)), 0644); err != nil {
			result.Error = "failed to write test file: " + err.Error()
			result.Status = 500
			return result
		}
		filePath = tmpPath
	} else {
		filePath = FunctionFilePath(fn)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			result.Error = "function file not found — redeploy required"
			result.Status = 500
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result
		}
	}

	ctxJSON, err := json.Marshal(input)
	if err != nil {
		result.Error = "failed to build execution context"
		result.Status = 500
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result
	}

	timeout := time.Duration(fn.TimeoutMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	appURL := os.Getenv("APP_URL")
	allowNet := "--allow-net"
	if appURL != "" {
		// Allow Koolbase API + any external (developer flexibility in Phase 1)
		allowNet = "--allow-net"
	}

	cmd := exec.CommandContext(ctx, denoPath(),
		"run",
		"--quiet",
		"--no-prompt",
		allowNet,
		"--allow-env=__KOOLBASE_CTX",
		"--deny-read",
		"--deny-write",
		filePath,
	)
	cmd.Env = append([]string{fmt.Sprintf("__KOOLBASE_CTX=%s", ctxJSON)}, os.Environ()...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	result.DurationMs = int(time.Since(start).Milliseconds())

	if ctx.Err() == context.DeadlineExceeded {
		result.Status = 504
		result.Error = "function execution timed out"
		return result
	}

	if runErr != nil {
		result.Status = 500
		result.Error = stderr.String()
		if result.Error == "" {
			result.Error = runErr.Error()
		}
		return result
	}

	output := stdout.String()
	marker := "__KOOLBASE_RESULT__"
	if idx := findMarker(output, marker); idx >= 0 {
		jsonStr := output[idx+len(marker):]
		var body map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &body); err == nil {
			result.Body = body
		}
	}

	if result.Body == nil {
		result.Body = map[string]interface{}{"ok": true}
	}

	result.Status = 200
	result.Output = output
	return result
}

func findMarker(s, marker string) int {
	for i := 0; i <= len(s)-len(marker); i++ {
		if s[i:i+len(marker)] == marker {
			return i
		}
	}
	return -1
}

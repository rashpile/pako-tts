package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProvidersConfig_ReadsModelID(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
server:
  port: 8080

providers:
  default: "elevenlabs"
  list:
    - name: "elevenlabs"
      type: "elevenlabs"
      api_key: "test-key"
      model_id: "eleven_flash_v2_5"
      max_concurrent: 2
      timeout: 15s
    - name: "elevenlabs-no-model"
      type: "elevenlabs"
      api_key: "test-key-2"
      max_concurrent: 1
      timeout: 10s
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Providers.List) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(cfg.Providers.List))
	}

	if got := cfg.Providers.List[0].ModelID; got != "eleven_flash_v2_5" {
		t.Errorf("expected first provider ModelID 'eleven_flash_v2_5', got %q", got)
	}
	if got := cfg.Providers.List[1].ModelID; got != "" {
		t.Errorf("expected second provider ModelID '' (omitted in yaml), got %q", got)
	}
}

func TestLoadProvidersConfig_ReadsDefaultStyle(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
providers:
  default: "gemini"
  list:
    - name: "gemini"
      type: "gemini"
      api_key: "test-key"
      default_style: "warm, slightly slow"
    - name: "gemini-no-style"
      type: "gemini"
      api_key: "test-key-2"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Providers.List) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(cfg.Providers.List))
	}

	if got := cfg.Providers.List[0].DefaultStyle; got != "warm, slightly slow" {
		t.Errorf("expected DefaultStyle 'warm, slightly slow', got %q", got)
	}
	if got := cfg.Providers.List[1].DefaultStyle; got != "" {
		t.Errorf("expected DefaultStyle '' (omitted in yaml), got %q", got)
	}
}

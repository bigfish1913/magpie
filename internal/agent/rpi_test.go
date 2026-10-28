package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/edit"
)

func TestRpiAgentDir(t *testing.T) {
	home := syncHome(t)
	custom := filepath.Join(home, "dotfiles", "rpi")
	for _, c := range []struct {
		env, want string
	}{
		{"", filepath.Join(home, ".rpi", "agent")},
		{custom, custom},
		{"~", home},
		{"~/dotfiles/rpi", custom},
		{"rel/rpi", filepath.Join(home, ".rpi", "agent")},
	} {
		t.Setenv("RPI_CODING_AGENT_DIR", c.env)
		if a := rpi(home); a.Dir != c.want || a.Path != filepath.Join(c.want, "settings.json") {
			t.Errorf("%q: %+v", c.env, a)
		}
	}

	t.Setenv("RPI_CODING_AGENT_DIR", custom)
	if a := rpi(home); a.Icon != "rpi" || a.Bin != "rpi" || a.UA[0] != "rpi" {
		t.Fatalf("rpi metadata: %+v", a)
	}
	root := t.TempDir()
	d := distro{Name: "Ubuntu", Root: root, Home: "/home/me", Has: map[string]bool{"dir:.rpi": true}, Mirrored: true, Running: true}
	if a := wslAgent(wslKindOf("rpi"), d); a.Dir != filepath.Join(root, "home", "me", ".rpi", "agent") {
		t.Fatalf("wsl rpi: %+v", a)
	}
}

func TestRpiAgentRoundTrip(t *testing.T) {
	home := syncHome(t)
	dir := filepath.Join(home, ".rpi", "agent")
	settings, models := filepath.Join(dir, "settings.json"), filepath.Join(dir, "models.json")
	userSettings := []byte(`{
  "theme": "dark",
  "defaultProvider": "xai",
  "defaultModel": "grok-4.6",
  "defaultThinkingLevel": "medium",
  "enabledModels": ["xai/*"]
}
`)
	userModels := `{
  "providers": {
    "mine": {
      "baseUrl": "http://127.0.0.1:9/v1",
      "api": "openai-completions",
      "apiKey": "MINE_KEY",
      "models": [{"id": "m1", "contextWindow": 64000}]
    }
  }
}
`
	writeFile(t, settings, string(userSettings))
	writeFile(t, models, userModels)
	a := rpi(home)
	if a.Name != "rpi" || a.Field("model").Get() != "xai/grok-4.6" || a.Field("effort").Get() != "medium" {
		t.Fatalf("before connect: model=%q effort=%q agent=%+v", a.Field("model").Get(), a.Field("effort").Get(), a)
	}
	if err := a.Connect(); err != nil {
		t.Fatal(err)
	}
	if !a.Wired() || a.Check() != "" {
		t.Fatalf("not wired: values=%v check=%q", a.Values(), a.Check())
	}
	for k, want := range map[string]string{
		"defaultProvider":      "magpie",
		"defaultModel":         "relay/glm-4.6",
		"theme":                "dark",
		"defaultThinkingLevel": "medium",
	} {
		if v, _ := edit.GetJSON(settings, k); v != want {
			t.Fatalf("%s = %q:\n%s", k, v, readFile(settings))
		}
	}
	if v, _ := edit.GetJSON(models, "providers.mine.apiKey"); v != "MINE_KEY" {
		t.Fatalf("user provider changed: %q", v)
	}
	if v, _ := edit.GetJSON(models, "providers.magpie.baseUrl"); v == "" {
		t.Fatalf("rpi magpie provider missing:\n%s", readFile(models))
	}
	if err := a.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if a.Wired() {
		t.Fatal("still wired")
	}
	if got := []byte(readFile(settings)); !bytes.Equal(got, userSettings) {
		t.Fatalf("settings not restored:\n%s", got)
	}
	if got := readFile(models); got != userModels {
		t.Fatalf("models not restored:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".pi")); !os.IsNotExist(err) {
		t.Fatalf("~/.pi was touched: %v", err)
	}
}

func TestRpiRegistered(t *testing.T) {
	home := syncHome(t)
	found := false
	for _, a := range All() {
		if a.ID == "rpi" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("rpi is not registered in All()")
	}
	if got := rpi(home).Field("effort").Options(nil); len(got) == 0 {
		t.Fatal("rpi has no thinking options")
	}
}

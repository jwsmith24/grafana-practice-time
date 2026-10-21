// Package static checks repository config without running the stack.
package static

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

type composeConfig struct {
	Services map[string]struct {
		Volumes []struct {
			Type     string `json:"type"`
			Source   string `json:"source"`
			Target   string `json:"target"`
			ReadOnly bool   `json:"read_only"`
		} `json:"volumes"`
	} `json:"services"`
}

func loadCompose(t *testing.T) composeConfig {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}
	out, err := exec.Command("docker", "compose", "-f", "../../docker-compose.yml", "config", "--format", "json").Output()
	if err != nil {
		t.Fatalf("docker compose config: %v", err)
	}
	var c composeConfig
	if err := json.Unmarshal(out, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

// T1.2-5
func TestAlloyMounts(t *testing.T) {
	alloy, ok := loadCompose(t).Services["alloy"]
	if !ok {
		t.Fatal("no alloy service")
	}
	found := false
	for _, v := range alloy.Volumes {
		if strings.Contains(v.Source, "docker.sock") || strings.Contains(v.Target, "docker.sock") {
			t.Errorf("alloy mounts the Docker socket: %+v", v)
		}
		if v.Type == "volume" && v.Source == "lab-logs" {
			found = true
			if !v.ReadOnly {
				t.Error("alloy mounts lab-logs read-write")
			}
		}
	}
	if !found {
		t.Error("alloy does not mount lab-logs")
	}
}

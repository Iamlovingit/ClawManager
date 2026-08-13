package db

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRuntimeManifestsAreValidYAML(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, manifest := range deploymentRuntimeManifests(repoRoot) {
		t.Run(manifest, func(t *testing.T) {
			file, err := os.Open(manifest)
			if err != nil {
				t.Fatalf("open manifest: %v", err)
			}
			defer file.Close()

			decoder := yaml.NewDecoder(file)
			documents := 0
			for {
				var document any
				err := decoder.Decode(&document)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("parse manifest document %d: %v", documents+1, err)
				}
				// A trailing document separator is accepted by kubectl and decodes
				// as an empty document. Ignore it while still validating syntax.
				if document == nil {
					continue
				}
				documents++
			}
			if documents == 0 {
				t.Fatal("manifest contains no YAML documents")
			}
		})
	}
}

func TestRuntimeManifestsStartHermesRuntime(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, manifest := range deploymentRuntimeManifests(repoRoot) {
		t.Run(manifest, func(t *testing.T) {
			raw, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatalf("read manifest: %v", err)
			}
			pattern := regexp.MustCompile(`(?s)name:\s+hermes-runtime.*?spec:\s+replicas:\s+([0-9]+)`)
			matches := pattern.FindSubmatch(raw)
			if len(matches) != 2 {
				t.Fatalf("could not find hermes-runtime replicas in %s", manifest)
			}
			if string(matches[1]) != "1" {
				t.Fatalf("expected hermes-runtime replicas 1 in %s, got %s", manifest, matches[1])
			}
		})
	}
}

func TestRuntimeManifestsStartOpenCodeRuntime(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, manifest := range deploymentRuntimeManifests(repoRoot) {
		t.Run(manifest, func(t *testing.T) {
			raw, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatalf("read manifest: %v", err)
			}
			pattern := regexp.MustCompile(`(?s)name:\s+opencode-runtime.*?spec:\s+replicas:\s+([0-9]+)`)
			matches := pattern.FindSubmatch(raw)
			if len(matches) != 2 {
				t.Fatalf("could not find opencode-runtime replicas in %s", manifest)
			}
			if string(matches[1]) != "1" {
				t.Fatalf("expected opencode-runtime replicas 1 in %s, got %s", manifest, matches[1])
			}
		})
	}
}

func TestRuntimeManifestsExposeOpenCodePublicURLTemplate(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, manifest := range deploymentRuntimeManifests(repoRoot) {
		t.Run(manifest, func(t *testing.T) {
			raw, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatalf("read manifest: %v", err)
			}
			if !strings.Contains(string(raw), "name: CLAWMANAGER_OPENCODE_PUBLIC_URL_TEMPLATE") {
				t.Fatalf("manifest %s must expose the OpenCode public URL template", manifest)
			}
		})
	}
}

func TestRuntimeManifestsSeedLiteDefaultImages(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, manifest := range append(deploymentRuntimeManifests(repoRoot), filepath.Join(repoRoot, "backend", "deployments", "k8s", "clawreef-incluster.yaml")) {
		t.Run(manifest, func(t *testing.T) {
			raw, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatalf("read manifest: %v", err)
			}
			for _, image := range []string{
				"ghcr.io/yuan-lab-llm/agentsruntime/openclaw-lite:latest",
				"ghcr.io/yuan-lab-llm/agentsruntime/hermes-lite:latest",
				"ghcr.io/yuan-lab-llm/agentsruntime/opencode-lite:latest",
			} {
				if !strings.Contains(string(raw), image) {
					t.Fatalf("manifest %s must seed lite image %s", manifest, image)
				}
			}
		})
	}
}

func TestRuntimeManifestsExposeOpenClawGatewayOnPodIP(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, manifest := range deploymentRuntimeManifests(repoRoot) {
		t.Run(manifest, func(t *testing.T) {
			raw, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatalf("read manifest: %v", err)
			}
			text := string(raw)
			want := "/usr/local/bin/openclaw gateway run --allow-unconfigured --auth token --bind lan --force"
			if !strings.Contains(text, want) {
				t.Fatalf("manifest %s must expose OpenClaw gateway on the pod network with %q", manifest, want)
			}
			if strings.Contains(text, "--auth token --bind auto --force") {
				t.Fatalf("manifest %s must not use OpenClaw --bind auto because it can bind to loopback inside runtime pods", manifest)
			}
		})
	}
}

func deploymentRuntimeManifests(repoRoot string) []string {
	return []string{
		filepath.Join(repoRoot, "deployments", "k8s", "cluster", "clawmanager.yaml"),
		filepath.Join(repoRoot, "deployments", "k8s", "single-node", "clawmanager.yaml"),
		filepath.Join(repoRoot, "deployments", "k3s", "cluster", "clawmanager.yaml"),
		filepath.Join(repoRoot, "deployments", "k3s", "single-node", "clawmanager.yaml"),
	}
}

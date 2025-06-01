package gen

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorWithQueryInterface(t *testing.T) {
	inputPath, err := filepath.Abs("../../examples/query.go")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	goldenPath, err := filepath.Abs("../../examples/output/query.go")
	if err != nil {
		t.Fatalf("failed to get absolute output path: %v", err)
	}

	outputDir := t.TempDir()

	g := &Generator{Name: "examples"}

	if err := g.Process(inputPath, outputDir); err != nil {
		t.Fatalf("Process error: %v", err)
	}
	if err := g.Gen(); err != nil {
		t.Fatalf("Gen error: %v", err)
	}

	files, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read output dir: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no files were generated in %s", outputDir)
	}

	goldenBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
	}
	goldenStr := string(goldenBytes)

	generatedFile := filepath.Join(outputDir, files[0].Name())
	genBytes, err := os.ReadFile(generatedFile)
	if err != nil {
		t.Fatalf("failed to read generated file %s: %v", generatedFile, err)
	}
	generatedStr := string(genBytes)

	if _, err := parser.ParseFile(token.NewFileSet(), generatedFile, genBytes, parser.AllErrors); err != nil {
		t.Errorf("generated code %s has invalid Go syntax: %v", generatedFile, err)
	}

	if strings.Replace(goldenStr, "package output", "", 1) != strings.Replace(generatedStr, "package examples", "", 1) {
		t.Errorf("generated file differs from golden file\nGOLDEN: %s\nGENERATED: %s\n %s",
			goldenPath, generatedFile, generatedStr)
	}
}

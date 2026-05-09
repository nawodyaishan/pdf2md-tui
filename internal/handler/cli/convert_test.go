package cli

import (
	"errors"
	"testing"
	"time"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestMergeResultsPreservesPreflightResults(t *testing.T) {
	preflight := []domain.Result{{InputPath: "ignored.pdf", Status: domain.StatusIgnored}}
	live := []domain.Result{{InputPath: "done.pdf", Status: domain.StatusOK}}

	results := mergeResults(preflight, live)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].InputPath != "ignored.pdf" || results[0].Status != domain.StatusIgnored {
		t.Fatalf("expected preflight ignored result to be preserved first, got %#v", results[0])
	}
}

func TestSummarizeResultsCountsStatuses(t *testing.T) {
	boom := errors.New("boom")
	results := []domain.Result{
		{InputPath: "ignored.pdf", Status: domain.StatusIgnored},
		{InputPath: "ok.pdf", Status: domain.StatusOK, InputBytes: 100, OutputBytes: 40, Duration: 2 * time.Second},
		{InputPath: "failed.pdf", Status: domain.StatusError, Err: boom},
	}

	totals := summarizeResults(results)

	if totals.converted != 1 {
		t.Fatalf("expected 1 converted file, got %d", totals.converted)
	}
	if totals.ignored != 1 {
		t.Fatalf("expected 1 ignored file, got %d", totals.ignored)
	}
	if totals.errCount != 1 {
		t.Fatalf("expected 1 error, got %d", totals.errCount)
	}
	if totals.inputBytes != 100 || totals.outputBytes != 40 {
		t.Fatalf("unexpected byte totals: %+v", totals)
	}
	if totals.duration != 2*time.Second {
		t.Fatalf("expected 2s duration, got %s", totals.duration)
	}
}

func TestResolveOverwritePolicyRequiresForceInNonInteractiveMode(t *testing.T) {
	needsPrompt, err := resolveOverwritePolicy([]string{"existing.md"}, false, false)

	if err == nil {
		t.Fatal("expected overwrite error without --force")
	}
	if needsPrompt {
		t.Fatal("non-interactive mode should not request a prompt")
	}
}

func TestResolveOverwritePolicyAllowsWithForceFlag(t *testing.T) {
	needsPrompt, err := resolveOverwritePolicy([]string{"existing.md"}, true, false)

	if err != nil {
		t.Fatalf("expected no error with --force flag, got %v", err)
	}
	if needsPrompt {
		t.Fatal("force flag should not require a prompt")
	}
}

func TestResolveOverwritePolicyAllowsEmptyExistingPaths(t *testing.T) {
	needsPrompt, err := resolveOverwritePolicy([]string{}, false, false)

	if err != nil {
		t.Fatalf("expected no error with empty existing paths, got %v", err)
	}
	if needsPrompt {
		t.Fatal("no existing files should not require a prompt")
	}
}

func TestMergeResultsHandlesEmptyLists(t *testing.T) {
	results := mergeResults([]domain.Result{}, []domain.Result{})

	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d items", len(results))
	}
}

func TestMergeResultsCombinesPreflightAndLive(t *testing.T) {
	preflight := []domain.Result{
		{InputPath: "a.pdf", Status: domain.StatusIgnored},
		{InputPath: "b.pdf", Status: domain.StatusIgnored},
	}
	live := []domain.Result{
		{InputPath: "c.pdf", Status: domain.StatusOK},
	}

	results := mergeResults(preflight, live)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestSummarizeResultsEmptyInput(t *testing.T) {
	totals := summarizeResults([]domain.Result{})

	if totals.converted != 0 || totals.ignored != 0 || totals.errCount != 0 {
		t.Fatalf("expected zero counts for empty input, got %+v", totals)
	}
}

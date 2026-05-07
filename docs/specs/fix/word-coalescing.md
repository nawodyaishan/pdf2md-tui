# Fix: Adaptive Word Coalescing — Correct Character-to-Word Detection

> **Fix #1 in the "Text Extraction Fidelity" series**
>
> **Status:** Draft | **Priority:** Critical | **Label:** `area/extraction`, `type/bug`

---

## Problem Statement

The PDF-to-Markdown converter produces output where separate words are erroneously concatenated together. This makes the output unreliable for any downstream use case — reading, search indexing, section parsing, or LLM consumption.

**Observed artifacts** (from `Strategic Career Plan Review v1.pdf` → markdown):

| Rendered Output | Expected Output | Category |
|---|---|---|
| `Thecertification-to-hire` | `The certification-to-hire` | Missing space |
| `Asia-Pacificspecifically` | `Asia-Pacific specifically` | Missing space |
| `communityengagement` | `community engagement` | Missing space |
| `portfolio code.3.7` | `portfolio code.` newline `### 3.7` | Missing paragraph break |
| `offers.10. Consider` | `offers.` newline `10. Consider` | Missing paragraph break |
| `Month 10–12.Flaw 4` | `Month 10–12.` newline `**Flaw 4**` | Missing paragraph break |

These are not edge cases — they affect the majority of extracted paragraphs, rendering structured output unreliable.

---

## Root Cause

The single function responsible is `coalesceChars()` in [`internal/repository/pdf/tables.go`](../../../internal/repository/pdf/tables.go):

```go
// Current (buggy) approach:
closeEnough := (t.X - lastX) < charWidth*1.8
```

Where `charWidth = t.FontSize * 0.6` (a rough per-glyph width estimate).

For **12pt body text**: `charWidth = 7.2pt`, `threshold = 7.2 × 1.8 = 12.96pt`.

The gap between consecutive characters on the same line expressed as `t.X - lastX` **includes the width of the previous character** since `lastX` is the left edge. For 12pt text:
- Within a word: `t.X - lastX` ≈ 5–10pt (character width + small kerning)
- Between words: `t.X - lastX` ≈ 8–14pt (character width + inter-word space)

**The 12.96pt threshold is 2–4× too large**, easily swallowing word boundaries. The root flaw is using a **single threshold** where a **dual-threshold system** (intra-word vs. inter-word) is required — a pattern every mature PDF text extractor uses.

---

## Research Backing (via Exa MCP)

Three independent reference implementations converge on the same design:

### 1. rsc/pdf (`findWords`) — the canonical Go PDF library

The upstream of `ledongthuc/pdf` uses a **two-threshold system**:

```go
charSpace := ck.FontSize / 6     // ~2pt at 12pt — intra-word gap
wordSpace := ck.FontSize * 2 / 3 // ~8pt at 12pt — inter-word gap (insert space)
```

Characters whose left edge falls within `end + charSpace` (right edge + ~2pt) are same-word concatenations. Characters within `end + wordSpace` get a space inserted. Characters beyond `wordSpace` are separate words or line breaks.

### 2. Apache PDFBox (`PDFTextStripper`)

Uses a **dual-estimate approach**:
- Space character width via `getWidthOfSpace()` × tolerance factor
- Average character width × `getAverageCharTolerance()` (0.3)
- Takes the **minimum of the two deltas** for conservative boundary detection

### 3. pdfplumber

Exposes `x_tolerance` as the configurable word-break distance. Community issue [#987](https://github.com/jsvine/pdfplumber/issues/987) proposes making it proportional to font size — confirming the same requirement.

---

## Solution Design

### 3.1 Replace Single Threshold with Dual Threshold

Modify `coalesceChars()` to use the library's canonically available `W` field (character width in points) and a font-size-relative two-threshold system:

| Threshold | Formula (12pt example) | Semantic |
|---|---|---|
| `sameWordGap` | `fontSize / 6` (~2pt) | Characters closer than this belong to the same word |
| `spaceGap` | `fontSize * 2 / 3` (~8pt) | Characters within this gap get a space inserted |

**Algorithm**:

```
For each character on the same line:
  gap = currentChar.X - (previousChar.X + previousChar.W)  // actual inter-character gap
  
  if gap < sameWordGap:
    append character to current word (no space)
  else if gap < spaceGap:
    append " " + character to current word (word break with space)
  else:
    flush current word, start new word
```

This is a direct port of rsc/pdf's `findWords()` with one improvement: using `W` from `ledongthuc/pdf.Text` instead of estimating glyph width.

### 3.2 Why This Works

The two-threshold approach exploits a fundamental property of type-set text: **intra-character kerning gaps are always smaller than inter-word spacing gaps**. No PDF engine produces text where characters within a word are spaced wider than the space between words. The ratio between the two is consistently 3–5×, making the threshold robust across font sizes, fonts, and PDF producers.

### 3.3 Guard: Font Size Validation

PDF corruption or mixed-font scenarios can produce `FontSize = 0`. Add a guard:

```go
if t.FontSize <= 0 {
    continue  // skip malformed characters
}
```

---

## Implementation

### File to Modify

**[`internal/repository/pdf/tables.go`](../../../internal/repository/pdf/tables.go)** — `coalesceChars()` function, lines 54–116.

### Current Code (lines 89–108)

```go
sameLine := math.Abs(t.Y-lastY) < 2.0
charWidth := t.FontSize * 0.6
if charWidth < 3.0 {
    charWidth = 5.0
}
closeEnough := (t.X - lastX) < charWidth*1.8

if sameLine && closeEnough {
    current.text += s
} else {
    if strings.TrimSpace(current.text) != "" {
        words = append(words, current)
    }
    current = word{x: t.X, y: t.Y, text: s}
}
```

### Replacement Code

```go
sameLine := math.Abs(t.Y-lastY) < 2.0
if !sameLine {
    // Different line — flush current word
    if strings.TrimSpace(current.text) != "" {
        words = append(words, current)
    }
    current = word{x: t.X, y: t.Y, text: s}
    lastX, lastY, lastW = t.X, t.Y, t.W
    continue
}

// Same line — determine word boundary using gap analysis
// gap = distance from right edge of previous char to left edge of current char
gap := t.X - (lastX + lastW)

// Thresholds adapted from rsc/pdf findWords()
charSpace := t.FontSize / 6   // intra-word gap: characters closer than this are same word
wordSpace := t.FontSize * 2 / 3 // inter-word gap: characters within this get a space

if gap < charSpace {
    // Same word — concatenate without space
    current.text += s
} else if gap < wordSpace {
    // Same line, different word — insert space
    current.text += " " + s
} else {
    // Far enough apart to be separate words on the same line
    if strings.TrimSpace(current.text) != "" {
        words = append(words, current)
    }
    current = word{x: t.X, y: t.Y, text: s}
}
```

### Key Differences From Current Code

| Aspect | Current | Fixed |
|---|---|---|
| Threshold formula | `fontSize × 0.6 × 1.8 = fontSize × 1.08` | `fontSize / 6` (intra), `fontSize × 2/3` (inter) |
| Number of thresholds | 1 (binary: same word or new word) | 2 (same word, space, or new word) |
| Uses `W` field | No (estimates via `fontSize × 0.6`) | Yes (actual `Text.W` field from library) |
| Gap calculation | `t.X - lastX` (includes char width) | `t.X - (lastX + lastW)` (actual inter-char gap) |
| Font size guard | No | Yes (`if fontSize <= 0, skip`) |

---

## Test Cases

**[`internal/repository/pdf/tables_test.go`](../../../internal/repository/pdf/tables_test.go)**

### New Tests to Add

| Test Name | Input | Expected | What It Covers |
|---|---|---|---|
| `TestCoalesceChars_DetectsWordBoundaries` | `"H"(X=10) "i"(X=17) " "(gap) "G"(X=35) "o"(X=42)` | `["Hi", "Go"]` | Two separated words on same line |
| `TestCoalesceChars_InsertsSpaceBetweenWords` | `"The"(X=10..22) "cert"(X=32..42)` (gap > charSpace, gap < wordSpace) | `["The cert"]` | Space insertion between words (not concatenation) |
| `TestCoalesceChars_PreservesWordOrder` | Three words on same line | `["First", "Second", "Third"]` | Multi-word line ordering preserved |
| `TestCoalesceChars_HandlesDifferentFontSizes` | Char at fontSize=24 with space then char at fontSize=12 | `["Big", "small"]` | Mixed font size on same line |
| `TestCoalesceChars_SkipsZeroFontSize` | Text with FontSize=0 | `["Valid"]` (skip malformed) | Corrupted PDF guard |
| `TestCoalesceChars_LargeGapSeparatesWords` | Words far apart (> wordSpace) on same line | `["Far", "Apart"]` | Separate word creation |
| `TestCoalesceChars_WideWordBoundaryWithActualCharWidths` | Uses `W` field for gap calculation | Correct boundary | Actual `W` usage in gap calc |
| `TestCoalesceChars_PunctuationNearWords` | `"word"(X=10..36) "."(X=40)` then `"Next"(X=200)` | `["word.", "Next"]` | Punctuation attached to preceding word |
| `TestCoalesceChars_HyphenatedTerms` | `"Asia-Pacific"(closely-spaced)` then `"specifically"(with larger gap)` | `["Asia-Pacific", "specifically"]` | Hyphenated word boundaries |

### Modified Existing Tests

| Test Name | Change |
|---|---|
| `TestCoalesceChars_JoinsAdjacentChars` | No change needed (narrow gaps stay same-word) |
| `TestCoalesceChars_SplitsDistantChars` | No change needed (200pt gap is well beyond wordSpace) |

### Regression Test

Add a helper function to run the converter against `testdata/` PDFs and verify the output has no concatenated-word artifacts:

```go
func TestRegression_NoWordMerging(t *testing.T) {
    // Convert a known PDF and check output for common merged-word patterns
    // e.g., "Thecert", "communityengagement", "Asia-Pacificspecifically"
    // Run against test PDFs in the testdata directory
}
```

---

## Integration Impact

### No Changes Required To

| Component | Reason |
|---|---|
| `internal/service/converter.go` | Only calls `ExtractPageText()`; output format unchanged |
| `internal/repository/pdf/pdf.go` | Only delegates to `extractWithTables()`; interface unchanged |
| `internal/repository/pdf/analyze.go` | Unrelated pre-flight analysis |
| `internal/repository/pdf/images.go` | Unrelated image extraction |
| `internal/handler/cli/` | Unrelated CLI layer |
| `internal/domain/` | Data types unchanged |

### Minimal Impact To

| Component | Impact |
|---|---|
| Table detection | Slightly more accurate word positions → identical or better table boundaries |
| `applyLLMOptimizations` | Unrelated (post-processing regex pass) |
| Tests specifically checking word counts in `renderRowsAsMarkdown` output | Verify test expectations still match (unlikely to change for single-character-per-cell test data) |

---

## Acceptance Criteria

- [ ] `Thecertification-to-hire` → `The certification-to-hire` in converted output
- [ ] `Asia-Pacificspecifically` → `Asia-Pacific specifically`
- [ ] `communityengagement` → `community engagement`
- [ ] All existing `tables_test.go` tests pass without modification
- [ ] 100% of new test cases pass
- [ ] Running `make test` (full suite with race detector) passes
- [ ] No regression on table detection: pipe tables still render correctly
- [ ] Running `make run` against testdata produces clean output
- [ ] Manual verification on at least 3 real-world PDFs with mixed layouts
- [ ] Benchmarks: no measurable performance regression from the additional `W`-field usage (negligible: one extra float read per character)

---

## Follow-up Issue: Plain-Text Fallback

After this fix is validated, a secondary spec should address the missing plain-text fallback described in the architecture docs. Currently `ExtractPageText()` always calls `extractWithTables()` with no fallback to `page.GetPlainText()` — a divergence from the documented two-path design.

---

## References

- [`rsc/pdf` `findWords()` implementation](https://pkg.go.dev/rsc.io/pdf#example-Reader-TextWords) — canonical two-threshold word detection in Go
- [Apache PDFBox `PDFTextStripper`](https://github.com/BrentDouglas/pdfbox) — dual-estimate word boundary detection
- [pdfplumber issue #606](https://github.com/jsvine/pdfplumber/issues/606) — community discussion on word-space detection
- [pdfplumber issue #987](https://github.com/jsvine/pdfplumber/issues/987) — feature request for font-size-proportional tolerance
- [PDF Oxide extraction docs](https://pdf.oxide.fyi/docs/extraction/text) — industry best practices for PDF text layout extraction

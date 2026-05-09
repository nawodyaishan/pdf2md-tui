# Advanced PDF Extraction Fixes

This specification details the engineering solution for identified conversion quality issues: character collapse, spaced-out headings, and loss of structural integrity during noise stripping.

## 1. Geometric Character Deduplication
**Issue:** "Fake bold" detection (double-printing) collapses legitimate double letters (e.g., "across" -> "acros").
**Solution:** Move from a fixed-distance offset check to a Bounding Box Overlap check.
**Implementation:**
- Calculate the bounding box for each character.
- Only deduplicate if `S_A == S_B` and `Overlap(Rect_A, Rect_B) > 0.8`.
- Real adjacent characters (like 'ss') have ~0% overlap.

## 2. Adaptive Line-Level Spacing
**Issue:** Styled headings with intentional letter-spacing are broken into individual letters (e.g., "D E F I N I T I V E").
**Solution:** Use local-adaptive thresholds based on line statistics.
**Implementation:**
- For each line, calculate `MedianGap` and `StandardDeviation`.
- If `StdDev` is low and `MedianGap` is high (tracked text), increase the `charSpaceThreshold` for that line to `Median + 2σ`.

## 3. Block-Aware Noise Stripping
**Issue:** `--strip-noise` collapses paragraph breaks (`\n\n`), creating massive text blocks.
**Solution:** Change optimization from "Global Regex" to "Block-Level Rule".
**Implementation:**
- Apply horizontal whitespace collapsing *within* `BlockTypeText`.
- Explicitly preserve boundaries between `PageBlock` elements.

## 4. Unicode & Ligature Normalization
**Issue:** Ligatures (fi, fl) and PUA characters are not always correctly mapped.
**Solution:** Implement a comprehensive normalizer using NFKC and the Adobe Glyph List.
**Implementation:**
- Update `sanitizeText` to use `golang.org/x/text/unicode/norm`.
- Add explicit mappings for common PDF-encoded ligatures.

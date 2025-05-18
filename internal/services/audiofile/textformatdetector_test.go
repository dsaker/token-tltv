package audiofile

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"talkliketv.com/tltv/internal/util"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectTextFormat(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	t.Run("detect SRT format", func(t *testing.T) {
		content := `1
00:00:01,418 --> 00:00:04,170
This is the first subtitle line.

2
00:00:05,600 --> 00:00:08,900
This is the second subtitle line.

3
00:00:10,800 --> 00:00:14,000
This is the third subtitle line.`

		reader := strings.NewReader(content)
		format, err := DetectTextFormat(reader)

		assert.NoError(t, err)
		assert.Equal(t, Srt, format)
	})

	t.Run("detect OnePhrasePerLine format", func(t *testing.T) {
		content := `This is line one.
This is line two.
This is line three.
This is line four.
This is line five.`

		reader := strings.NewReader(content)
		format, err := DetectTextFormat(reader)

		assert.NoError(t, err)
		assert.Equal(t, OnePhrasePerLine, format)
	})

	t.Run("detect Paragraph format", func(t *testing.T) {
		content := `This is a very long paragraph that contains multiple sentences. It continues for quite some time so that the average line length will be greater than 80 characters. This should trigger the paragraph detection logic in our function.

This is another paragraph with multiple sentences. It's also quite long to ensure that the average line length will exceed our threshold. The detector should identify this as a paragraph format rather than one phrase per line.`

		reader := strings.NewReader(content)
		format, err := DetectTextFormat(reader)

		assert.NoError(t, err)
		assert.Equal(t, Paragraph, format)
	})

	t.Run("empty file", func(t *testing.T) {
		content := ``
		reader := strings.NewReader(content)
		_, err := DetectTextFormat(reader)

		assert.Error(t, err)
		assert.Equal(t, "file is empty", err.Error())
	})

	t.Run("nil reader", func(t *testing.T) {
		_, err := DetectTextFormat(nil)

		assert.Error(t, err)
		assert.Equal(t, "fileStream is nil", err.Error())
	})

	t.Run("read error during format detection", func(t *testing.T) {
		mockReader := &mockReadSeeker{
			readErr: errors.New("mock read error"),
		}

		_, err := DetectTextFormat(mockReader)

		assert.Error(t, err)
		assert.Equal(t, "mock read error", err.Error())
	})

	t.Run("seek error after SRT check", func(t *testing.T) {
		// Create a reader that will succeed for the SRT check but fail on the first seek
		mockReader := &mockReadSeeker{
			content:        []byte("00:00:01,418 --> 00:00:04,170\nSome text"),
			seekErrOnCount: 1,
			seekErr:        errors.New("mock seek error"),
		}

		_, err := DetectTextFormat(mockReader)

		assert.Error(t, err)
		assert.Equal(t, "mock seek error", err.Error())
	})

	t.Run("seek error after lineCount check", func(t *testing.T) {
		// Create content that will pass the SRT check
		content := "Line 1\nLine 2\nLine 3"
		reader := &mockReadSeeker{
			content:        []byte(content),
			seekErrOnCount: 2, // Fail on the second seek
			seekErr:        errors.New("mock seek error"),
		}

		_, err := DetectTextFormat(reader)

		assert.Error(t, err)
		assert.Equal(t, "mock seek error", err.Error())
	})

	t.Run("mixed format with some SRT timestamps", func(t *testing.T) {
		content := `This is a normal line.
00:01:23,456 --> 00:04:56,789
This is another normal line.
This is a third normal line.`

		reader := strings.NewReader(content)
		format, err := DetectTextFormat(reader)

		assert.NoError(t, err)
		assert.Equal(t, Srt, format, "Should detect as SRT when timestamp is found")
	})

	t.Run("very short lines but many of them", func(t *testing.T) {
		// Create 20 short lines
		lines := make([]string, 20)
		for i := 0; i < 20; i++ {
			lines[i] = "Short."
		}
		content := strings.Join(lines, "\n")

		reader := strings.NewReader(content)
		format, err := DetectTextFormat(reader)

		assert.NoError(t, err)
		assert.Equal(t, OnePhrasePerLine, format, "Should detect as OnePhrasePerLine when many short lines")
	})

	t.Run("few long lines", func(t *testing.T) {
		// Create 3 very long lines
		longLine := strings.Repeat("Very long sentence with lots of words. ", 10)
		lines := []string{longLine, longLine, longLine}
		content := strings.Join(lines, "\n")

		reader := strings.NewReader(content)
		format, err := DetectTextFormat(reader)

		assert.NoError(t, err)
		assert.Equal(t, Paragraph, format, "Should detect as Paragraph when few long lines")
	})
}

func TestSrtFormatCheck(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "valid SRT timestamp",
			line:     "00:01:23,456 --> 00:04:56,789",
			expected: true,
		},
		{
			name:     "invalid SRT timestamp - wrong separator",
			line:     "00:01:23,456 -> 00:04:56,789",
			expected: false,
		},
		{
			name:     "invalid SRT timestamp - wrong format",
			line:     "00:01:23.456 --> 00:04:56.789",
			expected: false,
		},
		{
			name:     "invalid SRT timestamp - incomplete",
			line:     "00:01:23,456 -->",
			expected: false,
		},
		{
			name:     "completely different text",
			line:     "This is just some random text",
			expected: false,
		},
		{
			name:     "empty line",
			line:     "",
			expected: false,
		},
		{
			name:     "numbers only",
			line:     "1234",
			expected: false,
		},
		{
			name:     "almost correct format",
			line:     "00:00:00,000-->00:00:00,000", // missing space
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := srtFormatCheck(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// A mock reader/seeker for testing error cases
type mockReadSeeker struct {
	content        []byte
	position       int64
	readErr        error
	seekErr        error
	seekCount      int
	seekErrOnCount int
}

func (m *mockReadSeeker) Read(p []byte) (int, error) {
	if m.readErr != nil {
		return 0, m.readErr
	}

	if m.position >= int64(len(m.content)) {
		return 0, io.EOF
	}

	n := copy(p, m.content[m.position:])
	m.position += int64(n)
	return n, nil
}

func (m *mockReadSeeker) Seek(offset int64, whence int) (int64, error) {
	m.seekCount++

	if m.seekCount == m.seekErrOnCount && m.seekErr != nil {
		return 0, m.seekErr
	}

	switch whence {
	case io.SeekStart:
		m.position = offset
	case io.SeekCurrent:
		m.position += offset
	case io.SeekEnd:
		m.position = int64(len(m.content)) + offset
	}

	return m.position, nil
}

// Helper function to test with specific file content
func TestWithSpecificContent(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	testCases := []struct {
		name     string
		content  string
		expected TextFormat
	}{
		{
			name: "typical SRT file",
			content: `1
00:00:01,000 --> 00:00:04,000
This is subtitle one.

2
00:00:05,000 --> 00:00:09,000
This is subtitle two.`,
			expected: Srt,
		},
		{
			name: "script with short lines",
			content: `ALICE: Hello Bob.
BOB: Hi Alice, how are you?
ALICE: I'm fine, thank you.
BOB: That's good to hear.
ALICE: What about you?`,
			expected: OnePhrasePerLine,
		},
		{
			name: "prose paragraphs",
			content: `It was the best of times, it was the worst of times, it was the age of wisdom, it was the age of foolishness, it was the epoch of belief, it was the epoch of incredulity, it was the season of Light, it was the season of Darkness, it was the spring of hope, it was the winter of despair.

We had everything before us, we had nothing before us, we were all going direct to Heaven, we were all going direct the other way – in short, the period was so far like the present period, that some of its noisiest authorities insisted on its being received, for good or for evil, in the superlative degree of comparison only.`,
			expected: Paragraph,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.content)
			format, err := DetectTextFormat(reader)

			require.NoError(t, err)
			assert.Equal(t, tc.expected, format)

			// Verify we can re-read the content after detection
			_, err = reader.Seek(0, io.SeekStart)
			require.NoError(t, err)

			content, err := io.ReadAll(reader)
			require.NoError(t, err)
			assert.Equal(t, tc.content, string(content), "Original content should be preserved")
		})
	}
}

// Additional test to check buffer handling
func TestBufferHandling(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	// Test with a buffer exactly matching a typical buffer size
	t.Run("large buffer", func(t *testing.T) {
		// Create a file that's exactly 4096 bytes (common buffer size)
		content := bytes.Repeat([]byte("Line of text.\n"), 341)

		reader := bytes.NewReader(content)
		format, err := DetectTextFormat(reader)

		require.NoError(t, err)
		assert.Equal(t, OnePhrasePerLine, format)
	})
}

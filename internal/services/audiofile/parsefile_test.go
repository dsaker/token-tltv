package audiofile

import (
	"os"
	"path/filepath"
	"strings"
	"talkliketv.com/tltv/internal/util"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mime/multipart"
)

// TestParseFileContent tests the parseFileContent function using real files
func TestParseFileContent(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	// Create a temporary directory for our test files
	tmpDir, err := os.MkdirTemp("/tmp", "parsefiletest")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		content     string
		fileType    TextFormat
		expected    []string
		expectError bool
	}{
		{
			name: "Parse SRT file",
			content: `1
00:00:01,000 --> 00:00:05,000
Hello world this is a test subtitle.

2
00:00:06,000 --> 00:00:10,000
Another subtitle line for testing.`,
			fileType:    Srt,
			expected:    []string{"Hello world this is a test subtitle.", "Another subtitle line for testing."},
			expectError: false,
		},
		{
			name:        "Parse paragraph file",
			content:     "This is the first paragraph. It has multiple sentences. And some bad punctuation!\n\nThis is the second paragraph. With even more text.",
			fileType:    Paragraph,
			expected:    []string{"This is the first paragraph", "It has multiple sentences", "And some bad punctuation", "This is the second paragraph", "With even more text"},
			expectError: false,
		},
		{
			name:        "Parse one phrase per line",
			content:     "Line one phrase one.\nLine two phrase two.\nLine three phrase three.",
			fileType:    OnePhrasePerLine,
			expected:    []string{"Line one phrase one.", "Line two phrase two.", "Line three phrase three."},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a file for this test
			filePath := filepath.Join(tmpDir, tt.name+".txt")
			err := os.WriteFile(filePath, []byte(tt.content), 0600)
			require.NoError(t, err)

			// Open the file - os.File implements multipart.File
			file, err := os.Open(filePath)
			require.NoError(t, err)
			defer file.Close()

			// Call the function being tested - os.File satisfies multipart.File
			result, err := parseFileContent(file, tt.fileType)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expected), len(result), "Result length should match expected")
				for i, phrase := range tt.expected {
					if i < len(result) {
						assert.Contains(t, result[i], phrase, "Result should contain expected phrase")
					}
				}
			}
		})
	}
}

// TestParseSrt tests the parseSrt function specifically
func TestParseSrt(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	// Create a temporary directory for our test files
	tmpDir, err := os.MkdirTemp("/tmp", "parsesrttest")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "Basic SRT file",
			content: `1
00:00:01,000 --> 00:00:05,000
Hello world this is a test subtitle.

2
00:00:06,000 --> 00:00:10,000
Another subtitle line for testing.`,
			expected: []string{"Hello world this is a test subtitle.", "Another subtitle line for testing."},
		},
		{
			name: "SRT with formatting tags",
			content: `1
00:00:01,000 --> 00:00:05,000
<font color="white">This has font tags</font>

2
00:00:06,000 --> 00:00:10,000
[Music playing] This has some brackets.

3
00:00:11,000 --> 00:00:15,000
This has <i>italic</i> formatting.`,
			expected: []string{"This has some brackets.", "This has italic formatting."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a file for this test
			filePath := filepath.Join(tmpDir, tt.name+".srt")
			err := os.WriteFile(filePath, []byte(tt.content), 0600)
			require.NoError(t, err)

			// Open the file - os.File implements multipart.File
			file, err := os.Open(filePath)
			require.NoError(t, err)
			defer file.Close()

			// Test the parseSrt function
			result := parseSrt(file)

			assert.Equal(t, len(tt.expected), len(result), "Result length should match expected")
			for i, phrase := range tt.expected {
				if i < len(result) {
					assert.Equal(t, strings.TrimSpace(phrase), strings.TrimSpace(result[i]), "Result should match expected phrase")
				}
			}
		})
	}
}

// TestParseParagraph tests the parseParagraph function specifically
func TestParseParagraph(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	// Create a temporary directory for our test files
	tmpDir, err := os.MkdirTemp("/tmp", "parseparagraphtest")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "Simple paragraphs",
			content:  "This is the first paragraph. It has multiple sentences.\n\nThis is the second paragraph.",
			expected: []string{"This is the first paragraph", "It has multiple sentences", "This is the second paragraph"},
		},
		{
			name:     "Paragraphs with different punctuation",
			content:  "First sentence with exclamation! Second sentence with question? Third sentence with period.",
			expected: []string{"First sentence with exclamation", "Second sentence with question", "Third sentence with period"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a file for this test
			filePath := filepath.Join(tmpDir, tt.name+".txt")
			err := os.WriteFile(filePath, []byte(tt.content), 0600)
			require.NoError(t, err)

			// Open the file - os.File implements multipart.File
			file, err := os.Open(filePath)
			require.NoError(t, err)
			defer file.Close()

			// Test the parseParagraph function
			result := parseParagraph(file)

			assert.Equal(t, len(tt.expected), len(result), "Result length should match expected")
			for i, phrase := range tt.expected {
				if i < len(result) {
					assert.Contains(t, result[i], phrase, "Result should contain expected phrase")
				}
			}
		})
	}
}

// TestParseSingle tests the parseSingle function specifically
func TestParseSingle(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	// Create a temporary directory for our test files
	tmpDir, err := os.MkdirTemp("/tmp", "parsesingletest")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "Simple one phrase per line",
			content:  "Line one phrase one.\nLine two phrase two.\nLine three phrase three.",
			expected: []string{"Line one phrase one.", "Line two phrase two.", "Line three phrase three."},
		},
		{
			name:     "Mix of short and long phrases",
			content:  "short short phrase.\nThis is a very long phrase that should be split because it has more than ten words and it helps with text-to-speech. This is the second part.\nThis is a medium length phrase.",
			expected: []string{"This is a very long phrase that should be split because it has more than ten words and it helps with text-to-speech.", "This is the second part.", "This is a medium length phrase."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a file for this test
			filePath := filepath.Join(tmpDir, tt.name+".txt")
			err := os.WriteFile(filePath, []byte(tt.content), 0600)
			require.NoError(t, err)

			// Open the file - os.File implements multipart.File
			file, err := os.Open(filePath)
			require.NoError(t, err)
			defer file.Close()

			// Test the parseSingle function
			result := parseSingle(file)

			assert.Equal(t, len(tt.expected), len(result), "Result length should match expected")
			for i, phrase := range tt.expected {
				if i < len(result) {
					assert.Contains(t, result[i], phrase, "Result should contain expected phrase")
				}
			}
		})
	}
}

// TestSplitLongPhrases tests the helper function splitLongPhrases directly
func TestSplitLongPhrases(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Short phrase (below minimum)",
			input:    "Hi there",
			expected: []string{},
		},
		{
			name:     "Medium phrase (between min and max)",
			input:    "This is a test phrase with good length.",
			expected: []string{"This is a test phrase with good length."},
		},
		{
			name:     "Long phrase with punctuation",
			input:    "This is a very long sentence that should be split, because it has more than ten words and it helps with text processing. This is the second sentence.",
			expected: []string{"This is a very long sentence that should be split, ", "because it has more than ten words and it helps with text processing. ", "This is the second sentence."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLongPhrases(tt.input)
			assert.Equal(t, len(tt.expected), len(result), "Result length should match expected")
			for i, phrase := range tt.expected {
				if i < len(result) {
					assert.Equal(t, strings.TrimSpace(phrase), strings.TrimSpace(result[i]), "Result should match expected phrase")
				}
			}
		})
	}
}

// TestReplaceFmt tests the replaceFmt helper function
func TestReplaceFmt(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "With square brackets",
			input:    "This text has [some notes] in it.",
			expected: "This text has  in it.",
		},
		{
			name:     "With curly braces",
			input:    "This text has {some notes} in it.",
			expected: "This text has  in it.",
		},
		{
			name:     "With HTML tags",
			input:    "This text has <b>bold</b> and <i>italic</i> formatting.",
			expected: "This text has bold and italic formatting.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceFmt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSplitOnEndingPunctuation tests the splitOnEndingPunctuation helper function
func TestSplitOnEndingPunctuation(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple sentences with periods",
			input:    "This is the first sentence. This is the second sentence.",
			expected: []string{"This is the first sentence", "This is the second sentence"},
		},
		{
			name:     "Mixed punctuation",
			input:    "First sentence! Second sentence? Third sentence.",
			expected: []string{"First sentence", "Second sentence", "Third sentence"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitOnEndingPunctuation(tt.input)
			assert.Equal(t, len(tt.expected), len(result), "Result length should match expected")
			for i, sentence := range tt.expected {
				assert.Equal(t, sentence, result[i])
			}
		})
	}
}

// TestParseParagraphNilFile tests the parseParagraph function with a nil file
func TestParseParagraphNilFile(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}
	var file multipart.File = nil
	result := parseParagraph(file)
	assert.Nil(t, result, "Result should be nil for nil input")
}

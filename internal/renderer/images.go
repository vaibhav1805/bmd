package renderer

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImageProtocol identifies the terminal image rendering protocol to use.
type ImageProtocol int

const (
	ProtocolNone ImageProtocol = iota // No image support; use alt text
	ProtocolITerm2                      // iTerm2 inline images
	ProtocolKitty                       // Kitty graphics protocol
	ProtocolSixel                       // Sixel image format
	ProtocolUnicode                     // Unicode block characters (emoji)
)

// DetectImageProtocol checks terminal capabilities and returns the best-supported protocol.
func DetectImageProtocol() ImageProtocol {
	// Check environment variables
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")
	iterm := os.Getenv("ITERM_PROGRAM")
	iterm2 := os.Getenv("ITERM2_SHOULDMANAGEPASTEBOARD")

	fmt.Fprintf(os.Stderr, "[DEBUG] DetectImageProtocol:\n")
	fmt.Fprintf(os.Stderr, "  TERM=%q\n", term)
	fmt.Fprintf(os.Stderr, "  TERM_PROGRAM=%q\n", termProgram)
	fmt.Fprintf(os.Stderr, "  ITERM_PROGRAM=%q\n", iterm)
	fmt.Fprintf(os.Stderr, "  ITERM2_SHOULDMANAGEPASTEBOARD=%q\n", iterm2)

	// Explicit Kitty detection
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] → Kitty (KITTY_WINDOW_ID set)\n")
		return ProtocolKitty
	}
	if strings.Contains(term, "kitty") {
		fmt.Fprintf(os.Stderr, "[DEBUG] → Kitty (TERM=kitty)\n")
		return ProtocolKitty
	}

	// macOS checks (Terminal.app AND iTerm2)
	if termProgram == "Apple_Terminal" {
		fmt.Fprintf(os.Stderr, "[DEBUG] → macOS Terminal.app (iTerm2 protocol)\n")
		return ProtocolITerm2
	}
	if termProgram == "iTerm.app" || iterm != "" || iterm2 != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] → iTerm2 (ITERM_PROGRAM or ITERM2_* set)\n")
		return ProtocolITerm2
	}
	if strings.Contains(termProgram, "Terminal") && strings.Contains(termProgram, "Mac") {
		fmt.Fprintf(os.Stderr, "[DEBUG] → macOS Terminal (iTerm2 fallback)\n")
		return ProtocolITerm2
	}

	// Alacritty detection (multiple checks)
	if strings.Contains(term, "alacritty") || strings.Contains(strings.ToLower(os.Getenv("COLORTERM")), "alacritty") {
		fmt.Fprintf(os.Stderr, "[DEBUG] → Alacritty (Kitty protocol)\n")
		return ProtocolKitty
	}

	// WezTerm
	if strings.Contains(term, "wezterm") {
		fmt.Fprintf(os.Stderr, "[DEBUG] → WezTerm (Kitty protocol)\n")
		return ProtocolKitty
	}

	// Sixel support
	if strings.Contains(term, "sixel") {
		fmt.Fprintf(os.Stderr, "[DEBUG] → Sixel\n")
		return ProtocolSixel
	}

	// xterm-256color: Try Kitty protocol (works on Alacritty, modern xterm, WezTerm)
	if strings.Contains(term, "xterm") || strings.Contains(term, "screen") || strings.Contains(term, "tmux") {
		fmt.Fprintf(os.Stderr, "[DEBUG] → Modern terminal (Kitty fallback for %s)\n", term)
		return ProtocolKitty
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] → No protocol detected, using Unicode\n")
	return ProtocolUnicode
}

// CanRenderImages returns true if the terminal supports some form of image rendering.
func CanRenderImages() bool {
	return DetectImageProtocol() != ProtocolNone
}

// ResolveImageURL converts a relative image URL to an absolute path.
// basePath is the directory of the document being rendered.
// Returns the resolved path and true if it's a local file, or the original URL and false if it's remote.
func ResolveImageURL(imageURL, basePath string) (string, bool) {
	// Remote URLs: return as-is
	if strings.HasPrefix(imageURL, "http://") ||
		strings.HasPrefix(imageURL, "https://") ||
		strings.HasPrefix(imageURL, "ftp://") {
		return imageURL, false
	}

	// Local file URLs
	if strings.HasPrefix(imageURL, "file://") {
		imageURL = imageURL[7:]
	}

	// Relative paths: resolve relative to basePath
	if !strings.HasPrefix(imageURL, "/") {
		imageURL = filepath.Join(basePath, imageURL)
	}

	return imageURL, true
}

// LoadImageData reads image file data from disk or cache.
// For local files, returns the file contents. For remote URLs, would cache locally.
// Returns nil if the file cannot be loaded.
func LoadImageData(imagePath string, useCache bool) []byte {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil
	}
	return data
}

// ImageToITerm2 encodes image data as an iTerm2 inline image sequence.
// Returns an ANSI sequence that iTerm2 will render as an inline image.
func ImageToITerm2(imageData []byte, width, height int) string {
	if len(imageData) == 0 {
		return ""
	}

	// iTerm2 inline image protocol (proper format):
	// OSC 1337 ; File = [params] : base64 BEL
	// Params: name, size (original bytes), width (chars), height (chars), inline=1
	encoded := base64.StdEncoding.EncodeToString(imageData)

	// Debug output
	fmt.Fprintf(os.Stderr, "[DEBUG] ImageToITerm2: %d bytes -> %d chars encoded, w=%d h=%d\n",
		len(imageData), len(encoded), width, height)

	// Format: name=image.png;size=<bytes>;width=<chars>;height=<chars>;inline=1;preserveAspectRatio=1
	// Use \x1b\\ (ST) as terminator - more reliable than \x07 (BEL) on some terminals
	return fmt.Sprintf("\x1b]1337;File=name=image.png;size=%d;width=%d;height=%d;inline=1;preserveAspectRatio=1:%s\x1b\\",
		len(imageData), width, height, encoded)
}

// ImageToKitty encodes image data as a Kitty graphics protocol sequence.
// This protocol works in Kitty, Alacritty (with Kitty support), and WezTerm.
// Returns an ANSI sequence that compatible terminals will render as an inline image.
func ImageToKitty(imageData []byte, width, height int) string {
	if len(imageData) == 0 {
		return ""
	}

	// Kitty graphics protocol with image ID:
	// \x1b_Ga=T,i=ID,s=WIDTH,v=HEIGHT,f=100,m=0:base64data\x1b\\
	// a=T (transmit), i=ID (unique image ID), s=width, v=height, f=100 (PNG), m=0 (final chunk)
	encoded := base64.StdEncoding.EncodeToString(imageData)

	fmt.Fprintf(os.Stderr, "[DEBUG] ImageToKitty: %d bytes -> %d chars, w=%d h=%d\n",
		len(imageData), len(encoded), width, height)

	// Generate unique ID based on data hash (simple approach)
	imageID := 1 // Use static ID - Alacritty may need this

	// m=0 means final chunk (no more data coming)
	payload := fmt.Sprintf("\x1b_Ga=T,i=%d,s=%d,v=%d,f=100,m=0:%s\x1b\\", imageID, width, height, encoded)
	return payload + "\n"
}

// ImageToSixel converts image data to Sixel format (placeholder).
// Sixel encoding is complex; for Phase 5, this returns a placeholder or uses an external tool.
func ImageToSixel(imageData []byte, width, height int) string {
	// For Phase 5, Sixel support is optional; can use an external tool like "convert" from ImageMagick
	// or return a placeholder: "[Sixel image would render here]"
	return "[Image (Sixel format) would render here]"
}

// ImageToUnicode generates a Unicode block character representation of the image.
// This is a fallback that works in any terminal; the result is ASCII/Unicode art.
func ImageToUnicode(imageData []byte, altText string, width int) string {
	if altText != "" {
		return "[Image: " + altText + "]"
	}
	return "[Image]"
}

// ImageToTerminal renders image data using the best-supported protocol.
// If no protocol is available, saves image to a temp file and shows the path.
func ImageToTerminal(imageData []byte, imagePath, altText string, width, height int) string {
	if len(imageData) == 0 {
		// Fallback to alt text if no data
		return altText
	}

	protocol := DetectImageProtocol()
	protocolNames := map[ImageProtocol]string{
		ProtocolNone:    "None",
		ProtocolITerm2:  "iTerm2",
		ProtocolKitty:   "Kitty",
		ProtocolSixel:   "Sixel",
		ProtocolUnicode: "Unicode",
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] ImageToTerminal: protocol=%s, size=%d bytes\n", protocolNames[protocol], len(imageData))

	switch protocol {
	case ProtocolKitty:
		result := ImageToKitty(imageData, width, height)
		fmt.Fprintf(os.Stderr, "[DEBUG] Kitty sequence generated, %d bytes\n", len(result))
		return result
	case ProtocolITerm2:
		result := ImageToITerm2(imageData, width, height)
		fmt.Fprintf(os.Stderr, "[DEBUG] iTerm2 sequence generated, %d bytes\n", len(result))
		return result
	case ProtocolSixel:
		return ImageToSixel(imageData, width, height)
	case ProtocolUnicode:
		// Try to save image to a temp file and show path
		tempPath := SaveImageTemp(imageData, altText)
		if tempPath != "" {
			return "[Image: " + altText + " - saved to " + tempPath + "]"
		}
		return ImageToUnicode(imageData, altText, width)
	default:
		return altText
	}
}

// SaveImageTemp writes image data to a temporary file and returns the path.
// Returns empty string on failure.
func SaveImageTemp(data []byte, hint string) string {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "bmd-image-"+hint+".png")
	err := os.WriteFile(tmpFile, data, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] Failed to save image temp file: %v\n", err)
		return ""
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] Saved image to: %s\n", tmpFile)
	return tmpFile
}

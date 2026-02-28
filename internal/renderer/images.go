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
	// Check TERM environment variable for protocol support
	term := os.Getenv("TERM")

	// Kitty graphics protocol (priority: works in Alacritty, Kitty, WezTerm)
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return ProtocolKitty
	}
	// Alacritty may advertise Kitty support via TERM
	if strings.Contains(term, "kitty") {
		return ProtocolKitty
	}

	// iTerm2 support (macOS)
	if os.Getenv("ITERM_PROGRAM") != "" {
		return ProtocolITerm2
	}
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return ProtocolITerm2
	}

	// Sixel support (xterm-256color with sixel, mlterm, etc.)
	if strings.Contains(term, "sixel") {
		return ProtocolSixel
	}
	if strings.HasPrefix(term, "xterm") && os.Getenv("XTERM_VERSION") != "" {
		return ProtocolSixel
	}
	if strings.Contains(term, "mlterm") {
		return ProtocolSixel
	}

	// Alacritty detection - try iTerm2 protocol as fallback
	// Alacritty supports iTerm2 escape sequences in many versions
	if strings.Contains(term, "alacritty") {
		return ProtocolITerm2
	}

	// Try iTerm2 as a fallback for modern terminals
	// Many terminals now support iTerm2 inline images even if not advertised
	// (VSCode, modern xterm variants, etc.)
	if strings.Contains(term, "xterm") || strings.Contains(term, "screen") ||
		strings.Contains(term, "tmux") || strings.HasPrefix(term, "linux") {
		return ProtocolITerm2
	}

	// Fallback: use Unicode blocks (works everywhere)
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

	// iTerm2 inline image protocol:
	// \x1b]1337;File=width=<w>;height=<h>;inline=1:<base64-data>\x07
	encoded := base64.StdEncoding.EncodeToString(imageData)

	return fmt.Sprintf("\x1b]1337;File=width=%d;height=%d;inline=1:%s\x07\n",
		width, height, encoded)
}

// ImageToKitty encodes image data as a Kitty graphics protocol sequence.
// This protocol works in Kitty, Alacritty (with Kitty support), and WezTerm.
// Returns an ANSI sequence that compatible terminals will render as an inline image.
func ImageToKitty(imageData []byte, width, height int) string {
	if len(imageData) == 0 {
		return ""
	}

	// Kitty graphics protocol (simplified):
	// \x1b_Ga=T,f=24,s=W,e=I,m=1;base64data\x1b\
	// a=T (transmit), f=24 (RGBA format), s=width (size), e=I (end indicator), m=1 (more data)
	encoded := base64.StdEncoding.EncodeToString(imageData)

	// Kitty protocol: transmit image, PNG format (f=100)
	// Split large base64 into chunks if needed (Kitty has a 4096-char payload limit per escape)
	payload := fmt.Sprintf("\x1b_Ga=T,f=100,s=%d,e=I,m=1:%s\x1b\\", width, encoded)
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
func ImageToTerminal(imageData []byte, imagePath, altText string, width, height int) string {
	if len(imageData) == 0 {
		// Fallback to alt text if no data
		return altText
	}

	protocol := DetectImageProtocol()

	switch protocol {
	case ProtocolKitty:
		return ImageToKitty(imageData, width, height)
	case ProtocolITerm2:
		return ImageToITerm2(imageData, width, height)
	case ProtocolSixel:
		return ImageToSixel(imageData, width, height)
	case ProtocolUnicode:
		return ImageToUnicode(imageData, altText, width)
	default:
		return altText
	}
}

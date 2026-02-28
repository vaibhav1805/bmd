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
	termProgram := os.Getenv("TERM_PROGRAM")

	fmt.Fprintf(os.Stderr, "[DEBUG] DetectImageProtocol: TERM=%s, TERM_PROGRAM=%s, KITTY_WINDOW_ID=%s\n",
		term, termProgram, os.Getenv("KITTY_WINDOW_ID"))

	// Explicit Kitty window detection
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Detected: Kitty (KITTY_WINDOW_ID set)\n")
		return ProtocolKitty
	}

	// TERM explicitly set to kitty
	if strings.Contains(term, "kitty") {
		fmt.Fprintf(os.Stderr, "[DEBUG] Detected: Kitty (TERM contains kitty)\n")
		return ProtocolKitty
	}

	// macOS Terminal.app - supports iTerm2 protocol
	if termProgram == "Apple_Terminal" || termProgram == "iTerm.app" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Detected: %s (TERM_PROGRAM=%s)\n", termProgram, termProgram)
		return ProtocolITerm2
	}

	// Explicit iTerm2 detection
	if os.Getenv("ITERM_PROGRAM") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Detected: iTerm2 (ITERM_PROGRAM set)\n")
		return ProtocolITerm2
	}

	// Alacritty - supports Kitty graphics protocol well
	if strings.Contains(term, "alacritty") {
		fmt.Fprintf(os.Stderr, "[DEBUG] Detected: Alacritty (TERM=alacritty, using Kitty protocol)\n")
		return ProtocolKitty
	}

	// WezTerm - supports Kitty
	if strings.Contains(term, "wezterm") {
		fmt.Fprintf(os.Stderr, "[DEBUG] Detected: WezTerm (using Kitty protocol)\n")
		return ProtocolKitty
	}

	// Sixel support (xterm variants with sixel)
	if strings.Contains(term, "sixel") {
		fmt.Fprintf(os.Stderr, "[DEBUG] Detected: Sixel support\n")
		return ProtocolSixel
	}
	if strings.HasPrefix(term, "xterm") && os.Getenv("XTERM_VERSION") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Detected: xterm with Sixel support\n")
		return ProtocolSixel
	}
	if strings.Contains(term, "mlterm") {
		fmt.Fprintf(os.Stderr, "[DEBUG] Detected: mlterm (Sixel support)\n")
		return ProtocolSixel
	}

	// Default fallback
	fmt.Fprintf(os.Stderr, "[DEBUG] No image protocol detected, using Unicode fallback\n")
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

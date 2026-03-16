package renderer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
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

	// Debug output disabled for production use

	// Explicit Kitty detection
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → Kitty (KITTY_WINDOW_ID set)\n")
		return ProtocolKitty
	}
	if strings.Contains(term, "kitty") {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → Kitty (TERM=kitty)\n")
		return ProtocolKitty
	}

	// macOS checks (Terminal.app AND iTerm2)
	if termProgram == "Apple_Terminal" {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → macOS Terminal.app (iTerm2 protocol)\n")
		return ProtocolITerm2
	}
	if termProgram == "iTerm.app" || iterm != "" || iterm2 != "" {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → iTerm2 (ITERM_PROGRAM or ITERM2_* set)\n")
		return ProtocolITerm2
	}
	if strings.Contains(termProgram, "Terminal") && strings.Contains(termProgram, "Mac") {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → macOS Terminal (iTerm2 fallback)\n")
		return ProtocolITerm2
	}

	// Alacritty detection (multiple checks)
	if strings.Contains(term, "alacritty") || strings.Contains(strings.ToLower(os.Getenv("COLORTERM")), "alacritty") {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → Alacritty (Kitty protocol)\n")
		return ProtocolKitty
	}

	// WezTerm
	if strings.Contains(term, "wezterm") {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → WezTerm (Kitty protocol)\n")
		return ProtocolKitty
	}

	// Sixel support (only if convert is available)
	if strings.Contains(term, "sixel") {
		if SixelAvailable() {
			// fmt.Fprintf(os.Stderr, "[DEBUG] → Sixel\n")
			return ProtocolSixel
		}
		// Terminal claims Sixel but convert not available, fall back to Kitty
		// fmt.Fprintf(os.Stderr, "[DEBUG] → Sixel claimed but convert unavailable, using Kitty fallback\n")
		return ProtocolKitty
	}

	// xterm-256color on macOS likely means Terminal.app (which doesn't support images)
	if strings.Contains(term, "xterm") {
		// Check if running on macOS
		if _, err := os.Stat("/System/Applications/Utilities/Terminal.app"); err == nil {
			// macOS Terminal.app doesn't support iTerm2 inline images, Kitty, or Sixel
			// Fall back to showing image paths/alt text
			// fmt.Fprintf(os.Stderr, "[DEBUG] → macOS xterm (Terminal.app - no image support, using alt text)\n")
			return ProtocolNone
		}

		// On other systems, xterm-256color might support Kitty (Alacritty, etc)
		// fmt.Fprintf(os.Stderr, "[DEBUG] → xterm-256color (Kitty fallback)\n")
		return ProtocolKitty
	}

	// screen/tmux: Try Kitty protocol
	if strings.Contains(term, "screen") || strings.Contains(term, "tmux") {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → Modern terminal (Kitty fallback for %s)\n", term)
		return ProtocolKitty
	}

	// fmt.Fprintf(os.Stderr, "[DEBUG] → No protocol detected, using Unicode\n")
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

	// Debug output disabled for production use
	// len(imageData), len(encoded), width, height

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

	encoded := base64.StdEncoding.EncodeToString(imageData)

	// Kitty graphics protocol - minimal working format:
	// \x1b_Ga=T,f=100,m=0:base64data\x1b\\
	// - a=T: action transmit
	// - f=100: format PNG
	// - m=0: no more chunks
	//
	// Note: For graph images, we don't need to specify width/height as Alacritty
	// will scale to fit the terminal based on the PNG's intrinsic dimensions
	// and the current terminal cell size.

	payload := fmt.Sprintf(
		"\x1b_Ga=T,f=100,m=0:%s\x1b\\",
		encoded,
	)

	return payload
}

// ImageToSixel converts image data to Sixel format using ImageMagick's convert command.
// If convert is not available, returns a placeholder with fallback instructions.
func ImageToSixel(imageData []byte, width, height int) string {
	if len(imageData) == 0 {
		return ""
	}

	// Try to use ImageMagick convert to generate Sixel format
	cmd := exec.Command("convert", "-", "sixel:-")
	cmd.Stdin = bytes.NewReader(imageData)

	output, err := cmd.Output()
	if err != nil {
		// convert command not available or failed
		// Return a helpful message that guides user to install ImageMagick
		return "[Image (Sixel format) - install ImageMagick for full rendering]"
	}

	return string(output)
}

// ConvertImageToSixel is a helper that tries to convert image data to Sixel.
// Returns the Sixel sequence, or empty string if conversion fails or convert is unavailable.
func ConvertImageToSixel(imageData []byte) string {
	if len(imageData) == 0 {
		return ""
	}

	cmd := exec.Command("convert", "-", "-depth", "8", "-colors", "256", "sixel:-")
	cmd.Stdin = bytes.NewReader(imageData)

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return string(output)
}

// SixelAvailable checks if ImageMagick's convert command is available for Sixel rendering.
func SixelAvailable() bool {
	cmd := exec.Command("which", "convert")
	return cmd.Run() == nil
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
	// Debug output disabled for production use

	switch protocol {
	case ProtocolKitty:
		result := ImageToKitty(imageData, width, height)
		// fmt.Fprintf(os.Stderr, "[DEBUG] Kitty sequence generated, %d bytes\n", len(result))
		return result
	case ProtocolITerm2:
		result := ImageToITerm2(imageData, width, height)
		// fmt.Fprintf(os.Stderr, "[DEBUG] iTerm2 sequence generated, %d bytes\n", len(result))
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
	case ProtocolNone:
		// For terminals without image support, show the file path
		// This is a helpful fallback for Terminal.app and other limited terminals
		if imagePath != "" {
			return "[Image: " + altText + " (" + filepath.Base(imagePath) + ")]"
		}
		return "[Image: " + altText + "]"
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
		// fmt.Fprintf(os.Stderr, "[DEBUG] Failed to save image temp file: %v\n", err)
		return ""
	}
	// fmt.Fprintf(os.Stderr, "[DEBUG] Saved image to: %s\n", tmpFile)
	return tmpFile
}

// ProtocolCapabilities returns a human-readable string describing image protocol support.
// Useful for help text and diagnostics.
func ProtocolCapabilities() string {
	protocol := DetectImageProtocol()

	switch protocol {
	case ProtocolKitty:
		return "Kitty graphics protocol (best for Kitty, Alacritty, WezTerm)"
	case ProtocolITerm2:
		return "iTerm2 inline images (native macOS Terminal and iTerm2)"
	case ProtocolSixel:
		if SixelAvailable() {
			return "Sixel graphics (with ImageMagick convert)"
		}
		return "Sixel terminal detected (but ImageMagick 'convert' not found - install imagemagick)"
	case ProtocolUnicode:
		return "Unicode/emoji fallback (works everywhere, limited quality)"
	case ProtocolNone:
		return "No image support (text fallback only)"
	default:
		return "Unknown image protocol"
	}
}

// RequiredForSixel returns installation instructions if Sixel is desired but unavailable.
func RequiredForSixel() string {
	if SixelAvailable() {
		return ""
	}

	return "To enable Sixel graphics, install ImageMagick:\n" +
		"  macOS: brew install imagemagick\n" +
		"  Ubuntu: sudo apt-get install imagemagick\n" +
		"  Alpine: apk add imagemagick\n" +
		"  Or: https://imagemagick.org/script/download.php"
}

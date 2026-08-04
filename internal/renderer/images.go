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
	ProtocolNone    ImageProtocol = iota // No image support; use alt text
	ProtocolITerm2                       // iTerm2 inline images
	ProtocolKitty                        // Kitty graphics protocol
	ProtocolSixel                        // Sixel image format
	ProtocolUnicode                      // Unicode block characters (emoji)
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

	// Real iTerm2 supports its own inline-image protocol.
	if termProgram == "iTerm.app" || iterm != "" || iterm2 != "" {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → iTerm2 (ITERM_PROGRAM or ITERM2_* set)\n")
		return ProtocolITerm2
	}

	// macOS Terminal.app has no native image protocol at all (confirmed:
	// no iTerm2, Kitty, or Sixel support — see arewesixelyet.com). Route it
	// to the real half-block ANSI-art renderer instead of guessing a
	// protocol it can't display.
	if termProgram == "Apple_Terminal" {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → macOS Terminal.app (half-block fallback)\n")
		return ProtocolUnicode
	}
	if strings.Contains(termProgram, "Terminal") && strings.Contains(termProgram, "Mac") {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → macOS Terminal (half-block fallback)\n")
		return ProtocolUnicode
	}

	// Alacritty detection (multiple checks). Mainline Alacritty has no
	// native image protocol either (no Sixel, Kitty, or iTerm2 support —
	// see alacritty/alacritty#910, open since 2021) — route to the
	// half-block renderer rather than a protocol it can't display.
	if strings.Contains(term, "alacritty") || strings.Contains(strings.ToLower(os.Getenv("COLORTERM")), "alacritty") {
		// fmt.Fprintf(os.Stderr, "[DEBUG] → Alacritty (half-block fallback)\n")
		return ProtocolUnicode
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

	// A bare "xterm"-family TERM with no other identifying signal is
	// genuinely ambiguous: it could be real Terminal.app or Alacritty
	// configured with TERM=xterm-256color for broad compatibility (e.g. over
	// SSH, where an "alacritty" terminfo entry may not be installed on the
	// remote host), or something else entirely. With no reliable native
	// protocol signal either way, don't gamble on one — fall through to the
	// half-block renderer at the bottom of this function, which needs
	// nothing but truecolor ANSI text support to work.

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

// kittyChunkSize is the Kitty graphics protocol's hard limit on
// base64-encoded payload bytes per escape-sequence chunk. Per the spec:
// "the base64 encoded image data ... must be no larger than 4096 bytes"
// per chunk, with non-final chunks required to be a multiple of 4 bytes
// (so they don't split a base64 quantum, which would corrupt decoding).
// 4096 is itself a multiple of 4, so slicing at exactly this size for
// every non-final chunk satisfies that requirement automatically.
const kittyChunkSize = 4096

// ImageToKitty encodes image data as a Kitty graphics protocol sequence.
// This protocol works in Kitty, WezTerm, and other terminals that
// implement it (notably not mainline Alacritty — see DetectImageProtocol).
// Returns an ANSI sequence that compatible terminals will render as an inline image.
func ImageToKitty(imageData []byte, width, height int) string {
	if len(imageData) == 0 {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(imageData)

	// Kitty graphics protocol format (per the spec: "All graphics escape
	// codes are of the form: <ESC>_G<control data>;<payload><ESC>\"):
	// \x1b_Ga=T,f=100,m=0;base64data\x1b\\
	// - a=T: action transmit-and-display
	// - f=100: format PNG
	// - m=0/m=1: final/more-chunks-follow
	// The control data and payload are separated by a semicolon, NOT a
	// colon — using a colon here silently produced no image at all in a
	// real Kitty terminal, since the terminal never finds the expected
	// separator and the whole command fails to parse.
	//
	// Any payload over kittyChunkSize bytes MUST be split across multiple
	// escape sequences (m=1 on every chunk but the last, m=0 on the
	// final one) — sending it all as a single "final" chunk, as this
	// function previously did unconditionally, exceeds the protocol's
	// hard per-chunk limit and the terminal fails to parse the command
	// at all, silently dropping the image (any real photo/screenshot is
	// almost always well over 4096 bytes once base64-encoded).
	//
	// Only the first chunk carries the full control-data set (a=/f=/etc);
	// subsequent chunks carry only m= (and optionally q=), per the spec.
	//
	// r= constrains the display footprint to exactly height terminal
	// rows (bmd-xqh): without it, Kitty renders at the PNG's natural
	// size (intrinsic pixel dimensions / current cell size). height is
	// capped by the caller (renderImage's maxNativeImageHeight) to a
	// value that safely fits within any realistic terminal window — a
	// first attempt let it scale with terminal width instead and
	// corrupted rendering in practice (verified against a real Kitty
	// terminal).
	//
	// Deliberately NOT also setting c= (width parameter unused for
	// Kitty, still used by ImageToITerm2 below): per the Kitty graphics
	// protocol spec, "if only one of either r or c is specified, the
	// other one is computed based on the source image aspect ratio, so
	// that the image is displayed without distortion" — specifying BOTH
	// forces the image into that exact box regardless of its real
	// aspect ratio, which visibly stretched/squashed a non-square image
	// in real-terminal testing. Letting Kitty derive columns from the
	// PNG's own dimensions (which it already has, having just decoded
	// the same payload for f=100) is more reliable than bmd computing
	// and asserting an aspect-correct width itself.
	var sb strings.Builder
	for i := 0; i < len(encoded); i += kittyChunkSize {
		end := i + kittyChunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		more := 0
		if end < len(encoded) {
			more = 1
		}
		if i == 0 {
			fmt.Fprintf(&sb, "\x1b_Ga=T,f=100,r=%d,m=%d;%s\x1b\\", height, more, encoded[i:end])
		} else {
			fmt.Fprintf(&sb, "\x1b_Gm=%d;%s\x1b\\", more, encoded[i:end])
		}
	}

	return sb.String()
}

// kittyDeleteAllImagesSeq is the Kitty graphics protocol command to remove
// every currently displayed image and free its cached data (a=d: delete
// action, d=A: all placements + underlying image data — lowercase "a"
// would only clear the on-screen placement and leave the image cached
// server-side). Kitty images are an out-of-band pixel overlay independent
// of the terminal's text grid: unlike normal characters, they are NOT
// automatically cleared when bmd redraws different text in the same
// screen cells (e.g. navigating to a different file), so bmd must send
// this explicitly at every navigation choke point to avoid a "ghost"
// image from the previous document persisting on screen (bmd-xqh).
const kittyDeleteAllImagesSeq = "\x1b_Ga=d,d=A\x1b\\"

// KittyDeleteAllImages returns the Kitty graphics protocol escape sequence
// that deletes every image currently displayed by the terminal. Callers
// should prepend this to rendered output at document-navigation choke
// points (loading a new file, returning to the directory/search view) so
// a previous document's image cannot persist as a ghost overlay once its
// text content is gone. Sending it when the terminal isn't Kitty, or when
// no image is on screen, is a harmless no-op (either the terminal ignores
// the unrecognized APC sequence, or the delete matches nothing).
func KittyDeleteAllImages() string {
	return kittyDeleteAllImagesSeq
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

// ImageToUnicode generates a Unicode half-block character representation of
// the image (the same technique pixterm/viu/chafa use as a terminal-agnostic
// fallback — see renderHalfBlockImage). This is the real fallback that works
// in any terminal supporting ANSI truecolor text, needed for terminals like
// Alacritty and macOS Terminal.app that have no native image protocol at
// all. Falls back to a plain alt-text placeholder if the image can't be
// decoded (e.g. an unsupported format).
func ImageToUnicode(imageData []byte, altText string, width int) string {
	if art, err := renderHalfBlockImage(imageData, width); err == nil {
		return art
	}
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
		// Real half-block ANSI-art rendering first — this is the actual
		// image, not a text placeholder, and works in any terminal with
		// truecolor text support (Alacritty, Terminal.app, etc). Only fall
		// back to save-temp-file+alt-text if the image can't be decoded.
		if art, err := renderHalfBlockImage(imageData, width); err == nil {
			return art
		}
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
		return "Kitty graphics protocol (native Kitty, WezTerm)"
	case ProtocolITerm2:
		return "iTerm2 inline images (native iTerm2)"
	case ProtocolSixel:
		if SixelAvailable() {
			return "Sixel graphics (with ImageMagick convert)"
		}
		return "Sixel terminal detected (but ImageMagick 'convert' not found - install imagemagick)"
	case ProtocolUnicode:
		return "Half-block ANSI art (works in any terminal with truecolor text support, e.g. Alacritty, Terminal.app)"
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

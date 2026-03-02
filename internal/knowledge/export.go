package knowledge

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExportArgs holds parsed arguments for CmdExport.
type ExportArgs struct {
	From   string // source directory to export
	Output string // output tar.gz file path
	DB     string // database path override
}

// KnowledgeMetadata is the metadata stored in knowledge.json inside the archive.
type KnowledgeMetadata struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	FileCount int       `json:"file_count"`
	DBSize    int64     `json:"db_size_bytes"`
	SourceDir string    `json:"source_dir"`
}

// ParseExportArgs parses raw CLI arguments for the export command.
//
// Usage: bmd export --from <path> --output <path> [--db <path>]
func ParseExportArgs(args []string) (*ExportArgs, error) {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var a ExportArgs
	fs.StringVar(&a.From, "from", ".", "Source directory to export")
	fs.StringVar(&a.Output, "output", "knowledge.tar.gz", "Output tar.gz file path")
	fs.StringVar(&a.DB, "db", "", "Database path override (default: .bmd/knowledge.db inside source dir)")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("export: %w", err)
	}

	// Positional argument overrides --from.
	if pos := fs.Args(); len(pos) > 0 {
		a.From = pos[0]
	}

	return &a, nil
}

// CmdExport implements `bmd export`. It scans the source directory, builds
// fresh indexes, and packages everything into a tar.gz archive.
func CmdExport(args []string) error {
	a, err := ParseExportArgs(args)
	if err != nil {
		return err
	}

	absFrom, err := filepath.Abs(a.From)
	if err != nil {
		return fmt.Errorf("export: resolve dir %q: %w", a.From, err)
	}

	// Verify source directory exists.
	info, err := os.Stat(absFrom)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("export: source directory %q does not exist or is not a directory", absFrom)
	}

	fmt.Fprintf(os.Stderr, "Exporting knowledge from %s...\n", absFrom)
	start := time.Now()

	// Step 1: Build fresh indexes.
	dbPath := a.DB
	if dbPath == "" {
		dbPath = filepath.Join(absFrom, ".bmd", "knowledge.db")
	}
	fmt.Fprintf(os.Stderr, "  Building fresh indexes...\n")
	if err := CmdIndex([]string{"--dir", absFrom, "--db", dbPath}); err != nil {
		return fmt.Errorf("export: index build: %w", err)
	}

	// Step 2: Scan markdown files.
	docs, err := ScanDirectory(absFrom)
	if err != nil {
		return fmt.Errorf("export: scan: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  %d markdown files found\n", len(docs))

	// Step 3: Get DB size.
	var dbSize int64
	if stat, err := os.Stat(dbPath); err == nil {
		dbSize = stat.Size()
	}

	// Step 4: Create metadata.
	meta := KnowledgeMetadata{
		Version:   "1.0.0",
		CreatedAt: time.Now().UTC(),
		FileCount: len(docs),
		DBSize:    dbSize,
		SourceDir: absFrom,
	}

	// Step 5: Create tar.gz archive.
	outputPath := a.Output
	if !strings.HasSuffix(strings.ToLower(outputPath), ".tar.gz") && !strings.HasSuffix(strings.ToLower(outputPath), ".tgz") {
		outputPath += ".tar.gz"
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("export: create output %q: %w", outputPath, err)
	}
	defer outFile.Close()

	gzw := gzip.NewWriter(outFile)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// Add knowledge.json metadata.
	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("export: marshal metadata: %w", err)
	}
	if err := addBytesToTar(tw, "knowledge.json", metaJSON); err != nil {
		return fmt.Errorf("export: add metadata: %w", err)
	}

	// Add markdown files (preserving relative paths).
	for _, doc := range docs {
		relPath := filepath.ToSlash(doc.RelPath)
		if err := addFileToTar(tw, doc.Path, relPath); err != nil {
			return fmt.Errorf("export: add file %q: %w", relPath, err)
		}
	}

	// Add database file.
	if _, err := os.Stat(dbPath); err == nil {
		if err := addFileToTar(tw, dbPath, ".bmd/knowledge.db"); err != nil {
			return fmt.Errorf("export: add database: %w", err)
		}
	}

	elapsed := time.Since(start)
	absOutput, _ := filepath.Abs(outputPath)
	outStat, _ := os.Stat(outputPath)
	var sizeStr string
	if outStat != nil {
		sizeStr = humanBytes(outStat.Size())
	}

	fmt.Fprintf(os.Stderr, "  Archive: %s (%s)\n", absOutput, sizeStr)
	fmt.Fprintf(os.Stderr, "  Files: %d markdown + database + metadata\n", len(docs))
	fmt.Fprintf(os.Stderr, "  Completed in %dms\n", elapsed.Milliseconds())

	return nil
}

// addFileToTar adds a file from disk to the tar archive at the given archive path.
func addFileToTar(tw *tar.Writer, diskPath, archivePath string) error {
	f, err := os.Open(diskPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    archivePath,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, f)
	return err
}

// addBytesToTar adds in-memory bytes to the tar archive at the given path.
func addBytesToTar(tw *tar.Writer, archivePath string, data []byte) error {
	header := &tar.Header{
		Name:    archivePath,
		Size:    int64(len(data)),
		Mode:    0o644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err := tw.Write(data)
	return err
}

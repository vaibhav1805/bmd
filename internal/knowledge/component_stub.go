package knowledge

// PageIndex semantic search (trees, LLM querying) moved to graphmd.
// LoadTreeFiles stays as a stub so SearchAllDocumentsPageIndex can report
// ErrPageIndexNotAvailable and let callers fall back to BM25.

type FileTree struct {
	ID       string
	Children []*FileTree
}

func LoadTreeFiles(dir string) ([]FileTree, error) {
	return nil, nil
}

// filenameStem returns the identifier used to match a --service flag against
// a graph node when no exact or case-insensitive ID match is found.
func filenameStem(path string) string {
	return path
}

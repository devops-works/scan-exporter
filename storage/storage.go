package storage

// ListManager must implement those methods help us find differences with previous runs
type ListManager interface {
	ReadList(string) ([]string, error)
	ReplaceList(string, []string) error
}

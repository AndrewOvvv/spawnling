package backup

import "time"

//go:generate mockery --name=ArchiveStore --output=./mocks

// ArchiveStore abstracts compressed archive creation and retrieval.
type ArchiveStore interface {
	Create(instanceName string, worldPath string) (archivePath string, err error)
	List(instanceName string) ([]ArchiveMeta, error)
	Restore(archivePath string, targetPath string) error
	Prune(instanceName string, keepLast int) error
}

type ArchiveMeta struct {
	Path      string
	CreatedAt time.Time
	SizeBytes int64
}

package backup

type BackupService struct {
	store ArchiveStore
}

func NewBackupService(store ArchiveStore) *BackupService {
	return &BackupService{store: store}
}

func (s *BackupService) Create(instanceName string, worldPath string) (string, error) {
	return s.store.Create(instanceName, worldPath)
}

func (s *BackupService) List(instanceName string) ([]ArchiveMeta, error) {
	return s.store.List(instanceName)
}

func (s *BackupService) Restore(archivePath string, targetPath string) error {
	return s.store.Restore(archivePath, targetPath)
}

func (s *BackupService) Prune(instanceName string, keepLast int) error {
	return s.store.Prune(instanceName, keepLast)
}

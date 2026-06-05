package process

//go:generate mockery --name=OSProcess --output=./mocks
//go:generate mockery --name=FileSystem --output=./mocks

// OSProcess abstracts interaction with a running OS process.
type OSProcess interface {
	Pid() int
	Kill() error
	Wait() error
}

// FileSystem abstracts file I/O needed by the process backend.
type FileSystem interface {
	MkdirAll(path string, perm uint32) error
	WriteFile(path string, data []byte, perm uint32) error
	ReadFile(path string) ([]byte, error)
	Exists(path string) (bool, error)
}

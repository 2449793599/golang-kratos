package config

// KeyValue is config key value.
type KeyValue struct {
	Key    string // 文件名称
	Value  []byte // 文件内容
	Format string // 文件格式
}

// Source is config source.
type Source interface {
	Load() ([]*KeyValue, error)
	Watch() (Watcher, error)
}

// Watcher watches a source for changes.
type Watcher interface {
	Next() ([]*KeyValue, error)
	Stop() error
}

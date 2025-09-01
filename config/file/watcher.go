package file

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"

	"github.com/go-kratos/kratos/v2/config"
)

var _ config.Watcher = (*watcher)(nil)

// *********************************************************************************************************************
type watcher struct {
	f  *file             //
	fw *fsnotify.Watcher // 文件系统监控器

	ctx    context.Context    // 上下文
	cancel context.CancelFunc // 上下文取消函数
}

func newWatcher(f *file) (config.Watcher, error) {

	fw, err := fsnotify.NewWatcher()

	if err != nil {
		return nil, err
	}

	if err := fw.Add(f.path); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &watcher{f: f, fw: fw, ctx: ctx, cancel: cancel}, nil

}

func (w *watcher) Next() ([]*config.KeyValue, error) {

	select {
	case <-w.ctx.Done(): // 上下文取消

		return nil, w.ctx.Err()

	case event := <-w.fw.Events:

		if event.Op == fsnotify.Rename {

			if _, err := os.Stat(event.Name); err == nil || os.IsExist(err) {
				if err := w.fw.Add(event.Name); err != nil { // 重新监控新文件
					return nil, err
				}
			}

		}

		fi, err := os.Stat(w.f.path)

		if err != nil {
			return nil, err
		}

		path := w.f.path

		if fi.IsDir() {
			path = filepath.Join(w.f.path, filepath.Base(event.Name))
		}

		kv, err := w.f.loadFile(path)

		if err != nil {
			return nil, err
		}

		return []*config.KeyValue{kv}, nil

	case err := <-w.fw.Errors:

		return nil, err

	}

}

func (w *watcher) Stop() error {

	w.cancel()

	return w.fw.Close()

}

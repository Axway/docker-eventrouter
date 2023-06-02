package tools

import (
	"io/fs"
	"os"
	"path"
	"time"

	"axway.com/qlt-router/src/log"
	"github.com/esimov/gogu"
)

func FileRotateList(ctx, filename string) ([]string, error) {
	dir := path.Dir(filename)
	basename := path.Base(filename)
	// basenameLength = len(basename)

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Errorc(ctx, "error while rotating : readdir", "filename", filename, "err", err)
		return nil, err
	}
	// log.Debugc(ctx, "filerotateList", "filename", filename, "files", gogu.Map(files, func(f fs.DirEntry) string { return f.Name() }))
	// fmt.Println(ctx, "filerotateList", "filename", filename, "files", files)

	// List file with Filename prefix
	rotated := gogu.Filter(files, func(file fs.DirEntry) bool {
		return len(file.Name()) > len(basename) && file.Name()[:len(basename)] == basename
	})
	// log.Debugc(ctx, "filerotateList", "filename", filename, "files", gogu.Map(rotated, func(f fs.DirEntry) string { return f.Name() }))

	names := gogu.Map(rotated, func(f fs.DirEntry) string { return path.Join(dir, f.Name()) })
	return names, nil
}

func FileRotate(ctx string, filename string, maxfile int) error {
	rotated, err := FileRotateList(ctx, filename)
	if err != nil {
		return err
	}

	// Only keep Maxfile rotated files
	for i := 0; i < len(rotated)-maxfile+1; i++ {
		fname := rotated[i]
		log.Infoc(ctx, "Removing ", "filename", fname)
		err := os.Remove(fname)
		if err != nil {
			log.Errorc(ctx, "error while rotating : removing old", "filename", fname, "err", err)
			return err
		}
	}

	postfix := time.Now().UTC().Format(time.RFC3339Nano)
	newfname := filename + "-" + postfix
	err = os.Rename(filename, newfname)
	if err != nil {
		log.Errorc(ctx, "error while rotating : rename error", "filename", filename, "new", newfname, "err", err)
		return err
	}
	return nil
}

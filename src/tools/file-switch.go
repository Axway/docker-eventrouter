package tools

import (
	"io/fs"
	"os"
	"path"
	"time"
	"fmt"

	"axway.com/qlt-router/src/log"
	"github.com/esimov/gogu"
)

func TimestampedFilename(ctx, filename string) (string) {
	dir := path.Dir(filename)
	basename := path.Base(filename)
	extention := path.Ext(filename)

	/* removing extension of filename */
	basename = basename[:len(basename)-len(extention)]
	postfix := time.Now().UTC().Format(time.RFC3339Nano)
	timestampedFilename := basename + "-" + postfix + extention

	return path.Join(dir,timestampedFilename)
}

func NextFile(ctx, filename string, lastfilename string) (string, error) {
	var nextfilename string

	previousFiles, err := FileSwitchList(ctx, filename)
	if err != nil || len(previousFiles) == 0 {
		return lastfilename, err
	}

	nextfilename = previousFiles[len(previousFiles)-1]
	for i := len(previousFiles)-1; i >= 0; i-- {
		if previousFiles[i] == lastfilename {
			break;
		}
		nextfilename = previousFiles[i]
	}
	return nextfilename, nil
}

func FileToUse(ctx, filename string) (string) {
	newfname := TimestampedFilename(ctx, filename)

	filesUsed, err := FileSwitchList(ctx, filename)
	if err != nil {
		return newfname
	}
	if len(filesUsed) == 0 {
		return newfname
	}
	return filesUsed[len(filesUsed)-1]
}

func FileSwitchList(ctx, filename string) ([]string, error) {
	dir := path.Dir(filename)
	basename := path.Base(filename)
	extention := path.Ext(filename)

	/* removing extension of filename */
	basename = basename[:len(basename)-len(extention)]

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Errorc(ctx, "error on ReadDir : readdir", "filename", filename, "err", err)
		return nil, err
	}

	// List file with basename prefix
	previousFiles := gogu.Filter(files, func(file fs.DirEntry) bool {
		return len(file.Name()) > len(basename) && file.Name()[:len(basename)] == basename
	})

	names := gogu.Map(previousFiles, func(f fs.DirEntry) string { return path.Join(dir, f.Name()) })
	//log.Debugc(ctx, "FileSwitchList", "filename", filename, "files", names)
	return names, nil
}

func FileSwitch(ctx string, filename string, maxfile int) (string, error) {
	newfname := TimestampedFilename(ctx, filename)

	previousFiles, err := FileSwitchList(ctx, filename)
	if err != nil {
		return newfname, err
	}

	if maxfile > 0 && len(previousFiles) >= maxfile {
		fmt.Println("Need to remove file ", fmt.Sprint(len(previousFiles)), ">=", fmt.Sprint(maxfile))
		// Only keep Maxfile old files
		for i := 0; i < len(previousFiles)-maxfile+1; i++ {
			fname := previousFiles[i]
			log.Infoc(ctx, "Removing ", "filename", fname)
			err := os.Remove(fname)
			if err != nil {
				log.Errorc(ctx, "error while switching : removing old", "filename", fname, "err", err)
				return newfname, err
			}
		}
	}

	return newfname, nil
}

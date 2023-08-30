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

func TimestampedFilename(ctx, filenamePrefix string, filenameSuffix string) (string) {
	postfix := time.Now().UTC().Format(time.RFC3339Nano)
	timestampedFilename := filenamePrefix + "." + postfix
	if len(filenameSuffix) > 0 {
		timestampedFilename += "." + filenameSuffix
	}

	return timestampedFilename
}

func NextFile(ctx, filenamePrefix string, filenameSuffix string, lastfilename string) (string, error) {
	var nextfilename string

	previousFiles, err := FileSwitchList(ctx, filenamePrefix, filenameSuffix)
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

func FileToUse(ctx, filenamePrefix string, filenameSuffix string) (string) {
	newfname := TimestampedFilename(ctx, filenamePrefix, filenameSuffix)

	filesUsed, err := FileSwitchList(ctx, filenamePrefix, filenameSuffix)
	if err != nil {
		return newfname
	}
	if len(filesUsed) == 0 {
		return newfname
	}
	return filesUsed[len(filesUsed)-1]
}

func FileSwitchList(ctx, filenamePrefix string, filenameSuffix string) ([]string, error) {
	dir := path.Dir(filenamePrefix)

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Errorc(ctx, "error on ReadDir : readdir", "filenamePrefix", filenamePrefix, "err", err)
		return nil, err
	}

	// List file with basename prefix
	previousFiles := gogu.Filter(files, func(file fs.DirEntry) bool {
		basename := path.Base(filenamePrefix)
		lenBase := len(basename)
		lenFile := len(file.Name())
		lenSuf := len(filenameSuffix)

		return lenFile > lenBase && file.Name()[:lenBase] == basename &&
			   lenFile > lenSuf && file.Name()[lenFile-lenSuf:] == filenameSuffix
	})

	names := gogu.Map(previousFiles, func(f fs.DirEntry) string { return path.Join(dir, f.Name()) })
	//log.Debugc(ctx, "FileSwitchList", "filename", filename, "files", names)
	return names, nil
}

func FileSwitch(ctx string, filenamePrefix string, filenameSuffix string, maxfile int) (string, error) {
	newfname := TimestampedFilename(ctx, filenamePrefix, filenameSuffix)

	previousFiles, err := FileSwitchList(ctx, filenamePrefix, filenameSuffix)
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

package tools

import (
	"io/fs"
	"os"
	"path"
	"time"
	"fmt"
	"regexp"

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

	filenames, err := FileSwitchList(ctx, filenamePrefix, filenameSuffix)
	if err != nil || len(filenames) == 0 {
		return lastfilename, err
	}

	nextfilename = filenames[len(filenames)-1]
	for i := len(filenames)-1; i >= 0; i-- {
		if filenames[i] == lastfilename {
			break;
		}
		nextfilename = filenames[i]
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
	//log.Debugc(ctx, "FileSwitchList", "files", files)

	// List file with pattern: prefix.(time.RFC3339Nano)(.suffix)
	pattern := path.Base(filenamePrefix) + `\.` + `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z\.*` + filenameSuffix
	r, _ := regexp.Compile(pattern)
	filtered := gogu.Filter(files, func(file fs.DirEntry) bool {
		return r.MatchString(file.Name())
	})

	names := gogu.Map(filtered, func(f fs.DirEntry) string { return path.Join(dir, f.Name()) })
	//log.Debugc(ctx, "FileSwitchList", "names", names)
	return names, nil
}

func FileSwitch(ctx string, filenamePrefix string, filenameSuffix string, maxfile int) (string, error) {
	newfname := TimestampedFilename(ctx, filenamePrefix, filenameSuffix)

	filenames, err := FileSwitchList(ctx, filenamePrefix, filenameSuffix)
	if err != nil {
		return newfname, err
	}

	if maxfile > 0 && len(filenames) >= maxfile {
		fmt.Println("Need to remove file ", fmt.Sprint(len(filenames)), ">=", fmt.Sprint(maxfile))
		// Only keep Maxfile old files
		for i := 0; i < len(filenames)-maxfile+1; i++ {
			fname := filenames[i]
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

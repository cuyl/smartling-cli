package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Smartling/api-sdk-go"
	"github.com/gobwas/glob"
	"github.com/reconquest/hierr-go"
)

func globFilesRemote(
	client *smartling.Client,
	project string,
	uri string,
) ([]smartling.File, error) {
	if uri == "" {
		uri = "**"
	}

	pattern, err := glob.Compile(uri, '/')
	if err != nil {
		return nil, NewError(
			err,
			"Search file URI is malformed. Check out help for more "+
				"information about search patterns.",
		)
	}

	request := smartling.FilesListRequest{}

	files, err := client.ListAllFiles(project, request)
	if err != nil {
		if _, ok := err.(smartling.NotFoundError); ok {
			return nil, ProjectNotFoundError{}
		}

		return nil, hierr.Errorf(
			err,
			`unable to list files in project "%s"`,
			project,
		)
	}

	result := []smartling.File{}

	for _, file := range files {
		if pattern.Match(file.FileURI) {
			result = append(result, file)
		}
	}

	if len(result) == 0 {
		return nil, NewError(
			fmt.Errorf(
				"no files found on the remote server matching provided pattern",
			),

			"Check that file URI pattern is correct.",
		)
	}

	return result, nil
}

func getDirectoryFromPattern(mask string) (string, string) {
	matches := regexp.MustCompile(`^([^*?{}\[\]]+)/(.+)$`).FindStringSubmatch(
		mask,
	)

	if len(matches) < 2 {
		return "", mask
	}

	return matches[1], matches[2]
}

func globFilesLocallyFunc(
	directory string,
	base string,
	mask string,
) ([]string, error) {
	if strings.HasPrefix(base, "/") {
		directory = base
	} else {
		directory = filepath.Join(directory, base)
	}

	pattern, err := glob.Compile(mask, '/')
	if err != nil {
		return nil, NewError(
			err,
			"Search file pattern is malformed. Check out help for more "+
				"information about search patterns.",
		)
	}

	if _, err := os.Stat(filepath.Join(directory, mask)); err == nil {
		return []string{filepath.Join(directory, mask)}, nil
	}

	var result []string

	err = filepath.Walk(
		directory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			path = strings.TrimPrefix(path, directory)
			path = strings.TrimPrefix(path, "/")

			if pattern.Match(path) {
				result = append(
					result,
					filepath.Join(directory, path),
				)
			}

			return nil
		},
	)

	if err != nil {
		return nil, hierr.Errorf(
			err,
			`unable to walk down files in dir "%s"`,
			directory,
		)
	}

	return result, nil
}

var globFilesLocally = globFilesLocallyFunc

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"strings"

	smartling "github.com/Smartling/api-sdk-go"
	"github.com/gobwas/glob"
	"github.com/reconquest/hierr-go"
)

func SearchLocale(list []smartling.Locale, locale string) (*smartling.Locale) {
	for _, _locale := range list {
		if _locale.LocaleID == locale {
			return &_locale
		}
	}
	return nil
}

type GlobConfigPair struct {
	Glob glob.Glob
	Config FileConfig
}

type UploadItem struct {
	SourceFile smartling.File
	TranslationFile string
	Locale string
}

func doFilesTranslationUpdate(
	client *smartling.Client,
	config Config,
	args map[string]interface{},
) error {
	var (
		project     = config.ProjectID
		uri, useUri   = args["[uri]"].(string)
		// sourceLocale string
		branch, useBranch   = args["--branch"].(string)
		
	)
	if !useUri || uri == "" {
		uri = "**"
	}
	if useBranch {
		uri = branch + "/**"
	}

	// config.Locales
	info, err := client.GetProjectDetails(project)

	if err != nil {
		return err
	}
	
	// sourceLocale = info.SourceLocaleID

	// TODO: add project target language
	// https://api-reference.smartling.com/#operation/addLocaleToProject
	// Smartling/api-sdk-go do not have the api above.
	// for _, locale := range config.Locales {
	// 	if l := SearchLocale(info.TargetLocales, locale.Smartling); l == nil {
	// 		// client.up
	// 	}
	// } 
	files, err := globFilesRemote(
		client,
		project,
		uri,
	)

	if err != nil {
		return err
	}

	var globConfigList []GlobConfigPair
	for pattern, section := range config.Files {
		pattern, err := glob.Compile(pattern, '/')
		if err != nil {
			NewError(
				err,
				"Search file URI is malformed. Check out help for more "+
					"information about search patterns.",
			)
			continue
		}
		globConfigList = append(globConfigList, GlobConfigPair {
			Glob: pattern,
			Config: section,
		})
	}

	var uploadItems []UploadItem
	for _, file :=  range files {
		targetFileURI := file.FileURI
		if useBranch {
			targetFileURI = strings.TrimPrefix(file.FileURI, branch + "/")
		}
		// if section.Push.Type != "" {
		// 	patterns = append(patterns, pattern)
		// }

		for _, globConfig := range globConfigList {
			if globConfig.Glob.Match(targetFileURI) {
				for _, locale := range info.TargetLocales {
					AppLocale := locale.LocaleID
					if _AppLocale, ok := config.LocaleToAppLocaleMap[locale.LocaleID]; ok {
						AppLocale = _AppLocale
					}

					path, err := executeFileFormat(
						config,
						file,
						globConfig.Config.Pull.Format,
						usePullFormat,
						map[string]interface{}{
							"AppLocale": AppLocale,
							"FileURI":   targetFileURI,
							"Locale":    locale.LocaleID,
						},
					)
					if err != nil {
						continue
					}
					if _, err := os.Stat(filepath.Join(filepath.Dir(config.path), path)); err == nil {
						uploadItems = append(uploadItems, UploadItem {
							SourceFile:      file,
							TranslationFile: path,
							Locale:          locale.LocaleID,
						})
						
					}
				}
			}
		}
	}

	pool := NewThreadPool(config.Threads)

	for _, item := range uploadItems {
		// func closure required to pass different file objects to goroutines
		func(item UploadItem) {
			pool.Do(func(){

				contents, err := ioutil.ReadFile(item.TranslationFile)

				if err != nil {
					logger.Error(NewError(
						hierr.Errorf(err, "unable to read file for import"),
						"Check that specified file exists and you have permissions "+
							"to read it.",
					))
					return
				}

				request := smartling.ImportRequest{}
				request.File = contents
				request.FileType = item.SourceFile.FileType
				request.FileURI = item.SourceFile.FileURI
				request.TranslationState = smartling.TranslationStatePublished

				if args["--post-translation"].(bool) {
					request.TranslationState = smartling.TranslationStatePostTranslation
				}

				if args["--overwrite"].(bool) {
					request.Overwrite = true
				}
				logger.Infof("upload translations params: FileURI: %s TranslationState: %s Overwrite: %t",
				request.FileURI,
				request.TranslationState,
				request.Overwrite,
			  )
				result, err := client.Import(project, item.Locale, request)

				if err != nil {
					logger.Error(hierr.Errorf(
						err,
						`unable to import file "%s" (original "%s")`,
						item.TranslationFile,
						item.SourceFile.FileURI,
					))
				}
			
				fmt.Printf(
					"%s imported [%d strings %d words]\n",
					item.TranslationFile,
					result.StringCount,
					result.WordCount,
				)
			})
		}(item)
	}

	pool.Wait()

	return nil
}

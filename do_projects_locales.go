package main

import (
	"fmt"
	"io"
	"os"

	smartling "github.com/Smartling/api-sdk-go"
	"github.com/reconquest/hierr-go"
)

func doProjectsLocales(
	client *smartling.Client,
	config Config,
	args map[string]interface{},
) error {
	var (
		project   = config.ProjectID
		short, _  = args["--short"].(bool)
		source, _ = args["--source"].(bool)
	)

	if args["--format"] == nil {
		args["--format"] = defaultProjectsLocalesFormat
	}

	format, err := compileFormat(args["--format"].(string))
	if err != nil {
		return err
	}

	details, err := client.GetProjectDetails(project)
	if err != nil {
		if _, ok := err.(smartling.NotFoundError); ok {
			return ProjectNotFoundError{}
		}

		return hierr.Errorf(
			err,
			`unable to get project "%s" details`,
			project,
		)
	}

	table := NewTableWriter(os.Stdout)

	if source {
		if short {
			fmt.Fprintf(table, "%s\n", details.SourceLocaleID)
		} else {
			fmt.Fprintf(
				table,
				"%s\t%s\n",
				details.SourceLocaleID,
				details.SourceLocaleDescription,
			)
		}
	} else {
		for _, locale := range details.TargetLocales {
			if short {
				fmt.Fprintf(table, "%s\n", locale.LocaleID)
			} else {
				row, err := format.Execute(locale)
				if err != nil {
					return err
				}

				_, err = io.WriteString(table, row)
				if err != nil {
					return hierr.Errorf(
						err,
						"unable to write row to output table",
					)
				}
			}
		}
	}

	err = RenderTable(table)
	if err != nil {
		return err
	}

	return nil
}

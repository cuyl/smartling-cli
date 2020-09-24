package main

import (
	"fmt"
	"os"

	smartling "github.com/Smartling/api-sdk-go"
	"github.com/reconquest/hierr-go"
)

func doProjectsList(
	client *smartling.Client,
	config Config,
	args map[string]interface{},
) error {
	var (
		short = args["--short"].(bool)
	)

	projects, err := client.ListProjects(
		config.AccountID,
		smartling.ProjectsListRequest{},
	)
	if err != nil {
		return NewError(
			hierr.Errorf(err, "unable to list projects"),
			"",
		)
	}

	table := NewTableWriter(os.Stdout)

	for _, project := range projects.Items {
		if short {
			fmt.Fprintln(table, project.ProjectID)
		} else {
			fmt.Fprintf(
				table,
				"%s\t%s\t%s\n",
				project.ProjectID,
				project.ProjectName,
				project.SourceLocaleID,
			)
		}
	}

	err = RenderTable(table)
	if err != nil {
		return err
	}

	return nil
}

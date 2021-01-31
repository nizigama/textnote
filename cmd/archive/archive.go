package archive

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dkaslovsky/textnote/pkg/archive"
	"github.com/dkaslovsky/textnote/pkg/config"
	"github.com/dkaslovsky/textnote/pkg/file"
	"github.com/dkaslovsky/textnote/pkg/template"
	"github.com/spf13/cobra"
)

type commandOptions struct {
	delete  bool
	noWrite bool
}

// CreateArchiveCmd creates the today subcommand
func CreateArchiveCmd() *cobra.Command {
	cmdOpts := commandOptions{}
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "organize an archive of notes",
		Long:  "organize notes into time-based archive groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := config.LoadOrCreate()
			if err != nil {
				return err
			}
			return run(opts, cmdOpts)
		},
	}
	attachOpts(cmd, &cmdOpts)
	return cmd
}

func attachOpts(cmd *cobra.Command, cmdOpts *commandOptions) {
	flags := cmd.Flags()
	flags.BoolVarP(&cmdOpts.delete, "delete", "x", false, "delete individual files after archiving")
	flags.BoolVarP(&cmdOpts.noWrite, "nowrite", "n", false, "disable writing archive file (helpful for deleting previously archived files)")
}

func run(templateOpts config.Opts, cmdOpts commandOptions) error {
	archived := []string{}
	archiver := archive.NewArchiver(templateOpts, file.NewReadWriter(), time.Now())

	files, err := ioutil.ReadDir(config.AppDir)
	if err != nil {
		return err
	}

	// add template files to archiver
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if !template.IsTemplateFile(f.Name(), templateOpts.File) {
			continue
		}

		err := archiver.Add(f.Name())
		if err != nil {
			log.Printf("skipping unarchivable file [%s]: %s", f.Name(), err)
			continue
		}

		archived = append(archived, f.Name())
	}

	// write archive files
	if !cmdOpts.noWrite {
		err = archiver.Write()
		if err != nil {
			return err
		}
	}

	// return if not deleting archived files
	if !cmdOpts.delete {
		return nil
	}

	// delete individual archived files
	for _, f := range archived {
		err = os.Remove(filepath.Join(config.AppDir, f))
		if err != nil {
			log.Printf("unable to remove file [%s]: %s", f, err)
			continue
		}
	}

	return nil
}

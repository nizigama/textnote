package archive

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/dkaslovsky/textnote/pkg/config"
	"github.com/dkaslovsky/textnote/pkg/file"
	"github.com/dkaslovsky/textnote/pkg/template"
	"github.com/pkg/errors"
)

// Archiver consolidates templates into archives
type Archiver struct {
	opts config.Opts
	rw   readWriter
	date time.Time // timestamp for calculating if a file is old enough to be archived

	// archive templates by month keyed by formatted month timestamp
	Months map[string]*template.MonthArchiveTemplate
}

// NewArchiver constructs a new Archiver
func NewArchiver(opts config.Opts, rw readWriter, date time.Time) *Archiver {
	return &Archiver{
		opts: opts,
		rw:   rw,
		date: date,

		Months: map[string]*template.MonthArchiveTemplate{},
	}
}

// readWriter is the interface for executing file operations
type readWriter interface {
	Read(file.ReadWriteable) error
	Overwrite(file.ReadWriteable) error
	Exists(file.ReadWriteable) bool
}

// Add adds a file to the archive
func (a *Archiver) Add(fileName string) error {
	fileDate, err := parseFileName(fileName, a.opts.File.TimeFormat)
	if err != nil {
		errors.Wrapf(err, "cannot add unparsable file name [%s] to archive", fileName)
	}

	// recent files are not archived
	if a.date.Sub(fileDate).Hours() <= float64(a.opts.Archive.AfterDays*24) {
		return nil
	}

	t := template.NewTemplate(a.opts, fileDate)
	err = a.rw.Read(t)
	if err != nil {
		return errors.Wrapf(err, "cannot add unreadable file [%s] to archive", fileName)
	}

	monthKey := fileDate.Format(a.opts.Archive.MonthTimeFormat)
	if _, found := a.Months[monthKey]; !found {
		a.Months[monthKey] = template.NewMonthArchiveTemplate(a.opts, fileDate)
	}

	archive := a.Months[monthKey]
	for _, section := range a.opts.Section.Names {
		err := archive.ArchiveSectionContents(t, section)
		if err != nil {
			return errors.Wrapf(err, "cannot add contents from [%s] to archive", fileName)
		}
	}
	return nil
}

// Write writes all of the archive templates stored in the Archiver
func (a *Archiver) Write() error {
	for _, t := range a.Months {
		if a.rw.Exists(t) {
			existing := template.NewMonthArchiveTemplate(a.opts, t.GetDate())
			err := a.rw.Read(existing)
			if err != nil {
				return errors.Wrapf(err, "unable to open existing archive file [%s]", existing.GetFilePath())
			}
			err = t.Merge(existing)
			if err != nil {
				return errors.Wrapf(err, "unable to from merge existing archive file [%s]", existing.GetFilePath())
			}
		}

		err := a.rw.Overwrite(t)
		if err != nil {
			return errors.Wrapf(err, "failed to write archive file [%s]", t.GetFilePath())
		}
		log.Printf("wrote archive file [%s]", t.GetFilePath())
	}
	return nil
}

func parseFileName(fileName string, format string) (time.Time, error) {
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	return time.Parse(format, name)
}

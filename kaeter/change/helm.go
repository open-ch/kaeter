package change

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// HelmChange contains the list of modified Helm charts
type HelmChange struct {
	Charts []string
}

// HelmCheck performs change detection on Helm chart and returns the summary in HelmChange
func (d *Detector) HelmCheck(changes *Information) (c HelmChange) {
	// Resolve all Helm Charts
	charts, _ := d.findAllHelmCharts(d.RootPath)

	// Make a list of all touched files
	allTouchedFiles := append(append(changes.Files.Added, changes.Files.Modified...), changes.Files.Removed...)

	// Extract the list of affected Helm charts
	return HelmChange{
		Charts: d.matchFilesAndCharts(allTouchedFiles, charts),
	}
}

// finAllHelmCharts searches the repo for Helm charts. A Helm chart is identified by having a file called
// Chart.yaml
func (d *Detector) findAllHelmCharts(gitRoot string) (charts []string, err error) {
	charts = make([]string, 0)
	err = filepath.WalkDir(gitRoot, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		basename := filepath.Base(path)
		if basename == "Chart.yaml" {
			// Make the path relative to the repo root and terminate it with a slash
			chartPath := strings.TrimLeft(strings.TrimPrefix(filepath.Dir(path), gitRoot), "/") + "/"
			charts = append(charts, chartPath)
			d.Logger.Debugf("DetectorHelm: Found chart at %s", chartPath)
		}
		return nil
	})
	return charts, err
}

// matchFilesAndCharts matches files to helm charts
func (*Detector) matchFilesAndCharts(files []string, chartPaths []string) (charts []string) {
	// Find the Helm charts that have any of their file modified by looking for files that are in
	// a subfolder of the Helm chart folder.
	charts = make([]string, 0)
	for _, chart := range chartPaths {
		for _, file := range files {
			if strings.HasPrefix(file, chart) && !contains(charts, chart) {
				charts = append(charts, chart)
			}
		}
	}
	sort.Strings(charts)
	return charts
}

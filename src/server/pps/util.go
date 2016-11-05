package pps

import (
	"fmt"

	"github.com/sjezewski/pachyderm/src/client/pfs"
	ppsclient "github.com/sjezewski/pachyderm/src/client/pps"
)

// JobRepo creates a pfs repo for a given job.
func JobRepo(job *ppsclient.Job) *pfs.Repo {
	return &pfs.Repo{Name: fmt.Sprintf("job_%s", job.ID)}
}

// PipelineRepo creates a pfs repo for a given pipeline.
func PipelineRepo(pipeline *ppsclient.Pipeline) *pfs.Repo {
	return &pfs.Repo{Name: pipeline.Name}
}

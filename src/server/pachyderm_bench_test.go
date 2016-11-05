package server

import (
	"fmt"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/pachyderm/pachyderm/src/client"
	pfsclient "github.com/pachyderm/pachyderm/src/client/pfs"
	"github.com/pachyderm/pachyderm/src/client/pkg/require"
	ppsclient "github.com/pachyderm/pachyderm/src/client/pps"
	"github.com/pachyderm/pachyderm/src/server/pkg/workload"
	"golang.org/x/sync/errgroup"
)

const (
	MB = 1024 * 1024
)

type CountWriter struct {
	count int64
}

func (w *CountWriter) Write(p []byte) (int, error) {
	w.count += int64(len(p))
	return len(p), nil
}

func BenchmarkPachyderm(b *testing.B) {
	repo := uniqueString("BenchmarkPachyderm")
	c, err := client.NewInCluster()
	require.NoError(b, err)
	require.NoError(b, c.CreateRepo(repo))
	nFiles := 1000

	commit, err := c.StartCommit(repo, "master")
	require.NoError(b, err)
	if !b.Run(fmt.Sprintf("Put%dFiles", nFiles), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var eg errgroup.Group
			for k := 0; k < nFiles; k++ {
				k := k
				eg.Go(func() error {
					rand := rand.New(rand.NewSource(int64(time.Now().UnixNano())))
					_, err := c.PutFile(repo, "master", fmt.Sprintf("file%d", k), workload.NewReader(rand, MB))
					return err
				})
			}
			b.SetBytes(int64(nFiles * MB))
			require.NoError(b, eg.Wait())
		}
	}) {
		return
	}
	require.NoError(b, c.FinishCommit(repo, "master"))

	if !b.Run(fmt.Sprintf("Get%dFiles", nFiles), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var eg errgroup.Group
			w := &CountWriter{}
			defer func() { b.SetBytes(w.count) }()
			for k := 0; k < nFiles; k++ {
				k := k
				eg.Go(func() error {
					return c.GetFile(repo, commit.ID, fmt.Sprintf("file%d", k), 0, 0, "", false, nil, w)
				})
			}
			require.NoError(b, eg.Wait())
		}
	}) {
		return
	}
	if !b.Run(fmt.Sprintf("PipelineCopy%dFiles", nFiles), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pipeline := uniqueString("BenchmarkPachydermPipeline")
			require.NoError(b, c.CreatePipeline(
				pipeline,
				"",
				[]string{"bash"},
				[]string{fmt.Sprintf("cp -R %s /pfs/out", path.Join("/pfs", repo, "/*"))},
				&ppsclient.ParallelismSpec{
					Strategy: ppsclient.ParallelismSpec_CONSTANT,
					Constant: 4,
				},
				[]*ppsclient.PipelineInput{{Repo: client.NewRepo(repo)}},
				false,
			))
			_, err := c.FlushCommit([]*pfsclient.Commit{client.NewCommit(repo, "master")}, nil)
			require.NoError(b, err)
			b.StopTimer()
			repoInfo, err := c.InspectRepo(repo)
			require.NoError(b, err)
			b.SetBytes(int64(repoInfo.SizeBytes))
			repoInfo, err = c.InspectRepo(pipeline)
			require.NoError(b, err)
			b.SetBytes(int64(repoInfo.SizeBytes))
		}
	}) {
		return
	}
}

func BenchmarkBigPFSWorkload(b *testing.B) {
	repo := uniqueString("BenchmarkBigPFSWorkload")
	c, err := client.NewInCluster()
	require.NoError(b, err)
	require.NoError(b, c.CreateRepo(repo))

	// We use a Zipf generator to generate file size, because it's been
	// found that most files in a big-data workload tend to be small.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	zipf := rand.NewZipf(r, 1.1, 1, 20)

	commit1, err := c.StartCommit(repo, "master")
	require.NoError(b, err)
	numTopLevelFiles := 1000
	if !b.Run(fmt.Sprintf("Put%dTopLevelFiles", numTopLevelFiles), func(b *testing.B) {
		var eg errgroup.Group
		var totalMB uint64
		for k := 0; k < numTopLevelFiles; k++ {
			k := k
			fileSizeMB := zipf.Uint64() + 1
			totalMB += fileSizeMB
			eg.Go(func() error {
				rand := rand.New(rand.NewSource(int64(time.Now().UnixNano())))
				_, err := c.PutFile(repo, commit1.ID, fmt.Sprintf("file%d", k), workload.NewReader(rand, int(fileSizeMB*MB)))
				return err
			})
		}
		b.SetBytes(int64(totalMB * MB))
		require.NoError(b, eg.Wait())
	}) {
		return
	}
	require.NoError(b, c.FinishCommit(repo, commit1.ID))

	commit2, err := c.StartCommit(repo, commit1.ID)
	require.NoError(b, err)
	numDirs := 10
	numDeepFiles := 1000
	if !b.Run(fmt.Sprintf("Put%dDeepFiles", numDeepFiles), func(b *testing.B) {
		var eg errgroup.Group
		var totalMB uint64
		for k := 0; k < numDeepFiles; k++ {
			k := k
			fileSizeMB := zipf.Uint64() + 1
			totalMB += fileSizeMB
			eg.Go(func() error {
				rand := rand.New(rand.NewSource(int64(time.Now().UnixNano())))
				_, err := c.PutFile(repo, commit2.ID, fmt.Sprintf("dir%d/file%d", k%numDirs, k), workload.NewReader(rand, int(fileSizeMB*MB)))
				return err
			})
		}
		b.SetBytes(int64(totalMB * MB))
		require.NoError(b, eg.Wait())
	}) {
		return
	}
	require.NoError(b, c.FinishCommit(repo, commit2.ID))

	if !b.Run(fmt.Sprintf("Get%dTopLevelFiles", numTopLevelFiles), func(b *testing.B) {
		var eg errgroup.Group
		w := &CountWriter{}
		defer func() { b.SetBytes(w.count) }()
		for k := 0; k < numTopLevelFiles; k++ {
			k := k
			eg.Go(func() error {
				return c.GetFile(repo, commit1.ID, fmt.Sprintf("file%d", k), 0, 0, "", false, nil, w)
			})
		}
		require.NoError(b, eg.Wait())
	}) {
		return
	}

	if !b.Run(fmt.Sprintf("PipelineCopy%dFiles", numTopLevelFiles+numDeepFiles), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pipeline := uniqueString("BenchmarkPachydermPipeline")
			require.NoError(b, c.CreatePipeline(
				pipeline,
				"",
				[]string{"bash"},
				[]string{fmt.Sprintf("cp -R %s /pfs/out", path.Join("/pfs", repo, "/*"))},
				&ppsclient.ParallelismSpec{
					Strategy: ppsclient.ParallelismSpec_CONSTANT,
					Constant: 4,
				},
				[]*ppsclient.PipelineInput{{Repo: client.NewRepo(repo)}},
				false,
			))
			_, err := c.FlushCommit([]*pfsclient.Commit{client.NewCommit(repo, "master")}, nil)
			require.NoError(b, err)
			b.StopTimer()
			repoInfo, err := c.InspectRepo(repo)
			require.NoError(b, err)
			b.SetBytes(int64(repoInfo.SizeBytes))
			repoInfo, err = c.InspectRepo(pipeline)
			require.NoError(b, err)
			b.SetBytes(int64(repoInfo.SizeBytes))
		}
	}) {
		return
	}
}

func BenchmarkListFile(b *testing.B) {
	repo := uniqueString("BenchmarkListFile")
	c, err := client.NewInCluster()
	require.NoError(b, err)
	require.NoError(b, c.CreateRepo(repo))
	nCommits := 250
	nFilesPerCommit := 500
	var commits []*pfsclient.Commit

	for i := 0; i < nCommits; i++ {
		commit, err := c.StartCommit(repo, "master")
		require.NoError(b, err)
		commits = append(commits, commit)
		var eg errgroup.Group
		for j := 0; j < nFilesPerCommit; j++ {
			j := j
			eg.Go(func() error {
				rand := rand.New(rand.NewSource(int64(time.Now().UnixNano())))
				_, err := c.PutFile(repo, "master", fmt.Sprintf("file%d-%d", i, j), workload.NewReader(rand, 1))
				return err
			})
		}
		require.NoError(b, eg.Wait())
		require.NoError(b, c.FinishCommit(repo, "master"))
	}

	for i, commit := range commits {
		b.Run(fmt.Sprintf("ListFileFast%d", i), func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				_, err := c.ListFileFast(commit.Repo.Name, commit.ID, "", "", false, nil)
				require.NoError(b, err)
			}
		})
	}

	for i, commit := range commits {
		b.Run(fmt.Sprintf("ListFile%d", i), func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				_, err := c.ListFile(commit.Repo.Name, commit.ID, "", "", false, nil, false)
				require.NoError(b, err)
			}
		})
	}
}

package cmd

import (
	"os"
	"path"

	"github.com/open-ch/go-libs/gitshell"
)

func (s *e2eTestSuite) TestBazelSingleFileChange() {
	// Append to the Go lib Hello World example Kaeter Module
	modifiedFile := "blueprints/go-helloworld/hellolib/lib.go"
	modifiedFileAbs := path.Join(s.repoRoot, modifiedFile)

	f, err := os.OpenFile(modifiedFileAbs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	s.NoError(err)
	if _, err := f.Write([]byte("\n")); err != nil {
		s.NoError(err)
	}
	if err := f.Close(); err != nil {
		s.NoError(err)
	}
	_, err = gitshell.GitAdd(s.repoRoot, modifiedFile)
	s.NoError(err)
	_, err = gitshell.GitCommit(s.repoRoot, "WIP")
	s.NoError(err)
	newCommit, err := gitshell.GitResolveRevision(s.repoRoot, "HEAD")
	s.NoError(err)

	// Execute the query
	info, err := executeKaeterCI(s.kaeterPath, s.repoRoot, s.baseCommit, newCommit)
	s.NoError(err)

	// Verify the result
	// Files
	s.Contains(info.Files.Modified, "blueprints/go-helloworld/hellolib/lib.go")
	s.Equal(1, len(info.Files.Modified))

	// Bazel
	s.Contains(info.Bazel.SourceFiles, "blueprints/go-helloworld/hellolib/lib.go")
	s.Equal(1, len(info.Bazel.SourceFiles))
	s.Contains(info.Bazel.Targets, "//blueprints/go-helloworld:container")
	s.Contains(info.Bazel.Targets, "//blueprints/go-helloworld:hellolib")
	s.Contains(info.Bazel.Targets, "//blueprints/go-helloworld:helloworld_test")
	s.Contains(info.Bazel.Targets, "//blueprints/go-helloworld:main")
	s.Contains(info.Bazel.Targets, "//blueprints/go-helloworld:publish")
	s.Equal(5, len(info.Bazel.Targets))
	s.Contains(info.Bazel.Packages, "//blueprints/go-helloworld")
	s.Equal(1, len(info.Bazel.Packages))

	// Kaeter
	s.Contains(info.Kaeter.Modules, "ch.osag.blueprints:go-helloworld")
	s.Equal(1, len(info.Kaeter.Modules))
}

func (s *e2eTestSuite) TestBazelSingleFileChangeJava() {
	// Append to the Go lib Hello World example Kaeter Module
	modifiedFile := "lake/kafka-connect/kafka-connect-azure-sentinel/src/main/java/ch/open/lake/kafka/connect/azure/sentinel/AzureSentinelWriter.java"
	modifiedFileAbs := path.Join(s.repoRoot, modifiedFile)

	f, err := os.OpenFile(modifiedFileAbs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	s.NoError(err)
	if _, err := f.Write([]byte("\n")); err != nil {
		s.NoError(err)
	}
	if err := f.Close(); err != nil {
		s.NoError(err)
	}
	_, err = gitshell.GitAdd(s.repoRoot, modifiedFile)
	s.NoError(err)
	_, err = gitshell.GitCommit(s.repoRoot, "WIP")
	s.NoError(err)
	newCommit, err := gitshell.GitResolveRevision(s.repoRoot, "HEAD")
	s.NoError(err)

	// Execute the query
	info, err := executeKaeterCI(s.kaeterPath, s.repoRoot, s.baseCommit, newCommit)
	s.NoError(err)

	// Verify the result
	// Files
	s.Contains(info.Files.Modified, "lake/kafka-connect/kafka-connect-azure-sentinel/src/main/java/ch/open/lake/kafka/connect/azure/sentinel/AzureSentinelWriter.java")
	s.Equal(1, len(info.Files.Modified))

	// Kaeter
	s.Contains(info.Kaeter.Modules, "ch.osag.lake:docker-kafka-connect")
	s.Contains(info.Kaeter.Modules, "ch.osag.lake:strimzi-kafka-connect")
	s.Equal(2, len(info.Kaeter.Modules))
}

func (s *e2eTestSuite) TestBazelTouchBazelSourceAndWorkspace() {
	// Append to the Go lib Hello World example Kaeter Module
	sourceFile := "3rdparty/yq_workspace.bzl"
	sourceFileAbs := path.Join(s.repoRoot, sourceFile)
	f, err := os.OpenFile(sourceFileAbs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	s.NoError(err)
	if _, err := f.Write([]byte("\n")); err != nil {
		s.NoError(err)
	}
	if err := f.Close(); err != nil {
		s.NoError(err)
	}
	_, err = gitshell.GitAdd(s.repoRoot, sourceFile)
	s.NoError(err)

	workspaceFile := "WORKSPACE"
	workspaceFileAbs := path.Join(s.repoRoot, workspaceFile)
	f, err = os.OpenFile(workspaceFileAbs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	s.NoError(err)
	if _, err := f.Write([]byte("\n")); err != nil {
		s.NoError(err)
	}
	if err := f.Close(); err != nil {
		s.NoError(err)
	}
	_, err = gitshell.GitAdd(s.repoRoot, workspaceFile)
	s.NoError(err)
	_, err = gitshell.GitCommit(s.repoRoot, "WIP")
	s.NoError(err)
	newCommit, err := gitshell.GitResolveRevision(s.repoRoot, "HEAD")
	s.NoError(err)

	// Execute the query
	info, err := executeKaeterCI(s.kaeterPath, s.repoRoot, s.baseCommit, newCommit)
	s.NoError(err)

	// Verify the result
	// Files
	s.Contains(info.Files.Modified, workspaceFile)
	s.Contains(info.Files.Modified, sourceFile)
	s.Equal(2, len(info.Files.Modified))

	// Bazel
	s.True(info.Bazel.Workspace)
	s.Contains(info.Bazel.BazelSources, sourceFile)
	s.Equal(1, len(info.Bazel.BazelSources))
}

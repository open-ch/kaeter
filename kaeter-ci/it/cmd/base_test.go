package cmd

func (s *e2eTestSuite) TestEmptyChangeSet() {
	// Execute the query
	info, err := executeKaeterCI(s.kaeterPath, s.repoRoot, s.baseCommit, s.baseCommit)
	s.NoError(err)

	// Verify the result
	s.Equal(0, len(info.Files.Added))
	s.Equal(0, len(info.Files.Modified))
	s.Equal(0, len(info.Files.Removed))

	s.Equal(0, len(info.Bazel.BazelSources))
	s.Equal(0, len(info.Bazel.Packages))
	s.Equal(0, len(info.Bazel.Targets))
	s.Equal(0, len(info.Bazel.SourceFiles))
	s.Equal(0, len(info.Bazel.Packages))
	s.Equal(false, info.Bazel.Workspace)

	s.Equal(0, len(info.Kaeter.Modules))

	s.Equal(0, len(info.Helm.Charts))
}

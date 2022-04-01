package cmd

func (s *e2eTestSuite) TestCommitTagExtraction() {

	// Execute the query
	info, err := executeKaeterCI(s.kaeterPath, s.repoRoot, "d83dcff486e7e9fb41a7b7e29778c92586a61a36", "20d4df739304aac0229328b15bc7da7f4176360b")
	s.NoError(err)

	// Verify the result
	s.Equal([]string{"tickets"}, info.Commit.Tags)

	// Execute the query
	info, err = executeKaeterCI(s.kaeterPath, s.repoRoot, "d0e7c6f021a0313f29d8710b80fa1437c8e68ede", "798e3d8fd0ef37e61bafca955860bc1c815a66bd")
	s.NoError(err)

	// Verify the result
	s.Equal([]string{"kaeter", "release"}, info.Commit.Tags)

}

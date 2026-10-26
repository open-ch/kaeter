package modules

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-ch/kaeter/mocks"
)

// dxDependencyIgnore are the patterns the DX UI release passes, those of
// applications/customer-portal/tools/update-changelog.sh.
var dxDependencyIgnore = []string{"**/versions.yaml", "**/kubernetes/**", "**/*.md"}

type mockCommit struct {
	subject string
	files   []string
}

func TestGetReleasableCommitsFiltering(t *testing.T) {
	tests := []struct {
		name             string
		modulePath       string // defaults to testModule
		dependencies     []string
		dependencyIgnore []string
		ignorePattern    string
		commits          []mockCommit
		expectedSubjects []string
	}{
		{
			name:             "Expect a source change to count",
			commits:          []mockCommit{{"code", []string{"testModule/main.go"}}},
			expectedSubjects: []string{"code"},
		},
		{
			name:    "Expect a versions.yaml only change to be ignored",
			commits: []mockCommit{{"release", []string{"testModule/versions.yaml"}}},
		},
		{
			name:    "Expect a versions.yml only change to be ignored",
			commits: []mockCommit{{"release", []string{"testModule/versions.yml"}}},
		},
		{
			name:    "Expect a CHANGELOG.md only change to be ignored",
			commits: []mockCommit{{"changelog", []string{"testModule/CHANGELOG.md"}}},
		},
		{
			name:    "Expect a kubernetes only change to be ignored",
			commits: []mockCommit{{"bump", []string{"testModule/kubernetes/test/app/values.yaml"}}},
		},
		{
			name:    "Expect a kubernetes only change with a name git would quote to be ignored",
			commits: []mockCommit{{"bump", []string{"testModule/kubernetes/v\u00e4lues \"prod\".yaml"}}},
		},
		{
			name: "Expect a full release commit to be ignored",
			commits: []mockCommit{{"release with bump", []string{
				"testModule/versions.yaml",
				"testModule/CHANGELOG.md",
				"testModule/kubernetes/test/app/values.yaml",
				"testModule/kubernetes/prod/app/values.yaml",
			}}},
		},
		{
			name: "Expect a release commit with a source change to count",
			commits: []mockCommit{{"mixed", []string{
				"testModule/versions.yaml",
				"testModule/CHANGELOG.md",
				"testModule/main.go",
			}}},
			expectedSubjects: []string{"mixed"},
		},
		{
			name: "Expect only files under the module path to decide",
			commits: []mockCommit{{"release plus other module", []string{
				"testModule/versions.yaml",
				"otherModule/main.go",
			}}},
		},
		{
			name: "Expect files that only look like release files to count",
			commits: []mockCommit{
				{"kubernetes file", []string{"testModule/kubernetes.yaml"}},
				{"nested versions", []string{"testModule/sub/versions.yaml"}},
				{"nested kubernetes", []string{"testModule/sub/kubernetes/values.yaml"}},
				{"changelog backup", []string{"testModule/CHANGELOG.md.bak"}},
			},
			expectedSubjects: []string{"kubernetes file", "nested versions", "nested kubernetes", "changelog backup"},
		},
		{
			name:       "Expect release files of a module at the repository root to be ignored",
			modulePath: ".",
			commits: []mockCommit{
				{"release", []string{"versions.yaml", "CHANGELOG.md", "kubernetes/test/values.yaml"}},
				{"nested release", []string{"testModule/versions.yaml"}},
				{"code", []string{"main.go"}},
			},
			expectedSubjects: []string{"nested release", "code"},
		},
		{
			name:          "Expect the ignore pattern to apply to the subject",
			ignorePattern: "^chore\\(autorelease\\)",
			commits: []mockCommit{
				{"chore(autorelease): update versions", []string{"testModule/main.go"}},
				{"fix: chore(autorelease) is mentioned", []string{"testModule/main.go"}},
			},
			expectedSubjects: []string{"fix: chore(autorelease) is mentioned"},
		},
		{
			name: "Expect commits outside the module to be skipped and the rest listed oldest first",
			commits: []mockCommit{
				{"first", []string{"testModule/a.go"}},
				{"outside", []string{"otherModule/a.go"}},
				{"second", []string{"testModule/b.go"}},
				{"third", []string{"testModule/c.go"}},
			},
			expectedSubjects: []string{"first", "second", "third"},
		},
		{
			name:         "Expect commits to the dependencies to count, oldest first",
			dependencies: []string{"3rdparty/new_ts", "/libs/lib/", "tsconfig.json"},
			commits: []mockCommit{
				{"bump vite", []string{"3rdparty/new_ts/package.json", "3rdparty/new_ts/pnpm-lock.yaml"}},
				{"outside", []string{"otherModule/a.go"}},
				{"lib", []string{"libs/lib/src/a.ts"}},
				{"tsconfig", []string{"tsconfig.json"}},
				{"module", []string{"testModule/main.go"}},
			},
			expectedSubjects: []string{"bump vite", "lib", "tsconfig", "module"},
		},
		{
			name:         "Expect every change of the dependencies to count by default",
			dependencies: []string{"3rdparty/new_ts", "libs/lib"},
			commits: []mockCommit{
				{"lib release", []string{"libs/lib/versions.yaml", "libs/lib/CHANGELOG.md"}},
				{"kubernetes", []string{"libs/lib/kubernetes/values.yaml"}},
				{"docs", []string{"3rdparty/new_ts/README.md", "libs/lib/docs/usage.md"}},
			},
			expectedSubjects: []string{"lib release", "kubernetes", "docs"},
		},
		{
			name:             "Expect changes of the dependencies limited to files matching DependencyIgnore to be skipped",
			dependencies:     []string{"3rdparty/new_ts", "libs/lib"},
			dependencyIgnore: dxDependencyIgnore,
			commits: []mockCommit{
				{"lib release", []string{"libs/lib/versions.yaml", "libs/lib/CHANGELOG.md"}},
				{"nested release", []string{"libs/lib/sub/versions.yaml", "libs/lib/sub/kubernetes/test/values.yaml"}},
				{"kubernetes", []string{"libs/lib/kubernetes/values.yaml"}},
				{"docs", []string{"3rdparty/new_ts/README.md", "libs/lib/docs/usage.md"}},
				{"docs and code", []string{"libs/lib/README.md", "libs/lib/src/a.ts"}},
			},
			expectedSubjects: []string{"docs and code"},
		},
		{
			name:             "Expect DependencyIgnore patterns to match below each dependency only",
			dependencies:     []string{"platforms/kubernetes/lib", "libs/[lib]"},
			dependencyIgnore: dxDependencyIgnore,
			commits: []mockCommit{
				{"lib under a kubernetes directory", []string{"platforms/kubernetes/lib/main.go"}},
				{"lib with glob characters", []string{"libs/[lib]/a.ts"}},
				{"docs of lib with glob characters", []string{"libs/[lib]/README.md"}},
			},
			expectedSubjects: []string{"lib under a kubernetes directory", "lib with glob characters"},
		},
		{
			name:         "Expect a dependency change to count with the module's release files",
			dependencies: []string{"3rdparty/new_ts"},
			commits: []mockCommit{{"release and bump", []string{
				"testModule/versions.yaml",
				"testModule/CHANGELOG.md",
				"3rdparty/new_ts/package.json",
			}}},
			expectedSubjects: []string{"release and bump"},
		},
		{
			name:             "Expect files that only look like ignored files of a dependency to count",
			dependencies:     []string{"libs/lib"},
			dependencyIgnore: dxDependencyIgnore,
			commits: []mockCommit{
				{"kubernetes file", []string{"libs/lib/kubernetes.yaml"}},
				{"markdown backup", []string{"libs/lib/README.md.bak"}},
			},
			expectedSubjects: []string{"kubernetes file", "markdown backup"},
		},
		{
			name:             "Expect DependencyIgnore not to apply to the module",
			dependencies:     []string{"3rdparty/new_ts"},
			dependencyIgnore: dxDependencyIgnore,
			commits:          []mockCommit{{"module docs", []string{"testModule/README.md"}}},
			expectedSubjects: []string{"module docs"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, releaseHash := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: mocks.EmptyVersionsYAML,
			})
			for _, commit := range tc.commits {
				commitFiles(t, repo, commit.subject, commit.files...)
			}
			modulePath := tc.modulePath
			if modulePath == "" {
				modulePath = "testModule"
			}

			commits, err := GetReleasableCommits(&ReleasableCommitsQuery{
				RepoRoot:         repo,
				ModulePath:       modulePath,
				Dependencies:     tc.dependencies,
				DependencyIgnore: tc.dependencyIgnore,
				FromRef:          releaseHash,
				ToRef:            "HEAD",
				IgnorePattern:    tc.ignorePattern,
			})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedSubjects, subjectsOf(commits))
		})
	}
}

func TestGetReleasableDependencyCommits(t *testing.T) {
	tests := []struct {
		name             string
		dependencies     []string
		dependencyIgnore []string
		ignorePattern    string
		commits          []mockCommit
		expectedSubjects []string
	}{
		{
			name:         "Expect only commits to the dependencies to count, oldest first",
			dependencies: []string{"3rdparty/new_ts", "/libs/lib/", "tsconfig.json"},
			commits: []mockCommit{
				{"bump vite", []string{"3rdparty/new_ts/package.json"}},
				{"module", []string{"testModule/main.go"}},
				{"lib", []string{"libs/lib/src/a.ts"}},
				{"outside", []string{"otherModule/a.go"}},
				{"tsconfig", []string{"tsconfig.json"}},
				{"module and lib", []string{"testModule/main.go", "libs/lib/src/b.ts"}},
			},
			expectedSubjects: []string{"bump vite", "lib", "tsconfig", "module and lib"},
		},
		{
			name:             "Expect Markdown changes of the dependencies to count by default",
			dependencies:     []string{"libs/lib"},
			commits:          []mockCommit{{"docs", []string{"libs/lib/docs/usage.md"}}},
			expectedSubjects: []string{"docs"},
		},
		{
			name:             "Expect changes of the dependencies limited to files matching DependencyIgnore to be skipped",
			dependencies:     []string{"libs/lib"},
			dependencyIgnore: dxDependencyIgnore,
			commits: []mockCommit{
				{"lib release", []string{"libs/lib/versions.yaml", "libs/lib/CHANGELOG.md"}},
				{"kubernetes", []string{"libs/lib/kubernetes/values.yaml"}},
				{"docs", []string{"libs/lib/docs/usage.md"}},
				{"docs and module code", []string{"libs/lib/README.md", "testModule/main.go"}},
			},
		},
		{
			name:          "Expect the ignore pattern to apply",
			dependencies:  []string{"libs/lib"},
			ignorePattern: "^chore\\(autorelease\\)",
			commits: []mockCommit{
				{"chore(autorelease): update versions", []string{"libs/lib/a.ts"}},
				{"feat: lib", []string{"libs/lib/b.ts"}},
			},
			expectedSubjects: []string{"feat: lib"},
		},
		{
			name:    "Expect no commits without dependencies",
			commits: []mockCommit{{"module", []string{"testModule/main.go"}}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, releaseHash := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: mocks.EmptyVersionsYAML,
			})
			for _, commit := range tc.commits {
				commitFiles(t, repo, commit.subject, commit.files...)
			}

			commits, err := GetReleasableDependencyCommits(&ReleasableCommitsQuery{
				RepoRoot:         repo,
				ModulePath:       "testModule",
				Dependencies:     tc.dependencies,
				DependencyIgnore: tc.dependencyIgnore,
				FromRef:          releaseHash,
				ToRef:            "HEAD",
				IgnorePattern:    tc.ignorePattern,
			})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedSubjects, subjectsOf(commits))
		})
	}
}

func TestGetReleasableCommitsDetails(t *testing.T) {
	repo, releaseHash := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
		Path:         "testModule",
		VersionsYAML: mocks.EmptyVersionsYAML,
	})
	// The subject contains the separators of the git log format.
	firstHash := commitFiles(t, repo, "first \x1e change \x1f", "testModule/a.go")
	runGit(t, repo, "commit", "--allow-empty", "-m", "unrelated")
	secondHash := commitFiles(t, repo, "second change", "testModule/b.go")
	commitFiles(t, repo, "after end ref", "testModule/c.go")

	commits, err := GetReleasableCommits(&ReleasableCommitsQuery{
		RepoRoot:   repo,
		ModulePath: filepath.Join(repo, "testModule"), // absolute paths are accepted too
		FromRef:    releaseHash,
		ToRef:      secondHash,
	})

	require.NoError(t, err)
	assert.Equal(t, []ReleasableCommit{
		{SHA: firstHash, Subject: "first \x1e change \x1f", AuthorEmail: "unittest@example.ch"},
		{SHA: secondHash, Subject: "second change", AuthorEmail: "unittest@example.ch"},
	}, commits)
}

func TestGetReleasableCommitsExcludesMerges(t *testing.T) {
	repo, releaseHash := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
		Path:         "testModule",
		VersionsYAML: mocks.EmptyVersionsYAML,
	})
	mocks.SwitchToNewBranch(t, repo, "feature")
	commitFiles(t, repo, "on branch", "testModule/a.go")
	runGit(t, repo, "switch", "main")
	commitFiles(t, repo, "on main", "testModule/b.go")
	// Both sides touch the module, so git log would list the merge without --no-merges.
	runGit(t, repo, "merge", "--no-ff", "-m", "merge feature", "feature")

	commits, err := GetReleasableCommits(&ReleasableCommitsQuery{
		RepoRoot:   repo,
		ModulePath: "testModule",
		FromRef:    releaseHash,
		ToRef:      "HEAD",
	})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"on branch", "on main"}, subjectsOf(commits))
}

func TestGetReleasableCommitsCountsMovesIntoKubernetes(t *testing.T) {
	repo, _ := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
		Path:         "testModule",
		VersionsYAML: mocks.EmptyVersionsYAML,
	})
	releaseHash := commitFiles(t, repo, "add config", "testModule/config/values.yaml")
	runGit(t, repo, "mv", "testModule/config", "testModule/kubernetes")
	runGit(t, repo, "commit", "-m", "move config")

	commits, err := GetReleasableCommits(&ReleasableCommitsQuery{
		RepoRoot:   repo,
		ModulePath: "testModule",
		FromRef:    releaseHash,
		ToRef:      "HEAD",
	})

	require.NoError(t, err)
	// With rename detection only the new path would be listed, which looks release-only.
	assert.Equal(t, []string{"move config"}, subjectsOf(commits))
}

func TestGetReleasableCommitsRefs(t *testing.T) {
	const release = "<release hash>" // replaced by the hash of the release commit
	tests := []struct {
		name          string
		fromRef       string
		toRef         string
		ignorePattern string
		expectedErr   error
		errorContains string
	}{
		{name: "Expect INIT to have no commits", fromRef: InitRef, toRef: "HEAD"},
		{name: "Expect a pending AUTORELEASE to fail", fromRef: autoReleaseRef, toRef: "HEAD", expectedErr: ErrAutoreleasePending},
		// git would accept these refs, so only kaeter's own checks make them fail.
		{name: "Expect a short hash to fail", fromRef: "short", toRef: "HEAD", errorContains: "invalid previous release ref"},
		{name: "Expect an empty end ref to fail", fromRef: release, errorContains: "missing end ref"},
		{name: "Expect an unknown hash to fail", fromRef: strings.Repeat("a", hashLength), toRef: "HEAD", errorContains: "failed to compute git log"},
		{name: "Expect an invalid ignore pattern to fail", fromRef: release, toRef: "HEAD", ignorePattern: `\K`, errorContains: "invalid needsrelease ignore pattern"},
		{name: "Expect INIT to ignore an invalid ignore pattern", fromRef: InitRef, toRef: "HEAD", ignorePattern: `\K`},
	}

	repo, releaseHash := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
		Path:         "testModule",
		VersionsYAML: mocks.EmptyVersionsYAML,
	})
	commitFiles(t, repo, "code", "testModule/main.go")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fromRef := tc.fromRef
			switch fromRef {
			case release:
				fromRef = releaseHash
			case "short":
				fromRef = releaseHash[:12]
			default:
			}

			commits, err := GetReleasableCommits(&ReleasableCommitsQuery{
				RepoRoot:      repo,
				ModulePath:    "testModule",
				FromRef:       fromRef,
				ToRef:         tc.toRef,
				IgnorePattern: tc.ignorePattern,
			})

			switch {
			case tc.expectedErr != nil:
				assert.ErrorIs(t, err, tc.expectedErr)
			case tc.errorContains != "":
				assert.ErrorContains(t, err, tc.errorContains)
			default:
				assert.NoError(t, err)
				assert.Empty(t, commits)
			}
		})
	}
}

func TestGetReleasableCommitsRejectsInvalidDependencyIgnore(t *testing.T) {
	repo, releaseHash := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
		Path:         "testModule",
		VersionsYAML: mocks.EmptyVersionsYAML,
	})
	for _, pattern := range []string{"", "/abs/*.md", "../*.md", "docs/../../*.md"} {
		t.Run(pattern, func(t *testing.T) {
			for _, get := range []func(*ReleasableCommitsQuery) ([]ReleasableCommit, error){GetReleasableCommits, GetReleasableDependencyCommits} {
				_, err := get(&ReleasableCommitsQuery{
					RepoRoot:         repo,
					ModulePath:       "testModule",
					Dependencies:     []string{"libs/lib"},
					DependencyIgnore: []string{"**/*.md", pattern},
					FromRef:          releaseHash,
					ToRef:            "HEAD",
				})

				assert.ErrorContains(t, err, "invalid dependency ignore pattern")
			}
		})
	}
}

func TestGetReleasableCommitsRejectsPathOutsideRepo(t *testing.T) {
	repo, releaseHash := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
		Path:         "testModule",
		VersionsYAML: mocks.EmptyVersionsYAML,
	})

	_, err := GetReleasableCommits(&ReleasableCommitsQuery{
		RepoRoot:   repo,
		ModulePath: "../elsewhere",
		FromRef:    releaseHash,
		ToRef:      "HEAD",
	})

	assert.ErrorContains(t, err, "outside of repository")
}

// commitFiles writes the files, relative to repo, and commits them. It returns the commit hash.
func commitFiles(t *testing.T, repo, subject string, files ...string) string {
	t.Helper()
	for _, file := range files {
		absPath := filepath.Join(repo, file)
		require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0755))
		// Unique content so every commit changes each file
		require.NoError(t, os.WriteFile(absPath, []byte(subject), 0600))
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", subject)
	return runGit(t, repo, "rev-parse", "HEAD")
}

func runGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	return strings.TrimSpace(string(output))
}

func subjectsOf(commits []ReleasableCommit) []string {
	var subjects []string
	for _, commit := range commits {
		subjects = append(subjects, commit.Subject)
	}
	return subjects
}

package modules

import (
	"os"
	"github.com/open-ch/kaeter/mocks"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	var tests = []struct {
		name                 string
		config               InitializationConfig // Note ModulePath will be prepended with a tmp folder.
		createEmptyFilePaths []string             // Create empty files at these paths (will create dirs if needed)
		mkDirPaths           []string             // Create empty folders at these paths
		hasError             bool
	}{
		{
			name: "invalid version scheme is rejected",
			config: InitializationConfig{
				VersioningScheme: "NationalParks",
			},
			hasError: true,
		},
		{
			name: "Creates folder if it doesn't exist yet",
			config: InitializationConfig{
				VersioningScheme: "SemVer",
				ModulePath:       "NotAFolder",
			},
		},
		{
			name: "fails if versions.yaml file already exists",
			config: InitializationConfig{
				VersioningScheme: "SemVer",
				ModulePath:       "awesomeMod",
			},
			createEmptyFilePaths: []string{"awesomeMod/versions.yaml"},
			hasError:             true,
		},
		{
			name: "creates a basic module in empty folder",
			config: InitializationConfig{
				VersioningScheme: "SemVer",
				ModulePath:       "awesomeMod",
			},
			mkDirPaths: []string{"awesomeMod"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateMockRepo(t)
			defer os.RemoveAll(testFolder)
			t.Logf("Temp test folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)", testFolder)
			tc.config.ModulePath = path.Join(testFolder, tc.config.ModulePath)
			for _, filePath := range tc.createEmptyFilePaths {
				inRepoPath := path.Join(testFolder, filePath)
				err := os.MkdirAll(path.Dir(inRepoPath), 0755)
				assert.NoError(t, err)
				err = os.WriteFile(inRepoPath, []byte(""), 0644)
				assert.NoError(t, err)
			}
			for _, dirs := range tc.mkDirPaths {
				err := os.MkdirAll(path.Join(testFolder, dirs), 0755)
				assert.NoError(t, err)
			}

			_, err := Initialize(tc.config)

			if tc.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.FileExists(t, path.Join(tc.config.ModulePath, "versions.yaml"))
		})
	}
}

func TestValidateVersioningScheme(t *testing.T) {
	var tests = []struct {
		name       string
		requested  string
		expected   string
		isNotValid bool
	}{
		{
			name:      "SemVer is valid",
			requested: "SemVer",
			expected:  "SemVer",
		},
		{
			name:      "CalVer is valid",
			requested: "CalVer",
			expected:  "CalVer",
		},
		{
			name:      "AnyStringVer is valid",
			requested: "AnyStringVer",
			expected:  "AnyStringVer",
		},
		{
			name:       "LunarPhase is unfortunately not supported (yet)",
			requested:  "LunarPhase",
			isNotValid: true,
		},
		{
			name:      "Uppercase input is normalized",
			requested: "ANYSTRINGVER",
			expected:  "AnyStringVer",
		},
		{
			name:      "Lowercase input in normalized",
			requested: "anystringver",
			expected:  "AnyStringVer",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			versioningScheme, err := validateVersioningScheme(tc.requested)

			if tc.isNotValid {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, versioningScheme)
		})
	}
}

func TestGetAbsoluteNewModulePath(t *testing.T) {
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	var tests = []struct {
		name             string
		modulePath       string
		hasError         bool
		useTempFolder    bool
		createTempFileAt string
	}{
		{
			name:       "Accepts existing directories with relative path",
			modulePath: ".",
		},
		{
			name:          "Fails if directory doesn't exist",
			modulePath:    "awesomemod",
			useTempFolder: true,
		},
		{
			name:          "Creates nested directory as needed",
			modulePath:    "awesome/mod",
			useTempFolder: true,
		},
		{
			name:             "Fails if path is not a directory",
			modulePath:       "afilenotadir.yaml",
			hasError:         true,
			useTempFolder:    true,
			createTempFileAt: "afilenotadir.yaml",
		},
		{
			name:             "Fails if the version file already exists",
			modulePath:       ".",
			hasError:         true,
			useTempFolder:    true,
			createTempFileAt: "versions.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			modulePath := tc.modulePath
			basePath := cwd
			if tc.useTempFolder {
				basePath = mocks.CreateTmpFolder(t)
				defer os.RemoveAll(basePath)
				t.Logf("Temp test folder: %s\n(disable `defer os.RemoveAll(basePath)` to keep for debugging)", basePath)
				modulePath = path.Join(basePath, modulePath)
				if tc.createTempFileAt != "" {
					// TODO are there mocks for that in mocks.?
					tmpFileAbsPath := path.Join(basePath, tc.createTempFileAt)
					t.Logf("Creating empty file at %s", tmpFileAbsPath)
					err := os.MkdirAll(path.Dir(tmpFileAbsPath), 0755)
					assert.NoError(t, err)
					err = os.WriteFile(tmpFileAbsPath, []byte(""), 0644)
					assert.NoError(t, err)
				}
			}
			absPath, err := validateModulePathAndCreateDir(modulePath)

			if tc.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, path.Join(basePath, tc.modulePath), absPath)
		})
	}
}

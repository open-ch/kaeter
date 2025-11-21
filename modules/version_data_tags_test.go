package modules

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalReleaseDataWithTags(t *testing.T) {
	tests := []struct {
		name          string
		releaseData   string
		wantTimestamp string
		wantCommitID  string
		wantTags      []string
		wantErr       bool
	}{
		{
			name:          "backward compatible - no tags",
			releaseData:   "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091",
			wantTimestamp: "2019-04-01T16:06:07Z",
			wantCommitID:  "675156f77a931aa40ceb115b763d9d1230b26091",
			wantTags:      nil,
			wantErr:       false,
		},
		{
			name:          "single tag",
			releaseData:   "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091|production",
			wantTimestamp: "2019-04-01T16:06:07Z",
			wantCommitID:  "675156f77a931aa40ceb115b763d9d1230b26091",
			wantTags:      []string{"production"},
			wantErr:       false,
		},
		{
			name:          "multiple tags",
			releaseData:   "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091|production,stable,lts",
			wantTimestamp: "2019-04-01T16:06:07Z",
			wantCommitID:  "675156f77a931aa40ceb115b763d9d1230b26091",
			wantTags:      []string{"production", "stable", "lts"},
			wantErr:       false,
		},
		{
			name:          "tags with whitespace",
			releaseData:   "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091| production , stable , lts ",
			wantTimestamp: "2019-04-01T16:06:07Z",
			wantCommitID:  "675156f77a931aa40ceb115b763d9d1230b26091",
			wantTags:      []string{"production", "stable", "lts"},
			wantErr:       false,
		},
		{
			name:          "empty tag field",
			releaseData:   "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091|",
			wantTimestamp: "2019-04-01T16:06:07Z",
			wantCommitID:  "675156f77a931aa40ceb115b763d9d1230b26091",
			wantTags:      nil,
			wantErr:       false,
		},
		{
			name:          "tags with empty entries",
			releaseData:   "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091|prod,,stable,,,lts",
			wantTimestamp: "2019-04-01T16:06:07Z",
			wantCommitID:  "675156f77a931aa40ceb115b763d9d1230b26091",
			wantTags:      []string{"prod", "stable", "lts"},
			wantErr:       false,
		},
		{
			name:        "invalid timestamp",
			releaseData: "invalid-timestamp|675156f77a931aa40ceb115b763d9d1230b26091|production",
			wantErr:     true,
		},
		{
			name:        "missing commit id",
			releaseData: "2019-04-01T16:06:07Z",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timestamp, commitID, tags, err := unmarshalReleaseData(tt.releaseData)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTimestamp, timestamp.Format(time.RFC3339))
			assert.Equal(t, tt.wantCommitID, commitID)
			assert.Equal(t, tt.wantTags, tags)
		})
	}
}

func TestMarshalReleaseDataWithTags(t *testing.T) {
	tests := []struct {
		name     string
		metadata VersionMetadata
		want     string
	}{
		{
			name: "no tags - backward compatible",
			metadata: VersionMetadata{
				Timestamp: time.Date(2019, 4, 1, 16, 6, 7, 0, time.UTC),
				CommitID:  "675156f77a931aa40ceb115b763d9d1230b26091",
				Tags:      nil,
			},
			want: "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091",
		},
		{
			name: "empty tags slice - backward compatible",
			metadata: VersionMetadata{
				Timestamp: time.Date(2019, 4, 1, 16, 6, 7, 0, time.UTC),
				CommitID:  "675156f77a931aa40ceb115b763d9d1230b26091",
				Tags:      []string{},
			},
			want: "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091",
		},
		{
			name: "single tag",
			metadata: VersionMetadata{
				Timestamp: time.Date(2019, 4, 1, 16, 6, 7, 0, time.UTC),
				CommitID:  "675156f77a931aa40ceb115b763d9d1230b26091",
				Tags:      []string{"production"},
			},
			want: "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091|production",
		},
		{
			name: "multiple tags",
			metadata: VersionMetadata{
				Timestamp: time.Date(2019, 4, 1, 16, 6, 7, 0, time.UTC),
				CommitID:  "675156f77a931aa40ceb115b763d9d1230b26091",
				Tags:      []string{"production", "stable", "lts"},
			},
			want: "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091|production,stable,lts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.marshalReleaseData()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersionMetadataRoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		releaseData string
	}{
		{
			name:        "backward compatible - no tags",
			releaseData: "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091",
		},
		{
			name:        "with single tag",
			releaseData: "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091|production",
		},
		{
			name:        "with multiple tags",
			releaseData: "2019-04-01T16:06:07Z|675156f77a931aa40ceb115b763d9d1230b26091|production,stable,lts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Unmarshal
			timestamp, commitID, tags, err := unmarshalReleaseData(tt.releaseData)
			require.NoError(t, err)

			// Create metadata
			metadata := VersionMetadata{
				Timestamp: *timestamp,
				CommitID:  commitID,
				Tags:      tags,
			}

			// Marshal back
			marshaled := metadata.marshalReleaseData()

			// Unmarshal again to compare (handles whitespace normalization)
			timestamp2, commitID2, tags2, err := unmarshalReleaseData(marshaled)
			require.NoError(t, err)

			assert.Equal(t, timestamp.Format(time.RFC3339), timestamp2.Format(time.RFC3339))
			assert.Equal(t, commitID, commitID2)
			assert.Equal(t, tags, tags2)
		})
	}
}

func TestUnmarshalVersionMetadataWithTags(t *testing.T) {
	tests := []struct {
		name             string
		versionStr       string
		releaseData      string
		versioningScheme string
		wantVersion      string
		wantTags         []string
		wantErr          bool
	}{
		{
			name:             "SemVer without tags",
			versionStr:       "1.2.3",
			releaseData:      "2019-04-01T16:06:07Z|abc123",
			versioningScheme: SemVer,
			wantVersion:      "1.2.3",
			wantTags:         nil,
			wantErr:          false,
		},
		{
			name:             "SemVer with tags",
			versionStr:       "1.2.3",
			releaseData:      "2019-04-01T16:06:07Z|abc123|production,stable",
			versioningScheme: SemVer,
			wantVersion:      "1.2.3",
			wantTags:         []string{"production", "stable"},
			wantErr:          false,
		},
		{
			name:             "AnyStringVer with tags",
			versionStr:       "custom-v1",
			releaseData:      "2019-04-01T16:06:07Z|abc123|beta,experimental",
			versioningScheme: AnyStringVer,
			wantVersion:      "custom-v1",
			wantTags:         []string{"beta", "experimental"},
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := UnmarshalVersionMetadata(tt.versionStr, tt.releaseData, tt.versioningScheme)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, metadata.Number.String())
			assert.Equal(t, tt.wantTags, metadata.Tags)
		})
	}
}

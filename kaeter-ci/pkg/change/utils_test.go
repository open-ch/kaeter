package change

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRemoveTrailingEmptyStrings(t *testing.T) {
	tests := []struct {
		inputArray  []string
		outputArray []string
	}{
		{
			inputArray:  []string{},
			outputArray: []string{},
		},
		{
			inputArray:  []string{"tag"},
			outputArray: []string{"tag"},
		},
		{
			inputArray:  []string{"tag1", "tag2", "tag3"},
			outputArray: []string{"tag1", "tag2", "tag3"},
		},
		{
			inputArray:  []string{"tag1", "", ""},
			outputArray: []string{"tag1"},
		},
	}

	for _, test := range tests {
		a := removeTrailingEmptyStrings(test.inputArray)
		assert.Equal(t, test.outputArray, a)
	}
}

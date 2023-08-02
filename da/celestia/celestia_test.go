package celestia

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rollkit/rollkit/da"
)

func TestDataRequestErrorToStatus(t *testing.T) {
	randErr := errors.New("some random error")
	var test = []struct {
		statusCode da.StatusCode
		err        error
	}{
		// Status Success Cases
		{da.StatusSuccess, nil},
		{da.StatusSuccess, ErrNamespaceNotFound},
		{da.StatusSuccess, errors.Join(randErr, ErrNamespaceNotFound, randErr)},

		// TODO: cases that need investigating, are these possible? If
		// so, is this the correct status code?
		{da.StatusSuccess, errors.Join(ErrEDSNotFound, ErrNamespaceNotFound)},
		{da.StatusSuccess, errors.Join(ErrDataNotFound, ErrNamespaceNotFound)},

		// Status not Found Cases
		{da.StatusNotFound, ErrDataNotFound},
		{da.StatusNotFound, ErrEDSNotFound},
		{da.StatusNotFound, errors.Join(ErrEDSNotFound, ErrDataNotFound)},
		{da.StatusNotFound, errors.Join(ErrEDSNotFound, randErr)},
		{da.StatusNotFound, errors.Join(randErr, ErrDataNotFound)},

		// Status Error Cases
		{da.StatusError, randErr},
	}
	assert := assert.New(t)
	for _, tt := range test {
		t.Logf("Testing %v", tt.err)
		assert.Equal(tt.statusCode, dataRequestErrorToStatus(tt.err))
	}
}

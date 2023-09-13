package sequencer

import (
	cmtypes "github.com/cometbft/cometbft/types"
	"github.com/rollkit/rollkit/types"
)

var _ Sequencer = MockSequencer{}

func NewMockSequencer(genesis *cmtypes.GenesisDoc) (Sequencer, error) {
	return MockSequencer{}, nil
}

type MockSequencer struct{}

func (s MockSequencer) VerifyProposer(last types.Header, next types.Header) error {
	// dummy implementation that allows anything
	return nil
}

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

func (s MockSequencer) CheckSafetyInvariant(newBlock *types.Block, oldBlocks []*types.Block) uint {
	return Ok
}

func (s MockSequencer) ApplyForkChoiceRule(blocks []*types.Block) (*types.Block, error) {
	return blocks[0], nil
}

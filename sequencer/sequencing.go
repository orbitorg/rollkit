package sequencer

import (
	"github.com/rollkit/rollkit/types"
)

const (
	Ok = iota
	Junk
	ConsensusFault
)

type Sequencer interface {
	CheckSafetyInvariant(newBlock *types.Block, oldBlocks []*types.Block) uint
	ApplyForkChoiceRule(blocks []*types.Block) (*types.Block, error)
}

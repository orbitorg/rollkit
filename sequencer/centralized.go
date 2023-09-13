package sequencer

import (
	"fmt"

	"bytes"

	cmtypes "github.com/cometbft/cometbft/types"
	"github.com/rollkit/rollkit/types"
)

var _ Sequencer = CentralizedSequencer{}

type CentralizedSequencer struct {
	SequencerAddress []byte
}

func NewCentralizedSequencer(genesis *cmtypes.GenesisDoc) (Sequencer, error) {
	if genesis == nil {
		return nil, fmt.Errorf("genesis can't be nil for centralized sequencer")
	}
	if len(genesis.Validators) != 1 {
		return nil, fmt.Errorf("number of validators in genesis != 1")
	}
	return CentralizedSequencer{
		SequencerAddress: genesis.Validators[0].Address.Bytes(),
	}, nil
}

func (s CentralizedSequencer) CheckSafetyInvariant(newBlock *types.Block, oldBlocks []*types.Block) uint {
	if !bytes.Equal(newBlock.SignedHeader.Header.ProposerAddress, s.SequencerAddress) {
		return Junk
	}
	if len(oldBlocks) > 0 {
		return ConsensusFault
	}
	return Ok
}

func (s CentralizedSequencer) ApplyForkChoiceRule(blocks []*types.Block) (*types.Block, error) {
	return nil, nil
}

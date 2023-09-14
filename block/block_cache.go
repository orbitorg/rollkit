package block

import (
	"sync"

	"github.com/rollkit/rollkit/types"
)

type BlockCache struct {
	blocks            map[uint64][]*types.Block
	hashes            map[string]bool
	hardConfirmations map[string]bool
	mtx               *sync.RWMutex
}

func NewBlockCache() *BlockCache {
	return &BlockCache{
		blocks:            make(map[uint64][]*types.Block),
		hashes:            make(map[string]bool),
		hardConfirmations: make(map[string]bool),
		mtx:               new(sync.RWMutex),
	}
}

func (bc *BlockCache) getFirstBlock(height uint64) (*types.Block, bool) {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	blocks, ok := bc.blocks[height]
	if !ok || len(blocks) == 0 {
		return nil, false
	}
	return blocks[0], true
}

func (bc *BlockCache) getBlocks(height uint64) ([]*types.Block, bool) {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	blocks, ok := bc.blocks[height]
	if !ok || len(blocks) == 0 {
		return []*types.Block{}, false
	}
	return blocks, true
}

func (bc *BlockCache) addBlock(height uint64, block *types.Block) {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	_, keyExists := bc.blocks[height]
	if !keyExists {
		bc.blocks[height] = []*types.Block{}
	}
	bc.blocks[height] = append(bc.blocks[height], block)
}

func (bc *BlockCache) deleteBlock(height uint64) {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	delete(bc.blocks, height)
}

func (bc *BlockCache) isSeen(hash string) bool {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	return bc.hashes[hash]
}

func (bc *BlockCache) setSeen(hash string) {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	bc.hashes[hash] = true
}

func (bc *BlockCache) setHardConfirmed(hash string) {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	bc.hardConfirmations[hash] = true
}

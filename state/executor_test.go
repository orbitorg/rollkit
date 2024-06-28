package state

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/pubsub/query"
	"github.com/cometbft/cometbft/proxy"
	cmtypes "github.com/cometbft/cometbft/types"

	"github.com/rollkit/rollkit/mempool"
	"github.com/rollkit/rollkit/test/mocks"
	"github.com/rollkit/rollkit/types"
)

func prepareProposalResponse(_ context.Context, req *abci.PrepareProposalRequest) (*abci.PrepareProposalResponse, error) {
	return &abci.PrepareProposalResponse{
		Txs: req.Txs,
	}, nil
}

func doTestCreateBlock(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	logger := log.TestingLogger()

	app := &mocks.Application{}
	app.On("CheckTx", mock.Anything, mock.Anything).Return(&abci.CheckTxResponse{}, nil)
	app.On("PrepareProposal", mock.Anything, mock.Anything).Return(prepareProposalResponse)
	app.On("ProcessProposal", mock.Anything, mock.Anything).Return(&abci.ProcessProposalResponse{Status: abci.PROCESS_PROPOSAL_STATUS_ACCEPT}, nil)
	fmt.Println("App On CheckTx")
	client, err := proxy.NewLocalClientCreator(app).NewABCIConsensusClient()
	fmt.Println("Created New Local Client")
	require.NoError(err)
	require.NotNil(client)

	fmt.Println("Made NID")
	mpool := mempool.NewCListMempool(cfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(client, proxy.NopMetrics()), 0)
	fmt.Println("Made a NewTxMempool")
	executor := NewBlockExecutor([]byte("test address"), "test", mpool, proxy.NewAppConnConsensus(client, proxy.NopMetrics()), nil, 100, logger, NopMetrics(), types.GetRandomBytes(32))
	fmt.Println("Made a New Block Executor")

	state := types.State{}

	state.ConsensusParams.Block = &cmproto.BlockParams{}
	state.ConsensusParams.Block.MaxBytes = 100
	state.ConsensusParams.Block.MaxGas = 100000

	// empty block
	block, err := executor.CreateBlock(1, &types.Commit{}, abci.ExtendedCommitInfo{}, []byte{}, state)
	require.NoError(err)
	require.NotNil(block)
	assert.Empty(block.Data.Txs)
	assert.Equal(uint64(1), block.Height())

	// one small Tx
	_, err = mpool.CheckTx([]byte{1, 2, 3, 4}, "")
	require.NoError(err)
	block, err = executor.CreateBlock(2, &types.Commit{}, abci.ExtendedCommitInfo{}, []byte{}, state)
	require.NoError(err)
	require.NotNil(block)
	assert.Equal(uint64(2), block.Height())
	assert.Len(block.Data.Txs, 1)

	// now there are 3 Txs, and only two can fit into single block
	_, err = mpool.CheckTx([]byte{4, 5, 6, 7}, "")
	require.NoError(err)
	_, err = mpool.CheckTx(make([]byte, 100), "")
	require.NoError(err)
	block, err = executor.CreateBlock(3, &types.Commit{}, abci.ExtendedCommitInfo{}, []byte{}, state)
	require.NoError(err)
	require.NotNil(block)
	assert.Len(block.Data.Txs, 2)

	// limit max bytes
	mpool.Flush()
	executor.maxBytes = 10
	_, err = mpool.CheckTx(make([]byte, 10), "")
	require.NoError(err)
	block, err = executor.CreateBlock(4, &types.Commit{}, abci.ExtendedCommitInfo{}, []byte{}, state)
	require.NoError(err)
	require.NotNil(block)
	assert.Empty(block.Data.Txs)
}

func TestCreateBlockWithFraudProofsDisabled(t *testing.T) {
	doTestCreateBlock(t)
}

func doTestApplyBlock(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	logger := log.TestingLogger()

	var mockAppHash []byte
	_, err := rand.Read(mockAppHash[:])
	require.NoError(err)

	app := &mocks.Application{}
	app.On("CheckTx", mock.Anything, mock.Anything).Return(&abci.CheckTxResponse{}, nil)
	app.On("Commit", mock.Anything, mock.Anything).Return(&abci.CommitResponse{}, nil)
	app.On("PrepareProposal", mock.Anything, mock.Anything).Return(prepareProposalResponse)
	app.On("ProcessProposal", mock.Anything, mock.Anything).Return(&abci.ProcessProposalResponse{Status: abci.PROCESS_PROPOSAL_STATUS_ACCEPT}, nil)
	app.On("FinalizeBlock", mock.Anything, mock.Anything).Return(
		func(_ context.Context, req *abci.FinalizeBlockRequest) (*abci.FinalizeBlockResponse, error) {
			txResults := make([]*abci.ExecTxResult, len(req.Txs))
			for idx := range req.Txs {
				txResults[idx] = &abci.ExecTxResult{
					Code: abci.CodeTypeOK,
				}
			}

			return &abci.FinalizeBlockResponse{
				TxResults: txResults,
				AppHash:   mockAppHash,
			}, nil
		},
	)

	client, err := proxy.NewLocalClientCreator(app).NewABCIConsensusClient()
	require.NoError(err)
	require.NotNil(client)

	mpool := mempool.NewCListMempool(cfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(client, proxy.NopMetrics()), 0)
	eventBus := cmtypes.NewEventBus()
	require.NoError(eventBus.Start())

	txQuery, err := query.New("tm.event='Tx'")
	require.NoError(err)
	txSub, err := eventBus.Subscribe(context.Background(), "test", txQuery, 1000)
	require.NoError(err)
	require.NotNil(txSub)

	headerQuery, err := query.New("tm.event='NewBlockHeader'")
	require.NoError(err)
	headerSub, err := eventBus.Subscribe(context.Background(), "test", headerQuery, 100)
	require.NoError(err)
	require.NotNil(headerSub)

	vKey := ed25519.GenPrivKey()
	validators := []*cmtypes.Validator{
		{
			Address:          vKey.PubKey().Address(),
			PubKey:           vKey.PubKey(),
			VotingPower:      int64(100),
			ProposerPriority: int64(1),
		},
	}
	state := types.State{}
	state.InitialHeight = 1
	state.LastBlockHeight = 0
	state.ConsensusParams.Block = &cmproto.BlockParams{}
	state.ConsensusParams.Block.MaxBytes = 100
	state.ConsensusParams.Block.MaxGas = 100000
	chainID := "test"
	executor := NewBlockExecutor(vKey.PubKey().Address().Bytes(), chainID, mpool, proxy.NewAppConnConsensus(client, proxy.NopMetrics()), eventBus, 100, logger, NopMetrics(), types.GetRandomBytes(32))

	_, err = mpool.CheckTx([]byte{1, 2, 3, 4}, "")
	require.NoError(err)
	block, err := executor.CreateBlock(1, &types.Commit{Signatures: []types.Signature{types.Signature([]byte{1, 1, 1})}}, abci.ExtendedCommitInfo{}, []byte{}, state)
	require.NoError(err)
	require.NotNil(block)
	assert.Equal(uint64(1), block.Height())
	assert.Len(block.Data.Txs, 1)
	dataHash, err := block.Data.Hash()
	assert.NoError(err)
	block.SignedHeader.DataHash = dataHash

	// Update the signature on the block to current from last
	voteBytes := block.SignedHeader.Header.MakeCometBFTVote()
	sig, _ := vKey.Sign(voteBytes)
	block.SignedHeader.Commit = types.Commit{
		Signatures: []types.Signature{sig},
	}
	block.SignedHeader.Validators = cmtypes.NewValidatorSet(validators)

	newState, resp, err := executor.ApplyBlock(context.Background(), state, block)
	require.NoError(err)
	require.NotNil(newState)
	require.NotNil(resp)
	assert.Equal(uint64(1), newState.LastBlockHeight)
	appHash, _, err := executor.Commit(context.Background(), newState, block, resp)
	require.NoError(err)
	assert.Equal(mockAppHash, appHash)

	_, err = mpool.CheckTx([]byte{0, 1, 2, 3, 4}, "")
	require.NoError(err)
	_, err = mpool.CheckTx([]byte{5, 6, 7, 8, 9}, "")
	require.NoError(err)
	_, err = mpool.CheckTx([]byte{1, 2, 3, 4, 5}, "")
	require.NoError(err)
	_, err = mpool.CheckTx(make([]byte, 90), "")
	require.NoError(err)
	block, err = executor.CreateBlock(2, &types.Commit{Signatures: []types.Signature{types.Signature([]byte{1, 1, 1})}}, abci.ExtendedCommitInfo{}, []byte{}, newState)
	require.NoError(err)
	require.NotNil(block)
	assert.Equal(uint64(2), block.Height())
	assert.Len(block.Data.Txs, 3)
	dataHash, err = block.Data.Hash()
	assert.NoError(err)
	block.SignedHeader.DataHash = dataHash

	voteBytes = block.SignedHeader.Header.MakeCometBFTVote()
	sig, _ = vKey.Sign(voteBytes)
	block.SignedHeader.Commit = types.Commit{
		Signatures: []types.Signature{sig},
	}
	block.SignedHeader.Validators = cmtypes.NewValidatorSet(validators)

	newState, resp, err = executor.ApplyBlock(context.Background(), newState, block)
	require.NoError(err)
	require.NotNil(newState)
	require.NotNil(resp)
	assert.Equal(uint64(2), newState.LastBlockHeight)
	_, _, err = executor.Commit(context.Background(), newState, block, resp)
	require.NoError(err)

	// wait for at least 4 Tx events, for up to 3 second.
	// 3 seconds is a fail-scenario only
	timer := time.NewTimer(3 * time.Second)
	txs := make(map[int64]int)
	cnt := 0
	for cnt != 4 {
		select {
		case evt := <-txSub.Out():
			cnt++
			data, ok := evt.Data().(cmtypes.EventDataTx)
			assert.True(ok)
			assert.NotEmpty(data.Tx)
			txs[data.Height]++
		case <-timer.C:
			t.FailNow()
		}
	}
	assert.Zero(len(txSub.Out())) // expected exactly 4 Txs - channel should be empty
	assert.EqualValues(1, txs[1])
	assert.EqualValues(3, txs[2])

	require.EqualValues(2, len(headerSub.Out()))
	for h := 1; h <= 2; h++ {
		evt := <-headerSub.Out()
		_, ok := evt.Data().(cmtypes.EventDataNewBlockHeader)
		assert.True(ok)
	}
}

func TestApplyBlockWithFraudProofsDisabled(t *testing.T) {
	doTestApplyBlock(t)
}

func TestUpdateStateConsensusParams(t *testing.T) {
	logger := log.TestingLogger()
	app := &mocks.Application{}
	client, err := proxy.NewLocalClientCreator(app).NewABCIConsensusClient()
	require.NoError(t, err)
	require.NotNil(t, client)

	chainID := "test"

	mpool := mempool.NewCListMempool(cfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(client, proxy.NopMetrics()), 0)
	eventBus := cmtypes.NewEventBus()
	require.NoError(t, eventBus.Start())
	executor := NewBlockExecutor([]byte("test address"), chainID, mpool, proxy.NewAppConnConsensus(client, proxy.NopMetrics()), eventBus, 100, logger, NopMetrics(), types.GetRandomBytes(32))

	state := types.State{
		ConsensusParams: cmproto.ConsensusParams{
			Block: &cmproto.BlockParams{
				MaxBytes: 100,
				MaxGas:   100000,
			},
			Validator: &cmproto.ValidatorParams{
				PubKeyTypes: []string{cmtypes.ABCIPubKeyTypeEd25519},
			},
			Version: &cmproto.VersionParams{
				App: 1,
			},
			Feature: &cmproto.FeatureParams{},
		},
	}

	block := types.GetRandomBlock(1234, 2)

	txResults := make([]*abci.ExecTxResult, len(block.Data.Txs))
	for idx := range block.Data.Txs {
		txResults[idx] = &abci.ExecTxResult{
			Code: abci.CodeTypeOK,
		}
	}

	resp := &abci.FinalizeBlockResponse{
		ConsensusParamUpdates: &cmproto.ConsensusParams{
			Block: &cmproto.BlockParams{
				MaxBytes: 200,
				MaxGas:   200000,
			},
			Validator: &cmproto.ValidatorParams{
				PubKeyTypes: []string{cmtypes.ABCIPubKeyTypeEd25519},
			},
			Version: &cmproto.VersionParams{
				App: 2,
			},
		},
		TxResults: txResults,
	}

	updatedState, err := executor.updateState(state, block, resp)
	require.NoError(t, err)

	assert.Equal(t, uint64(1235), updatedState.LastHeightConsensusParamsChanged)
	assert.Equal(t, int64(200), updatedState.ConsensusParams.Block.MaxBytes)
	assert.Equal(t, int64(200000), updatedState.ConsensusParams.Block.MaxGas)
	assert.Equal(t, uint64(2), updatedState.ConsensusParams.Version.App)
}

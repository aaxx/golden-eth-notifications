package ethNotification

import (
	"context"
	"log"
)

type ethSub struct {
	engine             *Engine
	pendingTxSubChan   chan string
	newBlockSubChan    chan Block
	context            context.Context
	cancel             context.CancelFunc
	currentBlockNumber string
}

func newETHSub(engine *Engine) ethSub {
	ctx, cancelFunc := context.WithCancel(context.Background())

	return ethSub{
		engine:           engine,
		pendingTxSubChan: make(chan string),
		newBlockSubChan:  make(chan Block),
		context:          ctx,
		cancel:           cancelFunc,
	}
}

func (es *ethSub) StartEtherSub() {
	clientT := es.engine.cT
	clientB := es.engine.cB

	subTx, errSubTx := clientT.EthSubscribe(context.Background(), es.pendingTxSubChan, "newPendingTransactions")
	subBlock, errSubBlock := clientB.EthSubscribe(context.Background(), es.newBlockSubChan, "newHeads")

	unsubsribe := func() {
		if subBlock != nil {
			subBlock.Unsubscribe()
		}

		if subTx != nil {
			subTx.Unsubscribe()
		}
	}

	defer func() {
		unsubsribe()
		// go es.StartEtherSub()
	}()

	if errSubTx != nil || errSubBlock != nil {
		log.Println("No channel")
		return
	}

	for {
		select {
		case txHash := <-es.pendingTxSubChan:
			if !es.engine.isAllowPendingTx {
				subTx.Unsubscribe()
			} else {
				go func(th string) {
					log.Println("Transaction - " + th)
					NewTxHashHandler(es.engine, th).Handle()
				}(txHash)
			}
		case blockHeader := <-es.newBlockSubChan:
			if es.currentBlockNumber != blockHeader.Number {
				es.currentBlockNumber = blockHeader.Number

				go func(bh Block) {
					log.Println("Block - " + bh.Number)
					NewBlockHashHandler(es.engine, bh.Hash).Handle()
				}(blockHeader)
			}
		case <-subTx.Err():
		case blockErr := <-subBlock.Err():
			log.Println("Block sub error: ", blockErr)
		case <-es.context.Done():
			log.Println("Engine Done.")
			break
		}
	}
}

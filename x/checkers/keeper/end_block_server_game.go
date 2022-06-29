package keeper

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	rules "github.com/tolik22869/checkers/x/checkers/rules"
	"github.com/tolik22869/checkers/x/checkers/types"
)

func (k Keeper) ForfeitExpiredGames(goCtx context.Context) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	opponents := map[string]string{
		rules.PieceStrings[rules.BLACK_PLAYER]: rules.PieceStrings[rules.RED_PLAYER],
		rules.PieceStrings[rules.RED_PLAYER]:   rules.PieceStrings[rules.BLACK_PLAYER],
	}

	nextGame, found := k.GetNextGame(ctx)
	if !found {
		panic("NextGame not found")
	}
	storedGameId := nextGame.FifoHead
	var storedGame types.StoredGame
	for {
		if strings.Compare(storedGameId, types.NoFifoIdKey) == 0 {
			break
		}
		storedGame, found = k.GetStoredGame(ctx, storedGameId)
		if !found {
			panic("Fifo head game not found " + nextGame.FifoHead)
		}
		deadline, err := storedGame.GetDeadlineAsTime()
		if err != nil {
			panic(err)
		}

		if deadline.Before(ctx.BlockTime()) {
			k.RemoveFromFifo(ctx, &storedGame, &nextGame)
			if storedGame.MoveCount <= 1 {
				// No point in keeping a game that was never really played
				k.RemoveStoredGame(ctx, storedGameId)
			} else {
				storedGame.Winner, found = opponents[storedGame.Turn]
				if !found {
					panic(fmt.Sprintf(types.ErrCannotFindWinnerByColor.Error(), storedGame.Turn))
				}
				k.SetStoredGame(ctx, storedGame)
			}
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(sdk.EventTypeMessage,
					sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
					sdk.NewAttribute(sdk.AttributeKeyAction, types.ForfeitGameEventKey),
					sdk.NewAttribute(types.ForfeitGameEventIdValue, storedGameId),
					sdk.NewAttribute(types.ForfeitGameEventWinner, storedGame.Winner),
				),
			)
			// Move along FIFO
			storedGameId = nextGame.FifoHead
		} else {
			// All other games come after anyway
			break
		}
	}
	k.SetNextGame(ctx, nextGame)
}

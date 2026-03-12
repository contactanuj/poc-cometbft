package app

import (
	"encoding/json"
	"fmt"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

func (app *App) ExportAppStateAndValidators(forZeroHeight bool, jailAllowedAddrs, modulesToExport []string) (servertypes.ExportedApp, error) {
	ctx := app.NewContext(true)

	genState, err := app.ModuleManager.ExportGenesisForModules(ctx, app.appCodec, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, fmt.Errorf("failed to marshal app state: %w", err)
	}

	return servertypes.ExportedApp{
		AppState: appState,
	}, nil
}

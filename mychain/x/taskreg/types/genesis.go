package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Tasks:      []Task{},
		NextTaskId: 1,
	}
}

func (gs GenesisState) Validate() error {
	taskIDs := make(map[uint64]bool)
	for _, task := range gs.Tasks {
		if taskIDs[task.Id] {
			return ErrInvalidStatus.Wrapf("duplicate task id: %d", task.Id)
		}
		taskIDs[task.Id] = true
	}
	return nil
}

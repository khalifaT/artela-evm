package vm

import (
	"bytes"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type State struct {
	Account common.Address // the account that triggered the state change
	Value   []byte         // raw data of the updated state
}

// Eq compares two states, return true if two states are equal
func (s *State) Eq(other *State) bool {
	return other != nil &&
		bytes.Compare(other.Account[:], s.Account[:]) == 0 &&
		bytes.Compare(other.Value, s.Value) == 0
}

// StateChanges saves the changes of current state
// the mapping is address -> slot -> index -> changes
type StateChanges struct {
	slotIndex map[common.Address]map[uint256.Int]string
	changes   map[common.Address]map[string]map[string][]*State
}

func NewStateChanges() *StateChanges {
	return &StateChanges{
		slotIndex: make(map[common.Address]map[uint256.Int]string),
		changes:   make(map[common.Address]map[string]map[string][]*State),
	}
}

// Save saves a state change, if state already cached, skip the saving
func (s *StateChanges) Save(account common.Address, stateVarName string, slot *uint256.Int, index string, newState *State) error {
	if slot == nil {
		return errors.New("slot cannot be nil")
	}

	accountSlotIndex, ok := s.slotIndex[account]
	if !ok {
		s.slotIndex[account] = make(map[uint256.Int]string)
		accountSlotIndex = s.slotIndex[account]
	}
	if _, ok := accountSlotIndex[*slot]; !ok {
		accountSlotIndex[*slot] = stateVarName
	}

	accountChange, ok := s.changes[account]
	if !ok {
		s.changes[account] = make(map[string]map[string][]*State, 1)
		accountChange = s.changes[account]
	}
	slotChange, ok := accountChange[stateVarName]
	if !ok {
		accountChange[stateVarName] = make(map[string][]*State, 1)
		slotChange = accountChange[stateVarName]
	}
	stateChange, ok := slotChange[index]
	if !ok {
		slotChange[index] = make([]*State, 0, 1)
		stateChange = slotChange[index]
	}

	// compare with last state change, skip if equal
	count := len(stateChange)
	if count > 0 && newState.Eq(stateChange[count-1]) {
		return nil
	}

	// initial state, state change account should be empty
	if count == 0 {
		newState.Account = common.Address{}
	}

	slotChange[index] = append(stateChange, newState)
	return nil
}

// Variable looks up state changes by variable name
func (s *StateChanges) Variable(account common.Address, stateVarName string, index string) []*State {
	states, ok := s.changes[account][stateVarName][index]
	if !ok {
		return nil
	}

	return states
}

// Slot looks up state changes by storage slot
func (s *StateChanges) Slot(account common.Address, slot *uint256.Int, index string) []*State {
	if slot == nil {
		return nil
	}
	stateVar, ok := s.slotIndex[account][*slot]
	if !ok {
		return nil
	}

	return s.Variable(account, stateVar, index)
}

// Monitor monitors the state changes and traces the call stack changes during a tx execution
type Monitor struct {
	states *StateChanges
}

func NewMonitor() *Monitor {
	return &Monitor{states: NewStateChanges()}
}

func (m *Monitor) StateChanges() *StateChanges {
	return m.states
}

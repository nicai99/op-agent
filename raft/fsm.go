package oraft

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/hashicorp/raft"
	"github.com/openark/golib/log"
)

// fsm is a raft finite state machine
type fsm Store

// Apply applies a Raft log entry to the key-value store.
func (f *fsm) Apply(l *raft.Log) interface{} {
	var c storeCommand
	if err := json.Unmarshal(l.Data, &c); err != nil {
		log.Errorf("failed to unmarshal command: %s", err.Error())
	}

	if c.Op == YieldCommand {
		toPeer, err := normalizeRaftNode(string(c.Value))
		if err != nil {
			return log.Errore(err)
		}
		return f.yield(toPeer)
	}
	if c.Op == YieldHintCommand {
		hint := string(c.Value)
		return f.yieldByHint(hint)
	}
	log.Debugf("oraft: applying command %+v: %s", l.Index, c.Op)
	return store.applier.ApplyCommand(c.Op, c.Value)
}

// yield yields to a suggested peer, or does nothing if this peer IS the suggested peer
func (f *fsm) yield(toPeer string) interface{} {
	isThisPeer, err := IsPeer(toPeer)
	if err != nil {
		return log.Errorf("failed to unmarshal command: %s", err.Error())
	}
	if isThisPeer {
		log.Debugf("Will not yield to myself")
		return nil
	}
	log.Debugf("Yielding to %s", toPeer)
	return Yield()
}

// yieldByHint yields to a host that contains given hint
func (f *fsm) yieldByHint(hint string) interface{} {
	if hint == "" {
		log.Debugf("Will not yield by empty hint")
		return nil
	}
	isThisHost := strings.Contains(ThisHostname, hint)
	if isThisHost {
		log.Debugf("Will not yield to myself")
		return nil
	}
	log.Debugf("Yielding to hinted %s", hint)
	return Yield()
}

// Snapshot returns a snapshot object of freno's state
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	snapshot := newFsmSnapshot(f.snapshotCreatorApplier)
	return snapshot, nil
}

// Restore restores freno state
func (f *fsm) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	return f.snapshotCreatorApplier.Restore(rc)
}

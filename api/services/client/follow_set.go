package services

import (
	"sort"

	"github.com/astralp2p/astral-go/api/services"
)

// followSet is the set of offerings one follow shows, across reopens. A
// reopened follow is staged until its boundary and then applied as one
// difference.
type followSet struct {
	offerings map[watchKey]*services.Update
	ungranted []string

	// removals holds the removed offerings not offered again, each with its
	// last view and the cause of its removal.
	removals map[watchKey]Removal

	// unconfirmed holds the shown keys an incomplete boundary kept without
	// the reopened follow showing them; an update or a removal of a key
	// confirms it.
	unconfirmed map[watchKey]bool

	// staging holds a reopened follow's view until its boundary: the last
	// update per key, nil for a key it removed. A nil map means no follow is
	// staged.
	staging  map[watchKey]*services.Update
	stagedNG []string
	// stagedCause holds the cause of each key the staged follow removed.
	stagedCause map[watchKey]services.RemovalCause
}

func newFollowSet() followSet {
	return followSet{
		offerings:   map[watchKey]*services.Update{},
		removals:    map[watchKey]Removal{},
		unconfirmed: map[watchKey]bool{},
	}
}

// stale reports whether the shown set holds offerings no follow confirmed
// since a stream ended.
func (s *followSet) stale() bool {
	return s.staging != nil || len(s.unconfirmed) > 0
}

func keyOf(u *services.Update) watchKey {
	return watchKey{u.ProviderID.String(), string(u.Name)}
}

// reopen marks the offerings stale and stages the next follow from empty.
func (s *followSet) reopen() {
	s.staging = map[watchKey]*services.Update{}
	s.stagedNG = nil
	s.stagedCause = map[watchKey]services.RemovalCause{}
}

// add takes an update, a removal or a not_granted notice. While a follow is
// staged it goes to staging; add reports whether the shown set changed.
func (s *followSet) add(ev Event) bool {
	if s.staging != nil {
		s.stage(ev)
		return false
	}
	switch {
	case ev.Update != nil:
		delete(s.unconfirmed, keyOf(ev.Update))
		delete(s.removals, keyOf(ev.Update))
		if bool(ev.Update.Available) {
			s.offerings[keyOf(ev.Update)] = ev.Update
		} else {
			delete(s.offerings, keyOf(ev.Update))
		}
	case ev.NotGranted != nil:
		s.ungranted = ev.NotGranted
	case ev.Removed != nil:
		for _, rk := range ev.Removed {
			k := watchKey{rk.ProviderID.String(), string(rk.Name)}
			delete(s.unconfirmed, k)
			s.remove(k, ev.RemovedCause)
		}
	default:
		return false
	}
	return true
}

func (s *followSet) stage(ev Event) {
	switch {
	case ev.Update != nil:
		s.staging[keyOf(ev.Update)] = ev.Update
	case ev.NotGranted != nil:
		s.stagedNG = ev.NotGranted
	case ev.Removed != nil:
		for _, rk := range ev.Removed {
			k := watchKey{rk.ProviderID.String(), string(rk.Name)}
			s.staging[k] = nil
			s.stagedCause[k] = ev.RemovedCause
		}
	}
}

// remove drops a shown offering and keeps it as a removal with cause. A key
// not shown leaves no removal.
func (s *followSet) remove(k watchKey, cause services.RemovalCause) {
	if old, ok := s.offerings[k]; ok {
		s.removals[k] = Removal{Offering: old, Cause: cause}
		delete(s.offerings, k)
	}
}

// settle applies a staged follow at its boundary and returns the events that
// carry the difference: a NotGranted notice, the updates the follow shows,
// and one Removed per cause for the offerings it no longer shows, in
// ascending cause order. An offering the staged follow removed keeps the
// node's cause; an offering a complete outcome leaves out has cause zero,
// since no node removed it. Without a staged follow it does nothing.
//
// why only a complete outcome prunes: an offering missing from an incomplete
// outcome may belong to a member that did not answer in time, so it is kept
// and stays stale until a later update or removal of it, or a later complete
// boundary.
func (s *followSet) settle(o InitialOutcome) []Event {
	if s.staging == nil {
		return nil
	}
	var diff []Event
	s.ungranted = s.stagedNG
	if s.stagedNG != nil {
		diff = append(diff, Event{NotGranted: s.stagedNG})
	}
	removed := map[services.RemovalCause][]*services.OfferingKey{}
	drop := func(k watchKey, cause services.RemovalCause) {
		old := s.offerings[k]
		removed[cause] = append(removed[cause], &services.OfferingKey{ProviderID: old.ProviderID, Name: old.Name})
		s.remove(k, cause)
	}
	for _, k := range sortedKeys(s.staging) {
		u := s.staging[k]
		if u != nil && bool(u.Available) {
			s.offerings[k] = u
			delete(s.removals, k)
			diff = append(diff, Event{Update: u})
			continue
		}
		if _, ok := s.offerings[k]; ok {
			drop(k, s.stagedCause[k])
			if u != nil {
				// the provider's own unavailable answer is no removal, as in add
				delete(s.removals, k)
			}
		}
	}
	s.unconfirmed = map[watchKey]bool{}
	for _, k := range sortedKeys(s.offerings) {
		if _, seen := s.staging[k]; seen {
			continue
		}
		if !o.Complete {
			s.unconfirmed[k] = true
			continue
		}
		drop(k, 0)
	}
	causes := make([]services.RemovalCause, 0, len(removed))
	for c := range removed {
		causes = append(causes, c)
	}
	sort.Slice(causes, func(i, j int) bool { return causes[i] < causes[j] })
	for _, c := range causes {
		diff = append(diff, Event{Removed: removed[c], RemovedCause: c})
	}
	s.staging, s.stagedNG, s.stagedCause = nil, nil, nil
	return diff
}

// keys returns the shown offerings' keys, ordered by provider and name.
func (s *followSet) keys() []watchKey {
	return sortedKeys(s.offerings)
}

func sortedKeys(m map[watchKey]*services.Update) []watchKey {
	keys := make([]watchKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortWatchKeys(keys)
	return keys
}

// sortWatchKeys orders keys by provider and name.
func sortWatchKeys(keys []watchKey) {
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].provider != keys[j].provider {
			return keys[i].provider < keys[j].provider
		}
		return keys[i].name < keys[j].name
	})
}

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
	stale     bool // offerings holds keys no follow confirmed since a stream ended

	// staging holds a reopened follow's view until its boundary: the last
	// update per key, nil for a key it removed. A nil map means no follow is
	// staged.
	staging  map[watchKey]*services.Update
	stagedNG []string
}

func newFollowSet() followSet {
	return followSet{offerings: map[watchKey]*services.Update{}}
}

func keyOf(u *services.Update) watchKey {
	return watchKey{u.ProviderID.String(), string(u.Name)}
}

// reopen marks the offerings stale and stages the next follow from empty.
func (s *followSet) reopen() {
	s.stale = true
	s.staging = map[watchKey]*services.Update{}
	s.stagedNG = nil
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
		if bool(ev.Update.Available) {
			s.offerings[keyOf(ev.Update)] = ev.Update
		} else {
			delete(s.offerings, keyOf(ev.Update))
		}
	case ev.NotGranted != nil:
		s.ungranted = ev.NotGranted
	case ev.Removed != nil:
		for _, rk := range ev.Removed {
			delete(s.offerings, watchKey{rk.ProviderID.String(), string(rk.Name)})
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
			s.staging[watchKey{rk.ProviderID.String(), string(rk.Name)}] = nil
		}
	}
}

// settle applies a staged follow at its boundary and returns the events that
// carry the difference: a NotGranted notice, the updates the follow shows,
// and one Removed for the offerings it no longer shows. Without a staged
// follow it does nothing.
//
// why only a complete outcome prunes: an offering missing from an incomplete
// outcome may belong to a member that did not answer in time, so it is kept
// and the set stays stale.
func (s *followSet) settle(o InitialOutcome) []Event {
	if s.staging == nil {
		return nil
	}
	var diff []Event
	s.ungranted = s.stagedNG
	if s.stagedNG != nil {
		diff = append(diff, Event{NotGranted: s.stagedNG})
	}
	var removed []*services.OfferingKey
	for _, k := range sortedKeys(s.staging) {
		u := s.staging[k]
		if u != nil && bool(u.Available) {
			s.offerings[k] = u
			diff = append(diff, Event{Update: u})
			continue
		}
		if old, ok := s.offerings[k]; ok {
			removed = append(removed, &services.OfferingKey{ProviderID: old.ProviderID, Name: old.Name})
			delete(s.offerings, k)
		}
	}
	if o.Complete {
		for _, k := range sortedKeys(s.offerings) {
			if _, seen := s.staging[k]; !seen {
				old := s.offerings[k]
				removed = append(removed, &services.OfferingKey{ProviderID: old.ProviderID, Name: old.Name})
				delete(s.offerings, k)
			}
		}
	}
	if len(removed) > 0 {
		diff = append(diff, Event{Removed: removed})
	}
	s.stale = !o.Complete
	s.staging, s.stagedNG = nil, nil
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
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].provider != keys[j].provider {
			return keys[i].provider < keys[j].provider
		}
		return keys[i].name < keys[j].name
	})
	return keys
}

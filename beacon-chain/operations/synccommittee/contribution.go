package synccommittee

import (
	"strconv"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/container/queue"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

// To give two slots tolerance for objects that arrive earlier.
// This account for previous slot, current slot, two future slots.
const syncCommitteeMaxQueueSize = 8

// SaveSyncCommitteeContribution saves a sync committee contribution in to a priority queue.
// The priority queue is capped at syncCommitteeMaxQueueSize contributions.
func (s *Store) SaveSyncCommitteeContribution(cont *ethpb.SyncCommitteeContribution) error {
	if cont == nil {
		return errNilContribution
	}

	log.WithFields(log.Fields{
		"slot":  cont.Slot,
		"index": cont.SubcommitteeIndex,
		"bits":  cont.AggregationBits.Count(),
	}).Warn("Received sync committee contribution")

	s.contributionLock.Lock()
	defer s.contributionLock.Unlock()

	item, err := s.contributionCache.PopByKey(syncCommitteeKey(cont.Slot))
	if err != nil {
		return err
	}

	copied := ethpb.CopySyncCommitteeContribution(cont)

	// Contributions exist in the queue. Append instead of insert new.
	if item != nil {
		contributions, ok := item.Value.([]*ethpb.SyncCommitteeContribution)
		if !ok {
			return errors.New("not typed []ethpb.SyncCommitteeContribution")
		}

		contributions = append(contributions, copied)
		savedSyncCommitteeContributionTotal.Inc()
		log.WithFields(log.Fields{
			"slot":  cont.Slot,
			"index": cont.SubcommitteeIndex,
			"count": len(contributions),
			"bits":  cont.AggregationBits.Count(),
		}).Warn("Added sync committee contribution to pool")
		return s.contributionCache.Push(&queue.Item{
			Key:      syncCommitteeKey(cont.Slot),
			Value:    contributions,
			Priority: int64(cont.Slot),
		})
	}

	// Contribution does not exist. Insert new.
	log.WithFields(log.Fields{
		"slot":  cont.Slot,
		"index": cont.SubcommitteeIndex,
		"count": 1,
	}).Warn("Added new sync committee contribution to pool")
	if err := s.contributionCache.Push(&queue.Item{
		Key:      syncCommitteeKey(cont.Slot),
		Value:    []*ethpb.SyncCommitteeContribution{copied},
		Priority: int64(cont.Slot),
	}); err != nil {
		return err
	}
	savedSyncCommitteeContributionTotal.Inc()

	// Trim contributions in queue down to syncCommitteeMaxQueueSize.
	if s.contributionCache.Len() > syncCommitteeMaxQueueSize {
		item, err = s.contributionCache.Pop()
		log.WithFields(log.Fields{
			"slot": item.Priority,
		}).Warn("Popped sync committee contributions")
		return err
	}

	return nil
}

// SyncCommitteeContributions returns sync committee contributions by slot from the priority queue.
// Upon retrieval, the contribution is removed from the queue.
func (s *Store) SyncCommitteeContributions(slot types.Slot) ([]*ethpb.SyncCommitteeContribution, error) {
	s.contributionLock.RLock()
	defer s.contributionLock.RUnlock()

	item := s.contributionCache.RetrieveByKey(syncCommitteeKey(slot))
	log.WithFields(log.Fields{
		"slot": slot,
	}).Warn("Empty sync committee contributions")
	if item == nil {
		return []*ethpb.SyncCommitteeContribution{}, nil
	}

	contributions, ok := item.Value.([]*ethpb.SyncCommitteeContribution)
	log.WithFields(log.Fields{
		"slot":  slot,
		"count": len(contributions),
	}).Warn("Retrieved sync committee contributions")
	if !ok {
		return nil, errors.New("not typed []ethpb.SyncCommitteeContribution")
	}

	return contributions, nil
}

func syncCommitteeKey(slot types.Slot) string {
	return strconv.FormatUint(uint64(slot), 10)
}

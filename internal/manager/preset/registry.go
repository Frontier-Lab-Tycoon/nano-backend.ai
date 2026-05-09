package preset

import (
	"context"
	"fmt"
	"sort"

	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
)

// StaticRegistry stores presets in memory.
type StaticRegistry struct {
	presets map[ID]*TrainerPreset
}

// NewStaticRegistry creates an in-memory registry from structured presets.
func NewStaticRegistry(presets ...TrainerPreset) StaticRegistry {
	registered := make(map[ID]*TrainerPreset, len(presets))
	for _, trainerPreset := range presets {
		presetCopy := trainerPreset
		registered[trainerPreset.ID] = &presetCopy
	}
	return StaticRegistry{presets: registered}
}

// Get returns a preset by stable ID.
func (r StaticRegistry) Get(ctx context.Context, id ID) (Preset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	trainerPreset, ok := r.presets[id]
	if !ok {
		return nil, errordef.Errorf(errordef.NotFound, "preset %q not found", id)
	}
	return trainerPreset, nil
}

// List returns all registered presets ordered by stable ID.
func (r StaticRegistry) List(ctx context.Context) ([]Preset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(r.presets))
	for id := range r.presets {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)

	presets := make([]Preset, 0, len(ids))
	for _, id := range ids {
		trainerPreset, ok := r.presets[ID(id)]
		if !ok {
			return nil, fmt.Errorf("preset registry changed while listing")
		}
		presets = append(presets, trainerPreset)
	}
	return presets, nil
}

// NewPhase0Registry creates the default Phase 0 trainer preset registry.
func NewPhase0Registry() StaticRegistry {
	return NewStaticRegistry(Phase0Presets()...)
}

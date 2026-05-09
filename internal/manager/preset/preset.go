package preset

import "github.com/seedspirit/nano-backend.ai/internal/common/run"

// ID is the stable identity for a trainer preset.
type ID = run.PresetID

const (
	// PresetAxolotlLoRASFT identifies the Phase 0 Axolotl LoRA SFT preset.
	PresetAxolotlLoRASFT ID = "axolotl-lora-sft"
	// PresetUnslothLoRASFT identifies the Phase 0 Unsloth LoRA SFT preset.
	PresetUnslothLoRASFT ID = "unsloth-lora-sft"
)

// Preset is the validation and finalization contract consumed by RunSpec code.
type Preset interface {
	PresetID() ID
	OptionPolicy() OptionPolicy
	Defaults() map[string]any
}

// TrainerPreset defines a trainer runtime and the parameter surface exposed to users.
type TrainerPreset struct {
	ID            ID
	DisplayName   string
	Runtime       RuntimeSpec
	DefaultValues map[string]any
	Policy        OptionPolicy
}

// RuntimeSpec describes how a preset is materialized by a runtime adapter.
type RuntimeSpec struct {
	Image      string
	Entrypoint []string
	Env        map[string]string
}

// OptionPolicy describes which parameter keys and value shapes a preset accepts.
type OptionPolicy struct {
	Rules map[string]OptionRule
}

// OptionRule constrains a single parameter key.
type OptionRule struct {
	Type OptionValueType
	Min  *float64
	Max  *float64
}

// OptionValueType is a typed string enum used by validators.
type OptionValueType string

const (
	// OptionString requires a string parameter value.
	OptionString OptionValueType = "string"
	// OptionInt requires an integer parameter value.
	OptionInt OptionValueType = "int"
	// OptionFloat requires a numeric parameter value.
	OptionFloat OptionValueType = "float"
	// OptionBool requires a boolean parameter value.
	OptionBool OptionValueType = "bool"
)

// PresetID returns the stable identity of the preset.
func (p *TrainerPreset) PresetID() ID {
	return p.ID
}

// OptionPolicy returns a copy of the preset option policy.
func (p *TrainerPreset) OptionPolicy() OptionPolicy {
	return OptionPolicy{Rules: cloneRules(p.Policy.Rules)}
}

// Defaults returns a copy of the preset default training values.
func (p *TrainerPreset) Defaults() map[string]any {
	return cloneAnyMap(p.DefaultValues)
}

func cloneRules(rules map[string]OptionRule) map[string]OptionRule {
	if rules == nil {
		return nil
	}
	cloned := make(map[string]OptionRule, len(rules))
	for key, rule := range rules {
		cloned[key] = cloneRule(rule)
	}
	return cloned
}

func cloneRule(rule OptionRule) OptionRule {
	cloned := rule
	if rule.Min != nil {
		value := *rule.Min
		cloned.Min = &value
	}
	if rule.Max != nil {
		value := *rule.Max
		cloned.Max = &value
	}
	return cloned
}

func cloneAnyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = cloneAny(value)
	}
	return cloned
}

func cloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		cloned := make([]any, len(typed))
		for i, item := range typed {
			cloned[i] = cloneAny(item)
		}
		return cloned
	default:
		return value
	}
}

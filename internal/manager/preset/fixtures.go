package preset

// Phase0Presets returns the structured trainer presets supported in Phase 0.
func Phase0Presets() []TrainerPreset {
	return []TrainerPreset{
		AxolotlLoRASFT(),
		UnslothLoRASFT(),
	}
}

// AxolotlLoRASFT returns the Phase 0 Axolotl LoRA SFT trainer preset.
func AxolotlLoRASFT() TrainerPreset {
	return TrainerPreset{
		ID:          PresetAxolotlLoRASFT,
		DisplayName: "Axolotl LoRA SFT",
		Runtime: RuntimeSpec{
			Image:      "axolotl:latest",
			Entrypoint: []string{"axolotl", "train", "/workspace/resolved_config.yaml"},
			Env: map[string]string{
				"HF_HOME": "/cache/huggingface",
			},
		},
		DefaultValues: map[string]any{
			"learning_rate":    2.0e-4,
			"num_epochs":       3,
			"max_seq_length":   4096,
			"lora_r":           16,
			"lora_alpha":       32,
			"micro_batch_size": 1,
		},
		Policy: loraSFTOptionPolicy(),
	}
}

// UnslothLoRASFT returns the Phase 0 Unsloth LoRA SFT trainer preset.
func UnslothLoRASFT() TrainerPreset {
	return TrainerPreset{
		ID:          PresetUnslothLoRASFT,
		DisplayName: "Unsloth LoRA SFT",
		Runtime: RuntimeSpec{
			Image:      "unsloth:latest",
			Entrypoint: []string{"python", "-m", "nano_backend.train_unsloth", "--config", "/workspace/resolved_config.yaml"},
			Env: map[string]string{
				"HF_HOME": "/cache/huggingface",
			},
		},
		DefaultValues: map[string]any{
			"learning_rate":    2.0e-4,
			"num_epochs":       3,
			"max_seq_length":   4096,
			"lora_r":           16,
			"lora_alpha":       32,
			"micro_batch_size": 1,
		},
		Policy: loraSFTOptionPolicy(),
	}
}

func loraSFTOptionPolicy() OptionPolicy {
	return OptionPolicy{
		Rules: map[string]OptionRule{
			"learning_rate": {
				Type: OptionFloat,
				Min:  float64Ptr(0),
				Max:  float64Ptr(1),
			},
			"num_epochs": {
				Type: OptionInt,
				Min:  float64Ptr(1),
				Max:  float64Ptr(100),
			},
			"max_seq_length": {
				Type: OptionInt,
				Min:  float64Ptr(128),
				Max:  float64Ptr(32768),
			},
			"lora_r": {
				Type: OptionInt,
				Min:  float64Ptr(1),
				Max:  float64Ptr(256),
			},
			"lora_alpha": {
				Type: OptionInt,
				Min:  float64Ptr(1),
				Max:  float64Ptr(512),
			},
			"micro_batch_size": {
				Type: OptionInt,
				Min:  float64Ptr(1),
				Max:  float64Ptr(64),
			},
		},
	}
}

func float64Ptr(value float64) *float64 {
	return &value
}

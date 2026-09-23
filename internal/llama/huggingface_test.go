package llama

import "testing"

func TestQuantPat(t *testing.T) {
	m := quantPat.FindStringSubmatch("Llama-3.2-1B-Instruct-Q4_K_M")
	if len(m) < 2 || m[1] != "Q4_K_M" {
		t.Fatalf("%v", m)
	}
}

func TestQuantGuidance(t *testing.T) {
	for _, tc := range []struct {
		name, guidance string
		size, memory   int64
	}{
		{"modest", "Good starting point for modest hardware", 2 << 30, 2<<30 + (2<<30)/5},
		{"large", "Large model; check available memory first", 8 << 30, 8<<30 + (8<<30)/5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			memory, guidance := quantGuidance(tc.size)
			if memory != tc.memory || guidance != tc.guidance {
				t.Fatalf("quantGuidance = %d, %q", memory, guidance)
			}
		})
	}
	if memory, guidance := quantGuidance(0); memory != 0 || guidance != "" {
		t.Fatalf("unknown size = %d, %q", memory, guidance)
	}
}

// A quant whose files report no size is listed (without a size), not dropped:
// Q4_K_M is the recommended default and must stay first.
func TestHFQuantWithoutSizeIsKept(t *testing.T) {
	raw := []byte(`{"id":"o/m","siblings":[{"rfilename":"m-Q4_K_M.gguf","size":0},{"rfilename":"m-Q8_0.gguf","size":10},{"rfilename":"mmproj-f16.gguf","size":5}]}`)
	d, err := hfDetailsFrom(raw, "o/m")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Quantizations) != 2 || d.Quantizations[0].Name != "Q4_K_M" || d.Quantizations[0].Size != 0 || d.Quantizations[1].Name != "Q8_0" || d.Quantizations[1].Size != 10 {
		t.Fatalf("quants = %+v", d.Quantizations)
	}
}

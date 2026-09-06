package llama

import "testing"

func TestExecutionProfileArgsAndGuards(t *testing.T) {
	p := ExecutionProfile{Binary: "llama-server", ModelsDir: "/tmp/llama-models", Host: "127.0.0.1", Port: 8080, Context: 8192, GPULayers: 12, Jinja: true, NoAutoLoad: true}
	a, err := p.Args()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--models-dir", "/tmp/llama-models", "--ctx-size", "8192", "--n-gpu-layers", "12", "--jinja", "--no-models-autoload"} {
		found := false
		for _, got := range a {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("args missing %q: %v", want, a)
		}
	}
	for _, bad := range []ExecutionProfile{
		{Binary: "/tmp/other", ModelsDir: "/tmp/m", Host: "127.0.0.1", Port: 8080, Context: 8192},
		{Binary: "llama-server", ModelsDir: "relative", Host: "127.0.0.1", Port: 8080, Context: 8192},
		{Binary: "llama-server", ModelsDir: "/tmp/m", Host: "0.0.0.0", Port: 8080, Context: 8192},
	} {
		if err := bad.Validate(); err == nil {
			t.Errorf("expected guard for %+v", bad)
		}
	}
}

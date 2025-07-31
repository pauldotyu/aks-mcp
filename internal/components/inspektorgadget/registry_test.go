package inspektorgadget

import "testing"

func TestRegisterInspektorGadgetTool(t *testing.T) {
	tool := RegisterInspektorGadgetTool()
	if tool.Name != "inspektor_gadget_observability" {
		t.Errorf("Expected tool name 'inspektor_gadget_observability', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Tool description should not be empty")
	}

	_, ok := tool.InputSchema.Properties["action_params"].(map[string]any)
	if !ok {
		t.Error("action_params should be an object")
		return
	}
	_, ok = tool.InputSchema.Properties["filter_params"].(map[string]any)
	if !ok {
		t.Error("filter_params should be an object")
		return
	}
}

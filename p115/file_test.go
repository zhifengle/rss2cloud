package p115

import "testing"

// Test MoveFlattenFiles
func TestMoveFlattenFiles(t *testing.T) {
	ag, err := New()
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	err = ag.SearchAndMoveFiles("", "", "[Nekomoe kissaten]", 4)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
}

// Test RemoveEmptyDir
func TestRemoveEmptyDir(t *testing.T) {
	ag, err := New()
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	err = ag.RemoveEmptyDir("")
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
}

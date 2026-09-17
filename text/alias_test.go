package text

import (
	"bytes"
	"testing"
	"unsafe"
)

func sameBacking(a, b []byte) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	return unsafe.SliceData(a) == unsafe.SliceData(b)
}

// TestBytesAliasesOriginalWhenUnedited is the memory fix: an unedited document
// must not pay for a second full copy of itself. The editor calls Bytes()
// during construction, so an 86 MB response body used to cost 172 MB.
func TestBytesAliasesOriginalWhenUnedited(t *testing.T) {
	src := []byte("line one\nline two\nline three\n")
	pt := New(src)

	got := pt.Bytes()
	if !bytes.Equal(got, src) {
		t.Fatalf("Bytes() = %q, want %q", got, src)
	}
	if !sameBacking(got, pt.original) {
		t.Error("Bytes() copied an unedited document instead of aliasing original")
	}
}

// TestEditAfterAliasDoesNotCorruptOriginal is the dangerous case: once flat
// aliases original, a rebuild must allocate rather than append into the buffer
// the pieces still point at.
func TestEditAfterAliasDoesNotCorruptOriginal(t *testing.T) {
	src := []byte("AAAA\nBBBB\nCCCC\n")
	keep := append([]byte(nil), src...)
	pt := New(src)

	_ = pt.Bytes() // establish the alias
	pt.Insert(0, []byte("xx"))

	if got := string(pt.Bytes()); got != "xxAAAA\nBBBB\nCCCC\n" {
		t.Fatalf("after insert Bytes() = %q", got)
	}
	if !bytes.Equal(pt.original, keep) {
		t.Errorf("original buffer was mutated: %q, want %q", pt.original, keep)
	}
	if sameBacking(pt.Bytes(), pt.original) {
		t.Error("flat still aliases original after an edit")
	}

	// A delete that happens to restore a whole-original span must stay correct.
	pt.Delete(0, 2)
	if got := string(pt.Bytes()); got != "AAAA\nBBBB\nCCCC\n" {
		t.Errorf("after delete Bytes() = %q", got)
	}
	if !bytes.Equal(pt.original, keep) {
		t.Errorf("original mutated by delete: %q", pt.original)
	}
}

// TestRepeatedEditsStayConsistent exercises the alias/copy transitions.
func TestRepeatedEditsStayConsistent(t *testing.T) {
	pt := New([]byte("0123456789\n"))
	want := "0123456789\n"

	for i := 0; i < 20; i++ {
		_ = pt.Bytes()
		pt.Insert(pt.Len(), []byte("x"))
		want += "x"
		if got := string(pt.Bytes()); got != want {
			t.Fatalf("iteration %d: got %q want %q", i, got, want)
		}
		if lc := pt.LineCount(); lc != 2 {
			t.Fatalf("iteration %d: LineCount = %d, want 2", i, lc)
		}
	}
}

// TestNewSharedDoesNotCopy documents the zero-copy constructor.
func TestNewSharedDoesNotCopy(t *testing.T) {
	src := []byte("hello\nworld\n")
	pt := NewShared(src)

	if !sameBacking(pt.original, src) {
		t.Error("NewShared copied the content instead of sharing it")
	}
	if got := string(pt.Bytes()); got != "hello\nworld\n" {
		t.Errorf("Bytes() = %q", got)
	}
	// Editing a shared table must still leave the shared buffer alone.
	pt.Insert(5, []byte("!"))
	if got := string(pt.Bytes()); got != "hello!\nworld\n" {
		t.Errorf("after insert = %q", got)
	}
	if string(src) != "hello\nworld\n" {
		t.Errorf("shared buffer was mutated: %q", src)
	}
}

// TestNewCopiesSoCallerMutationIsSafe pins the difference between the two
// constructors.
func TestNewCopiesSoCallerMutationIsSafe(t *testing.T) {
	src := []byte("abc\n")
	pt := New(src)
	src[0] = 'Z'

	if got := string(pt.Bytes()); got != "abc\n" {
		t.Errorf("New did not copy: Bytes() = %q after caller mutation", got)
	}
}

// TestLineStartsCorrectOnBothPaths checks line indexing for aliased and copied
// flat views.
func TestLineStartsCorrectOnBothPaths(t *testing.T) {
	pt := New([]byte("a\nbb\nccc"))
	if lc := pt.LineCount(); lc != 3 {
		t.Fatalf("LineCount = %d, want 3", lc)
	}
	for i, want := range []string{"a", "bb", "ccc"} {
		if got := pt.Line(i); got != want {
			t.Errorf("aliased Line(%d) = %q, want %q", i, got, want)
		}
	}

	pt.Insert(2, []byte("X"))
	if lc := pt.LineCount(); lc != 3 {
		t.Fatalf("after edit LineCount = %d, want 3", lc)
	}
	for i, want := range []string{"a", "Xbb", "ccc"} {
		if got := pt.Line(i); got != want {
			t.Errorf("copied Line(%d) = %q, want %q", i, got, want)
		}
	}
}

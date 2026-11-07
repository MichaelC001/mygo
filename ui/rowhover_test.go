package ui

import (
	"image/color"
	"testing"
	"time"
)

// hoverList is a long list of choosable rows, with what says where the
// hover is drawn: the band of the column at x in the hover's color, from
// the row under the pointer at rest.
func hoverList(t *testing.T, onRow func(row int, cell *node)) (tt *Tester, litAt func() (top, bottom int, ok bool)) {
	t.Helper()
	sel := -1
	s := &ListState{Selected: &sel}
	tt = coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreList(c, s, 2000, func(row int) {
				cell := coreRow(c).Height(28).Children(func() { coreTextf(c, "row %d", row) })
				if onRow != nil {
					onRow(row, cell)
				}
			}).Grow(1)
		})
	}, 400, 600)
	tt.Frame()
	tt.Move(300, 300)
	img := tt.Image()
	scale := img.Bounds().Dx() / 400
	var lit color.RGBA = img.RGBAAt(300*scale, 300*scale)
	return tt, func() (top, bottom int, ok bool) {
		im := tt.Image()
		top = -1
		for y := 0; y < im.Bounds().Dy(); y++ {
			if im.RGBAAt(300*scale, y) == lit {
				if top < 0 {
					top = y / scale
				}
				bottom = y / scale
			}
		}
		return top, bottom, top >= 0
	}
}

// While a list scrolls under a pointer standing still no row shows the
// hover, which would trail the pointer by what each frame scrolled; a
// moment after it stops, the row under the pointer does.
func TestRowsShowNoHoverWhileScrolling(t *testing.T) {
	tt, lit := hoverList(t, nil)
	if top, bottom, ok := lit(); !ok || top > 300 || bottom < 300 {
		t.Fatalf("at rest the hover is at %d..%d (%v)", top, bottom, ok)
	}
	for i := range 10 {
		tt.Scroll(300, 300, 0, 90)
		tt.Advance(16 * time.Millisecond)
		if top, _, ok := lit(); ok {
			t.Fatalf("frame %d of the scroll: a row hovers at %d", i, top)
		}
	}
	tt.Advance(rowHoverHold - 20*time.Millisecond)
	if _, _, ok := lit(); ok {
		t.Fatal("hovers before the wait is over")
	}
	tt.Advance(20 * time.Millisecond)
	if top, bottom, ok := lit(); !ok || top > 300 || bottom < 300 {
		t.Errorf("after the wait the hover is at %d..%d (%v)", top, bottom, ok)
	}
}

// A wheel's notch glides on after its one event: no row hovers until the
// wait after the glide, however long the glide takes.
func TestRowsShowNoHoverWhileAWheelGlides(t *testing.T) {
	tt, lit := hoverList(t, nil)
	tt.Wheel(300, 300, 0, 400)
	moved := false
	for i := 0; tt.rt.rowHoverHeld() && i < 500; i++ {
		tt.Advance(8 * time.Millisecond)
		if len(tt.rt.wheeling) > 0 {
			moved = true
			if top, _, ok := lit(); ok {
				t.Fatalf("a row hovers at %d in the glide", top)
			}
		}
	}
	if !moved {
		t.Fatal("the wheel never glided")
	}
	tt.Advance(time.Millisecond)
	if top, bottom, ok := lit(); !ok || top > 300 || bottom < 300 {
		t.Errorf("after the glide the hover is at %d..%d (%v)", top, bottom, ok)
	}
}

// The pointer moving on its own shows the hover at once; a hand's tremble
// on the wheel does not.
func TestThePointerMovingEndsTheHold(t *testing.T) {
	tt, lit := hoverList(t, nil)
	tt.Scroll(300, 300, 0, 90)
	tt.Move(301, 301)
	if _, _, ok := lit(); ok {
		t.Error("a tremble showed the hover")
	}
	tt.Move(300, 310)
	if top, bottom, ok := lit(); !ok || top > 310 || bottom < 310 {
		t.Errorf("moved, the hover is at %d..%d (%v)", top, bottom, ok)
	}
}

// RowPressed tells a cell its row is pressed, in a frame of its own with no
// frame asked for, until the release or the pointer leaves the row; the
// press still goes to the row, which chooses.
func TestRowPressedTellsTheCells(t *testing.T) {
	pressed := map[int]bool{}
	tt, _ := hoverList(t, func(row int, cell *node) {
		if cell.RowPressed() {
			pressed[row] = true
		} else {
			delete(pressed, row)
		}
	})
	r, ok := tt.Find("row 10")
	if !ok {
		t.Fatal("no row 10")
	}
	x, y := float32(r.X+5), float32(r.Y+r.H/2)
	tt.Press(x, y)
	if !pressed[10] || len(pressed) != 1 {
		t.Errorf("pressed: %v", pressed)
	}
	tt.Move(x, y+100)
	if pressed[10] {
		t.Error("still pressed, off the row")
	}
	tt.Move(x, y)
	if !pressed[10] {
		t.Error("not pressed, back on the row")
	}
	tt.Release(x, y)
	if pressed[10] {
		t.Error("still pressed, released")
	}
}

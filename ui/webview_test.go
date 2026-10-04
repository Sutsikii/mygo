package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/surface"
)

type testView struct{ name string }

func (*testView) PlaceView(*surface.Conn, platform.RectF, bool) {}

func TestWebViewPlacement(t *testing.T) {
	a, b, away := &testView{"a"}, &testView{"b"}, &testView{"away"}
	tab := 0
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Children(func() {
			Box(c).Height(30)
			if tab == 0 {
				WebView(c, a).Grow(1)
			} else {
				WebView(c, b).Grow(1)
			}
		})
		// Out of the window: not shown.
		WebView(c, away).Absolute().Left(10).Top(500).Size(10, 10)
	}, 200, 100)
	views := tt.h.views
	if len(views) != 1 || views[a] != (Rect{0, 30, 200, 70}) {
		t.Fatalf("shown: %v", views)
	}

	tab = 1
	tt.Frame()
	if len(views) != 1 || views[b] != (Rect{0, 30, 200, 70}) {
		t.Fatalf("after switching tabs: %v", tt.h.views)
	}

	// The window growing moves it.
	tt.h.w = 300
	tt.Frame()
	if views[b] != (Rect{0, 30, 300, 70}) {
		t.Errorf("after resizing: %v", views)
	}
}

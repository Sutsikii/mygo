package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/surface"
)

// NativeView is a view of the system that MyGo shows over the native UI it
// draws, as a *mygo.WebView.
type NativeView = surface.View

// WebView shows a web view, made with mygo.Window.NewWebView, in the
// element's box. A web view has no size of its own: the element stretches
// across its container (AlignSelf(Stretch)), and takes the room along it
// with Grow or a size:
//
//	ui.Row(c).Fill().Children(func() {
//		sidebar(c)
//		ui.WebView(c, app.docs).Grow(1)
//	})
//
// The web view shows while a frame builds its element, and hides when one
// does not, keeping its page: a view switching tabs builds the web view of
// the tab shown. It is the system's own view, above what MyGo draws, so
// elements over its box, as popovers and dialogs, show under it, and a
// scroll container does not clip it: it shows whole as long as part of its
// box is in view. It takes the pointer and the keyboard over its box itself.
func WebView(c *Context, v NativeView) *Element {
	e := Box(c).AlignSelf(Stretch)
	e.view = v
	return e
}

// placedView is a native view a frame shows, and where.
type placedView struct {
	v NativeView
	r Rect
}

// noteView notes where the frame shows the native view of e, unless it is
// out of sight. Called while committing a frame.
func (rt *engine) noteView(e *Element, visible Rect) {
	if visible.W <= 0 || visible.H <= 0 {
		return
	}
	rt.views = append(rt.views, placedView{e.view, Rect{e.x, e.y, e.w, e.h}})
}

// placeViews tells the native views where the frame shows them, and hides
// those it no longer shows.
func (rt *engine) placeViews() {
	if len(rt.views) == 0 && len(rt.shown) == 0 {
		return
	}
	if rt.shown == nil {
		rt.shown = map[NativeView]Rect{}
	}
	seen := make(map[NativeView]bool, len(rt.views))
	for _, p := range rt.views {
		seen[p.v] = true
		if r, ok := rt.shown[p.v]; ok && r == p.r {
			continue
		}
		rt.shown[p.v] = p.r
		rt.host.placeView(p.v, p.r, true)
	}
	for v := range rt.shown {
		if !seen[v] {
			delete(rt.shown, v)
			rt.host.placeView(v, Rect{}, false)
		}
	}
	rt.views = rt.views[:0]
}

func (h *windowHost) placeView(v NativeView, r Rect, visible bool) {
	v.PlaceView(h.conn, platform.RectF{X: float64(r.X), Y: float64(r.Y), W: float64(r.W), H: float64(r.H)}, visible)
}

//go:build linux && (amd64 || arm64)

package linux

import (
	"math"

	"github.com/egoist/mygo/internal/platform"
)

// A web view over a surface (platform.Surface.NewWebView) is a window of
// this package without a GtkWindow of its own: win is its host's, and its
// WebKitWebView is a child of the overlay around the surface's area, at
// the start of both axes, the margins placing it. Its signals find it by
// its id in webViews, apart from the windows.

func (s *surface) NewWebView(o *platform.WindowOptions, h platform.WindowHandler) (platform.WebView, error) {
	if err := webKit(); err != nil {
		return nil, err
	}
	host := s.w
	b := host.b
	b.nextID++
	v := &window{b: b, id: b.nextID, h: h, opts: o, win: host.win, host: host}
	v.createWebView()
	gtkWidgetSetHalign(v.web, 1) // GTK_ALIGN_START
	gtkWidgetSetValign(v.web, 1)
	gtkWidgetSetNoShowAll(v.web, true) // hidden until placed
	gtkOverlayAddOverlay(s.overlay, v.web)
	b.webViews[v.id] = v
	b.byWebView[v.web] = v
	host.webViews = append(host.webViews, v)
	return v, nil
}

// top returns the window that shows w: its host for a web view.
func (w *window) top() *window {
	if w.host != nil {
		return w.host
	}
	return w
}

// SetFrame shows the web view where the native UI places it.
func (w *window) SetFrame(r platform.RectF, visible bool) {
	if w.closed {
		return
	}
	if !visible {
		if gtkWidgetHasFocus(w.web) {
			// A hidden widget keeps no keyboard: it goes back to the
			// native UI.
			gtkWidgetGrabFocus(w.host.surface.area)
		}
		gtkWidgetHide(w.web)
		return
	}
	x, y := int32(math.Round(r.X)), int32(math.Round(r.Y))
	gtkWidgetSetMarginStart(w.web, x)
	gtkWidgetSetMarginTop(w.web, y)
	gtkWidgetSetSizeRequest(w.web, int32(math.Round(r.X+r.W))-x, int32(math.Round(r.Y+r.H))-y)
	gtkWidgetShow(w.web)
}

// closeWebView closes a web view: by Close, or as its window closes.
func (w *window) closeWebView() {
	if w.closed {
		return
	}
	w.closed = true
	b := w.b
	delete(b.webViews, w.id)
	delete(b.byWebView, w.web)
	if gtkWidgetHasFocus(w.web) && w.host.surface != nil {
		gtkWidgetGrabFocus(w.host.surface.area)
	}
	webkitUserContentManagerUnregisterHandler(w.ucm, cs("mygo"))
	webkitUserContentManagerRemoveAllScripts(w.ucm)
	gtkWidgetDestroy(w.web)
	gObjectUnref(w.ucm)
	if w.press.event != 0 {
		gdkEventFree(w.press.event)
		w.press.event = 0
	}
	h := w.host
	for i, v := range h.webViews {
		if v == w {
			h.webViews = append(h.webViews[:i:i], h.webViews[i+1:]...)
			break
		}
	}
}

// closeWebViews closes the web views of a window that closes.
func (w *window) closeWebViews() {
	for len(w.webViews) > 0 {
		w.webViews[len(w.webViews)-1].closeWebView()
	}
}

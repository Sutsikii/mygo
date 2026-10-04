//go:build windows && (amd64 || arm64)

package windows

import (
	"math"

	"github.com/egoist/mygo/internal/platform"
)

// A web view over a surface (platform.Surface.NewWebView) is a window of
// this package without a window of its own: its hwnd is the surface's
// window, whose child its WebView2 controller is, below the controls of a
// hidden title bar, and host is the window of the surface. The page code
// of webview.go serves both; the methods below tell them apart.

func (s *surface) NewWebView(o *platform.WindowOptions, h platform.WindowHandler) (platform.WebView, error) {
	b := s.w.b
	if err := b.startEnvironment(); err != nil {
		return nil, err
	}
	v := &window{
		b: b, h: h, opts: o, hwnd: s.hwnd, host: s.w,
		opacity: 1, zoom: 1, bg: o.BackgroundColor,
		htmlFor: map[string]string{}, calls: map[int]func(string, error){},
	}
	s.w.webViews = append(s.w.webViews, v)
	b.whenEnvironment(v.createWebView)
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
	w.frame, w.shown = r, visible
	w.withWebView(w.placeWebView)
}

// placeWebView puts the controller of a web view at its frame, in the
// pixels of the surface's window, which change with its DPI.
func (w *window) placeWebView() {
	if w.shown {
		s := float64(dpiOf(w.host.hwnd)) / 96
		r := w.frame
		px := func(v float64) int32 { return int32(math.Round(v * s)) }
		putBounds(w.controller, rect{px(r.X), px(r.Y), px(r.X + r.W), px(r.Y + r.H)})
	}
	comCall(w.controller, ctlPutIsVisible, boolArg(w.shown))
}

// placeWebViews places the web views of a window again, after its DPI or
// size changed.
func (w *window) placeWebViews() {
	for _, v := range w.webViews {
		v.withWebView(v.placeWebView)
	}
}

// closeWebView closes a web view: by Close, or as its window closes.
func (w *window) closeWebView() {
	if w.closed {
		return
	}
	w.closed = true
	for id, cb := range w.calls {
		delete(w.calls, id)
		cb("", errDestroyed)
	}
	w.pending = nil
	if w.controller != 0 {
		comCall(w.controller, ctlClose)
		release(w.settings)
		release(w.webview)
		release(w.controller)
		w.controller, w.webview, w.settings = 0, 0, 0
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

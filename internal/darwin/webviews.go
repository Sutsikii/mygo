//go:build darwin

package darwin

import "github.com/egoist/mygo/internal/platform"

// A web view over a surface (platform.Surface.NewWebView) is a window of
// this package without an NSWindow of its own: win is its host's, which
// its sheets and print panels use, and its MyGoWebView is a subview of the
// host's content view, above the surface. Its delegate is the web view's
// alone, never the NSWindow's, so it hears only of its page.

func (s *surface) NewWebView(o *platform.WindowOptions, h platform.WindowHandler) (platform.WebView, error) {
	host := s.w
	b := host.b
	v := &window{b: b, h: h, opts: o, win: host.win, host: host, downloads: map[id][2]string{}}
	withPool(func() {
		v.delegate = alloc("MyGoWindowDelegate")
		v.createWebView(NSRect{})
		// AppKit's origin is at the bottom: keep the web view's distance to
		// the top as the window resizes, until the view places it again.
		send(v.web, "setAutoresizingMask:", nsViewMinYMargin)
		send(v.web, "setHidden:", 1)
		send(host.view, "addSubview:", uintptr(v.web))
	})
	b.byDelegate[v.delegate] = v
	b.byWebView[v.web] = v
	host.webViews = append(host.webViews, v)
	return v, nil
}

// nsViewMinYMargin is NSViewMinYMargin: the margin below a view grows.
const nsViewMinYMargin = 8

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
	if visible {
		height := msgRect(w.host.view, sel("bounds")).Size.Height
		msgSetRect(w.web, sel("setFrame:"), NSRect{Origin: NSPoint{r.X, height - r.Y - r.H}, Size: NSSize{r.W, r.H}})
	} else if w.hasKeyboard() {
		// A hidden view keeps no keyboard: it goes back to the native UI.
		send(w.win, "makeFirstResponder:", uintptr(w.host.surface.view))
	}
	send(w.web, "setHidden:", boolArg(!visible))
}

// hasKeyboard reports whether the web view, or a view in it, is the first
// responder of its window.
func (w *window) hasKeyboard() bool {
	r := send(w.win, "firstResponder")
	return r != 0 && sendBool(r, "isKindOfClass:", uintptr(class("NSView"))) &&
		sendBool(r, "isDescendantOf:", uintptr(w.web))
}

// closeWebView closes a web view: by Close, or as its window closes.
func (w *window) closeWebView() {
	if w.closed {
		return
	}
	w.closed = true
	b := w.b
	withPool(func() {
		if w.hasKeyboard() && w.host.surface != nil {
			send(w.win, "makeFirstResponder:", uintptr(w.host.surface.view))
		}
		send(w.web, "removeObserver:forKeyPath:", uintptr(w.delegate), uintptr(nsString("title")))
		send(w.ucc, "removeScriptMessageHandlerForName:", uintptr(nsString("mygo")))
		send(w.ucc, "removeAllUserScripts")
		send(w.web, "stopLoading")
		send(w.web, "setNavigationDelegate:", 0)
		send(w.web, "setUIDelegate:", 0)
		send(w.web, "removeFromSuperview")
		delete(b.byDelegate, w.delegate)
		delete(b.byWebView, w.web)
		release(w.ucc)
		release(w.web)
		// WebKit may still hold the delegate as the page goes.
		autorelease(w.delegate)
	})
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

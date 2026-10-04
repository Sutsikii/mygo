package mygo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

type callerService struct{ page atomic.Pointer[Page] }

// Who tells the window of the calling page, and keeps the page.
func (s *callerService) Who(ctx context.Context) int {
	s.page.Store(CallerPage(ctx))
	return CallerWindow(ctx).ID()
}

// webViewWindow creates a window whose view shows a web view while *show is
// true, and returns them with the fake web view.
func webViewWindow(t *testing.T, show *bool) (*Window, *fake.Window, *WebView, *fake.Window) {
	t.Helper()
	var wv *WebView
	w, fw, s := contentWindow(t, func(c *ui.Context) {
		if *show {
			ui.WebView(c, wv).Absolute().Left(10).Top(20).Size(100, 50)
		}
	})
	wv, err := w.NewWebView(WebViewOptions{URL: "/docs", BackgroundColor: "#123456"})
	if err != nil {
		t.Fatal(err)
	}
	views := fw.WebViews()
	if len(views) != 1 {
		t.Fatalf("%d web views", len(views))
	}
	fv := views[0]
	w.Invalidate()
	onMain(func() { s.Frame() })
	return w, fw, wv, fv
}

func TestWebViewShowsWhereTheViewPlacesIt(t *testing.T) {
	show := true
	w, fw, wv, fv := webViewWindow(t, &show)
	if !strings.HasSuffix(fv.URL(), "/docs") {
		t.Errorf("the web view loaded %q", fv.URL())
	}
	if c := fv.Opts.BackgroundColor; c == nil || *c != (platform.Color{R: 0x12, G: 0x34, B: 0x56, A: 255}) {
		t.Errorf("background %v", c)
	}
	if !strings.Contains(fv.Opts.UserScripts[0].Source, fmt.Sprintf(`"windowId":%d`, w.ID())) {
		t.Error("the page of the web view does not have the id of its window")
	}
	if !fv.Shown || fv.Frame != (platform.RectF{X: 10, Y: 20, W: 100, H: 50}) {
		t.Errorf("shown %v at %v", fv.Shown, fv.Frame)
	}

	show = false
	w.Invalidate()
	onMain(func() { fw.FakeSurface().Frame() })
	if fv.Shown {
		t.Error("the web view shows while the view does not build it")
	}
	if fv.IsClosed() || wv.IsDestroyed() {
		t.Error("hiding the web view closed it")
	}

	if wv.Window() != w || wv.Page().Window() != w || wv.Page().WebView() == nil || w.Page() != nil {
		t.Error("the web view's page is not in its window")
	}
	wv.Focus()
	if fv.Focused != 1 {
		t.Errorf("focused %d times", fv.Focused)
	}
}

func TestWebViewPageCallsGo(t *testing.T) {
	who := &callerService{}
	bindForTest(t, "Who", who)
	ev := newEventForTest[progress](t, "test:webview")
	show := true
	w, _, wv, fv := webViewWindow(t, &show)
	onMain(func() { fv.H.NavigationCommitted("mygo://localhost/docs") })
	page(fv, `{"t":"dom-ready"}`)

	m := call(t, fv, 1, "Who.Who")
	if m["v"] != float64(w.ID()) {
		t.Errorf("CallerWindow %v, want %d", m["v"], w.ID())
	}
	if who.page.Load() != wv.Page() {
		t.Error("CallerPage is not the web view's page")
	}
	if err := ev.EmitPage(wv.Page(), progress{1, 2}); err != nil {
		t.Fatal(err)
	}
	received(t, fv, func(m map[string]any) bool { return m["t"] == "event" && m["n"] == "test:webview" })
	if err := ev.Broadcast(progress{2, 2}); err != nil {
		t.Fatal(err)
	}
	received(t, fv, func(m map[string]any) bool {
		p, _ := m["p"].(map[string]any)
		return m["n"] == "test:webview" && p["done"] == float64(2)
	})

	// window.close() of the web view's page leaves the window open.
	onMain(fv.H.ClosedByPage)
	if w.IsDestroyed() || wv.IsDestroyed() {
		t.Error("the page of a web view closed something")
	}
}

func TestWebViewDestroy(t *testing.T) {
	show := true
	w, fw, wv, fv := webViewWindow(t, &show)
	wv.Destroy()
	if !fv.IsClosed() || !wv.IsDestroyed() {
		t.Fatal("Destroy left the web view")
	}
	if _, err := wv.Page().Eval("1"); !errors.Is(err, errDestroyed) {
		t.Errorf("Eval on a destroyed web view: %v", err)
	}
	// The view placing it still is harmless.
	w.Invalidate()
	onMain(func() { fw.FakeSurface().Frame() })

	other, err := w.NewWebView(WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	fo := fw.WebViews()[1]
	w.Destroy()
	if !fo.IsClosed() || !other.IsDestroyed() {
		t.Error("the web view outlived its window")
	}
	if _, err := w.NewWebView(WebViewOptions{}); !errors.Is(err, errDestroyed) {
		t.Errorf("NewWebView on a closed window: %v", err)
	}
}

func TestWebViewNeedsContent(t *testing.T) {
	w, _ := testWindow(t, WindowOptions{})
	if _, err := w.NewWebView(WebViewOptions{}); !errors.Is(err, errNoContent) {
		t.Errorf("NewWebView in a window showing a page: %v", err)
	}
	cw, _, _ := contentWindow(t, func(c *ui.Context) {})
	if _, err := cw.NewWebView(WebViewOptions{BackgroundColor: "nope"}); err == nil {
		t.Error("NewWebView took a bad background color")
	}
}

# Web view

`ui.WebView` shows a web page inside native UI: a sidebar MyGo draws beside
a document the web engine renders, a page per tab, a preview of HTML the
app makes. Its page is a page as a window's is: it loads the app's
frontend, calls bound Go methods, hears events and runs scripts.

Create the web view once the window exists, with `Window.NewWebView`, and
place it in the view:

```go
win := mygo.NewWindow(mygo.WindowOptions{Title: "Docs", Content: ui.View(app.view)})
docs, err := win.NewWebView(mygo.WebViewOptions{URL: "/docs"})
if err != nil {
	log.Fatal(err)
}
app.docs = docs
win.Invalidate()
```

```go
ui.Row(c).Fill().Children(func() {
	app.sidebar(c)
	ui.WebView(c, app.docs).Grow(1)
})
```

A web view has no size of its own: its element stretches across its
container, and takes the room along it with `Grow` or a size. `URL` takes
what `Page.LoadURL` takes: `"/docs"` is a page of the app's frontend, as
in a window, and URLs of schemes the app handles (`Protocol.Handle`), of
the web and of files work too. `WebViewOptions.Page` holds the options of
its page, as `WindowOptions.Page` does, and `BackgroundColor` fills it
until its page paints.

## Showing and hiding

The web view shows while the view builds its element, and hides when a
frame does not, keeping its page. Tabs build the web view of the tab
shown, and the others keep where they were:

```go
ui.WebView(c, app.tabs[app.current].view).Grow(1)
```

`Destroy` closes a web view and its page; one that the view still places
shows nothing. A window's web views close with it.

## Its page

`WebView.Page` returns its `*mygo.Page`, whose methods are those of a
window's page: `LoadURL`, `Eval`, `OnDidNavigate`, `OpenDevTools`, …

Bound methods run for its page as for any other. `CallerWindow` returns
the window it shows in, and `CallerPage` the page that called, which tells
a web view's page from the others:

```go
func (Notes) Save(ctx context.Context, text string) {
	if p := mygo.CallerPage(ctx); p.WebView() != nil {
		// Called from a web view.
	}
}
```

`Event.EmitPage` sends an event to one page, and `Event.Broadcast` sends
it to every window and every web view. In the page, `mygo.windowId` is the
id of the window, and the window controls of mygo-runtime act on the
window: a drag region of a frameless window's web view moves the window.

## Above what MyGo draws

A web view is the system's own view, laid over the window above what MyGo
draws:

- Elements over its box, such as popovers, menus of the toolkit and
  dialogs, show under it. Keep them clear of it, or hide the web view while
  they show.
- A scroll container does not clip it. It shows whole as long as part of
  its box is in view, and hides when none is.
- It takes the pointer and the keyboard over its box itself. `Focus` gives
  it the keyboard; a click on native UI takes the keyboard back.
- `Window.CapturePage` captures what MyGo draws, without the web views;
  `WebView.CapturePage` captures a web view's page.

It is WKWebView on macOS, WebKitGTK on Linux and WebView2 on Windows, as in
windows showing a web page: the first web view starts the web engine, which
an app whose windows show native UI only never loads.

## Accessibility

Assistive technology reads the page as it reads any web page, from the web
engine.

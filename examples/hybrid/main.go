// Hybrid shows web pages inside native UI: a sidebar MyGo draws chooses
// among web views, whose pages call Go and hear from it, and the native
// UI follows what they do.
//
//	go run ./examples/hybrid
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// app is the state the window shows. The view reads and changes it on the
// main thread.
type app struct {
	win      *mygo.Window
	selected string
	views    map[string]*mygo.WebView
	likes    int
	words    int
}

// liked tells the pages how many likes there are.
var liked = mygo.NewEvent[int]("liked")

// Pages is what the pages call.
type Pages struct{ a *app }

// Like adds a like and returns how many there are.
func (p Pages) Like() int {
	var n int
	mygo.RunOnMain(func() {
		p.a.likes++
		n = p.a.likes
	})
	p.a.win.Invalidate()
	liked.Broadcast(n)
	return n
}

// Likes returns how many likes there are.
func (p Pages) Likes() int {
	var n int
	mygo.RunOnMain(func() { n = p.a.likes })
	return n
}

// SetNotes tells the native UI what the notes say.
func (p Pages) SetNotes(ctx context.Context, text string) {
	p.a.win.Update(func() { p.a.words = len(strings.Fields(text)) })
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Children(func() {
		ui.Column(c).Width(220).FillHeight().Background(t.Surface).Children(func() {
			ui.Sidebar(c, &a.selected, func() {
				ui.SidebarItem(c, "welcome", nil, "Welcome")
				ui.SidebarItem(c, "notes", nil, "Notes").Children(func() {
					if a.words > 0 {
						ui.Badge(c, fmt.Sprint(a.words))
					}
				})
			}).Grow(1)
			ui.Column(c).Padding(t.Space(3)).Gap(t.Space(2)).Children(func() {
				ui.Textf(c, "%d likes", a.likes).TextColor(t.TextMuted)
				if ui.Button(c, "Like from Go").Clicked() {
					a.likes++
					liked.Broadcast(a.likes)
				}
			})
		})
		// The web view of the item chosen; the other keeps its page.
		ui.WebView(c, a.views[a.selected]).Grow(1)
	})
}

const style = `<style>
:root { color-scheme: light dark; font: 15px system-ui, sans-serif }
body { margin: 0; padding: 24px 32px }
button { font: inherit; padding: 6px 14px }
textarea { width: 100%; height: 60vh; font: inherit; box-sizing: border-box }
</style>`

var pages = map[string]string{
	"welcome": `<!doctype html><title>Welcome</title>` + style + `
<h1>Web pages in native UI</h1>
<p>This page is a web view in a window MyGo draws: the sidebar on the left
is native UI, written in Go.</p>
<p><button id="like">Like from the page</button> <span id="count"></span></p>
<script>
const count = document.getElementById("count");
const show = (n) => (count.textContent = n + " likes");
mygo.call("Pages.Likes").then(show);
mygo.on("liked", show);
document.getElementById("like").onclick = () => mygo.call("Pages.Like").then(show);
</script>`,
	"notes": `<!doctype html><title>Notes</title>` + style + `
<h1>Notes</h1>
<p>The sidebar counts the words you type here.</p>
<textarea id="notes" placeholder="Type something"></textarea>
<script>
const notes = document.getElementById("notes");
notes.oninput = () => mygo.call("Pages.SetNotes", notes.value);
</script>`,
}

func main() {
	a := &app{selected: "welcome", views: map[string]*mygo.WebView{}}
	mygo.Bind(Pages{a})
	// The pages come from Go, at hybrid://localhost/<name>.
	mygo.Protocol.HandleFunc("hybrid", func(w http.ResponseWriter, r *http.Request) {
		page, ok := pages[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, page)
	})
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Hybrid",
			Width:   900,
			Height:  600,
			Content: ui.View(a.view),
		})
		for name := range pages {
			v, err := a.win.NewWebView(mygo.WebViewOptions{URL: "hybrid://localhost/" + name})
			if err != nil {
				log.Fatal(err)
			}
			a.views[name] = v
		}
		a.win.Invalidate()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

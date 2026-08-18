// Package htmlx provides small DOM helpers over golang.org/x/net/html,
// mirroring the JSDOM querySelector usage in the original scrapers.
package htmlx

import (
	"strings"

	"golang.org/x/net/html"
)

// Parse parses HTML into a node tree.
func Parse(doc string) (*html.Node, error) {
	return html.Parse(strings.NewReader(doc))
}

// Filter returns all nodes matching the predicate, in document order.
func Filter(n *html.Node, pred func(*html.Node) bool) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if pred(n) {
			out = append(out, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

// Tag returns all elements with the given tag name (lowercase).
func Tag(n *html.Node, tag string) []*html.Node {
	return Filter(n, func(x *html.Node) bool { return x.Type == html.ElementNode && x.Data == tag })
}

// Classes returns all elements having all the given classes.
func Classes(n *html.Node, classes ...string) []*html.Node {
	return Filter(n, func(x *html.Node) bool {
		if x.Type != html.ElementNode {
			return false
		}
		cur := classSet(x)
		for _, c := range classes {
			if !cur[c] {
				return false
			}
		}
		return true
	})
}

// ID returns all elements with the given id.
func ID(n *html.Node, id string) []*html.Node {
	return Filter(n, func(x *html.Node) bool { return x.Type == html.ElementNode && attr(x, "id") == id })
}

// Attr returns an element's attribute value (or "").
func Attr(n *html.Node, key string) string { return attr(n, key) }

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// Text returns the concatenated text content of a node.
func Text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// InnerHTML returns the serialized inner HTML of a node.
func InnerHTML(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(Render(c))
	}
	return b.String()
}

// Render serializes a node to HTML.
func Render(n *html.Node) string {
	var b strings.Builder
	renderNode(&b, n)
	return b.String()
}

func renderNode(b *strings.Builder, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		b.WriteString(n.Data)
	case html.ElementNode:
		switch n.Data {
		case "br", "hr", "img", "input", "meta", "link":
			b.WriteString("<" + n.Data)
			for _, a := range n.Attr {
				if a.Namespace == "" {
					b.WriteString(" " + a.Key + `="` + a.Val + `"`)
				}
			}
			b.WriteString(">")
			return
		}
		b.WriteString("<" + n.Data)
		for _, a := range n.Attr {
			if a.Namespace == "" {
				b.WriteString(" " + a.Key + `="` + a.Val + `"`)
			}
		}
		b.WriteString(">")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderNode(b, c)
		}
		b.WriteString("</" + n.Data + ">")
	case html.CommentNode:
		b.WriteString("<!--" + n.Data + "-->")
	}
}

// First returns the first matching node or nil.
func First(n *html.Node, pred func(*html.Node) bool) *html.Node {
	for _, m := range Filter(n, pred) {
		return m
	}
	return nil
}

// Children returns element children of n.
func Children(n *html.Node) []*html.Node {
	var out []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			out = append(out, c)
		}
	}
	return out
}

// classSet returns the class set of a node.
func classSet(n *html.Node) map[string]bool {
	out := map[string]bool{}
	for _, a := range n.Attr {
		if a.Key == "class" || a.Key == "className" {
			for _, c := range strings.Fields(a.Val) {
				out[c] = true
			}
		}
	}
	return out
}

// HasClass reports whether a node has the class.
func HasClass(n *html.Node, class string) bool {
	return classSet(n)[class]
}
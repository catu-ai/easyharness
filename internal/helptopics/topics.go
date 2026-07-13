package helptopics

import (
	"fmt"
	"io"
	"strings"

	helpassets "github.com/catu-ai/easyharness/assets/help"
)

type Topic struct {
	Path    []string
	Summary string
	Asset   string
}

var registry = []Topic{
	{
		Path:    []string{"review"},
		Summary: "Integrated finalize review, advisors, and repair deltas.",
		Asset:   "review.md",
	},
	{
		Path:    []string{"repo"},
		Summary: "Repository resources and customization guidance.",
		Asset:   "repo.md",
	},
	{
		Path:    []string{"repo", "config"},
		Summary: "Customize .harness/config.yaml and repo-defined harness paths.",
		Asset:   "repo/config.md",
	},
}

type node struct {
	topic    *Topic
	children map[string]*node
}

type UnknownTopicError struct {
	Path         []string
	Nearest      []string
	Subtopics    []Topic
	RootFallback bool
}

func (e UnknownTopicError) Error() string {
	return fmt.Sprintf("unknown help topic %q", strings.Join(e.Path, " "))
}

func Render(w io.Writer, path []string) error {
	root := buildTree()
	current := root
	var nearest []string
	for i, segment := range path {
		next := current.children[segment]
		if next == nil {
			subtopics := childTopics(current)
			rootFallback := false
			if len(subtopics) == 0 {
				subtopics = childTopics(root)
				rootFallback = true
			}
			return UnknownTopicError{
				Path:         path,
				Nearest:      nearest,
				Subtopics:    subtopics,
				RootFallback: rootFallback,
			}
		}
		current = next
		nearest = path[:i+1]
	}

	if len(path) == 0 {
		fmt.Fprintln(w, "Harness Help")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Use `harness help <topic>` to read agent-facing product guidance.")
		fmt.Fprintln(w)
		writeTopicList(w, "Available topics:", childTopics(current))
		return nil
	}

	if current.topic != nil && strings.TrimSpace(current.topic.Asset) != "" {
		body, err := helpassets.Read(current.topic.Asset)
		if err != nil {
			return fmt.Errorf("read help topic %q: %w", strings.Join(path, " "), err)
		}
		fmt.Fprint(w, body)
	}
	children := childTopics(current)
	if len(children) > 0 {
		fmt.Fprintln(w)
		writeTopicList(w, "Available subtopics:", children)
	}
	return nil
}

func WriteUnknown(w io.Writer, err UnknownTopicError) {
	fmt.Fprintf(w, "%s\n\n", err.Error())
	if len(err.Subtopics) == 0 {
		return
	}
	if len(err.Nearest) == 0 || err.RootFallback {
		writeTopicList(w, "Available topics:", err.Subtopics)
		return
	}
	writeTopicList(w, "Available subtopics:", err.Subtopics)
}

func buildTree() *node {
	root := &node{children: map[string]*node{}}
	for i := range registry {
		topic := &registry[i]
		current := root
		for _, segment := range topic.Path {
			if current.children == nil {
				current.children = map[string]*node{}
			}
			if current.children[segment] == nil {
				current.children[segment] = &node{children: map[string]*node{}}
			}
			current = current.children[segment]
		}
		current.topic = topic
	}
	return root
}

func childTopics(parent *node) []Topic {
	if parent == nil || len(parent.children) == 0 {
		return nil
	}
	topics := make([]Topic, 0, len(parent.children))
	for name, child := range parent.children {
		if child.topic != nil {
			topics = append(topics, *child.topic)
			continue
		}
		topics = append(topics, Topic{Path: []string{name}})
	}
	sortTopics(topics)
	return topics
}

func sortTopics(topics []Topic) {
	for i := 1; i < len(topics); i++ {
		for j := i; j > 0 && topicKey(topics[j]) < topicKey(topics[j-1]); j-- {
			topics[j], topics[j-1] = topics[j-1], topics[j]
		}
	}
}

func topicKey(topic Topic) string {
	return strings.Join(topic.Path, " ")
}

func writeTopicList(w io.Writer, title string, topics []Topic) {
	if len(topics) == 0 {
		return
	}
	fmt.Fprintln(w, title)
	for _, topic := range topics {
		name := topic.Path[len(topic.Path)-1]
		fmt.Fprintf(w, "  %-8s %s\n", name, topic.Summary)
	}
}

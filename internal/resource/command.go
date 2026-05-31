package resource

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Command is the wire shape served by GET /command (openapi Command). Template
// and Hints are required by the spec; Hints is always a non-nil array.
type Command struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Agent       string   `json:"agent,omitempty"`
	Model       string   `json:"model,omitempty"`
	Source      string   `json:"source"`
	Template    string   `json:"template"`
	Subtask     bool     `json:"subtask,omitempty"`
	Hints       []string `json:"hints"`
}

// commandFrontmatter is the YAML frontmatter of an .opencode/command/*.md file
// (config/command.ts:14-20). The markdown body becomes the template.
type commandFrontmatter struct {
	Description string `yaml:"description"`
	Agent       string `yaml:"agent"`
	Model       string `yaml:"model"`
	Subtask     bool   `yaml:"subtask"`
}

// LoadCommands returns the commands for dir: the config `command` map (lowest
// priority) overlaid with .opencode/command(s)/**/*.md files (nearest dir wins),
// sorted by name. source is "command" for all (MCP/skill commands are out of
// scope for this slice).
func LoadCommands(dir string, cfg map[string]any) []Command {
	commands := map[string]*Command{}
	mergeConfigCommands(commands, cfg)
	for _, cd := range ConfigDirs(dir) {
		for name, c := range loadCommandDir(cd) {
			c.Name = name
			commands[name] = c
		}
	}
	out := make([]Command, 0, len(commands))
	for _, c := range commands {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func loadCommandDir(configDir string) map[string]*Command {
	out := map[string]*Command{}
	for _, file := range globMarkdown(configDir, []string{"command", "commands"}) {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(configDir, file)
		name := entryName(rel, []string{"command/", "commands/"})
		out[name] = parseCommand(data)
	}
	return out
}

func parseCommand(data []byte) *Command {
	yamlBytes, body := splitFrontmatter(data)
	var fm commandFrontmatter
	if len(yamlBytes) > 0 {
		_ = yaml.Unmarshal(yamlBytes, &fm)
	}
	return &Command{
		Description: fm.Description, Agent: fm.Agent, Model: fm.Model,
		Subtask: fm.Subtask, Source: "command", Template: body, Hints: []string{},
	}
}

// mergeConfigCommands overlays the config `command` map (lowest priority).
func mergeConfigCommands(commands map[string]*Command, cfg map[string]any) {
	raw, ok := cfg["command"].(map[string]any)
	if !ok {
		return
	}
	for name, v := range raw {
		entry, ok := v.(map[string]any)
		if !ok {
			continue
		}
		c := &Command{Name: name, Source: "command", Hints: []string{}}
		if t, ok := entry["template"].(string); ok {
			c.Template = t
		}
		if d, ok := entry["description"].(string); ok {
			c.Description = d
		}
		if a, ok := entry["agent"].(string); ok {
			c.Agent = a
		}
		if m, ok := entry["model"].(string); ok {
			c.Model = m
		}
		if s, ok := entry["subtask"].(bool); ok {
			c.Subtask = s
		}
		commands[name] = c
	}
}

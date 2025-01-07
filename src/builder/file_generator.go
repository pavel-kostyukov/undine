package builder

import (
	"fmt"
	"os"
	"strings"

	"github.com/undine-project/undine/src/config"
)

type FileGenerator struct {
	templatePath string
	resultPath   string
	data         map[string]string
	tabsHtml     string
	graphsHtml   string
}

func NewFileGenerator(templatePath string, resultPath string, devMode bool, fileDefs []config.File) *FileGenerator {
	var tabs, graphs string
	for _, fileDef := range fileDefs {
		if fileDef.Name != "template" {
			tabs += fmt.Sprintf("<li><a href=\"#%s\">%s</a></li>\n", fileDef.Name, fileDef.Title)
			graphs += fmt.Sprintf("<pre id='mermaid-%s' class='mermaid'>\n{{%s}}\n</pre>", fileDef.Name, fileDef.Name)
		}
	}

	fg := &FileGenerator{
		templatePath: templatePath,
		resultPath:   resultPath,
		data:         make(map[string]string),
		tabsHtml:     tabs,
		graphsHtml:   graphs,
	}

	if devMode {
		fg.data["devMode"] = "true"
	} else {
		fg.data["devMode"] = "false"
	}

	return fg
}

func (fg *FileGenerator) SetContent(content FileContent) {
	fg.data[content.Name] = content.Content
	if content.Name == "template" {
		fg.enrichTemplate()
	}
}

func (fg *FileGenerator) Generate() error {
	var resultStr string

	if v, ok := fg.data["template"]; !ok || v == "" {
		templateBytes, err := os.ReadFile(fg.templatePath)
		if err != nil {
			return err
		}

		fg.data["template"] = string(templateBytes)
		fg.enrichTemplate()
	}

	resultStr = fg.data["template"]

	for name, content := range fg.data {
		resultStr = strings.Replace(resultStr, "{{"+name+"}}", content, 1)
	}

	err := os.WriteFile(fg.resultPath, []byte(resultStr), 0644)
	if err != nil {
		return err
	}

	fmt.Println("index.html generated successfully")
	return nil
}

func (fg *FileGenerator) enrichTemplate() {
	fg.data["template"] = strings.Replace(fg.data["template"], "{{tabs}}", fg.tabsHtml, 1)
	fg.data["template"] = strings.Replace(fg.data["template"], "{{graphs}}", fg.graphsHtml, 1)
}

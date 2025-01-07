package builder

import (
	"bufio"
	"iter"
	"log"
	"os"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/fsnotify/fsnotify"
	"github.com/undine-project/undine/src/config"
)

type SourceProcessor struct {
	files   map[string]config.File
	watcher *fsnotify.Watcher
}

func NewSourceProcessor(filePaths []config.File, watcher *fsnotify.Watcher) *SourceProcessor {
	definitions := make(map[string]config.File, len(filePaths))
	for _, filePath := range filePaths {
		definitions[filePath.Path] = filePath
	}

	return &SourceProcessor{files: definitions, watcher: watcher}
}

func (sp *SourceProcessor) Process() iter.Seq[FileContent] {
	return func(yield func(FileContent) bool) {
		for _, fileDefinition := range sp.files {
			c, _ := sp.extractMermaidContent(fileDefinition.Path)
			content := FileContent{
				Name:    fileDefinition.Name,
				Content: c,
			}

			if !yield(content) {
				return
			}
		}
	}
}

func (sp *SourceProcessor) Watch(contentChannel chan<- FileContent) {
	fileHashes := make(map[string]uint64, len(sp.files))

	for _, fileDefinition := range sp.files {
		err := sp.watcher.Add(fileDefinition.Path)
		if err != nil {
			log.Fatal(err)
		}
	}

	go func() {
		for {
			select {
			case event, ok := <-sp.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					var (
						content string
						err     error
					)

					fileDefinition := sp.files[event.Name]

					if fileDefinition.Name == "template" {
						content, err = sp.getFullContent(fileDefinition.Path)
					} else {
						content, err = sp.extractMermaidContent(event.Name)
					}

					if err == nil {
						hash := sp.getContentHash(content)
						if hash != fileHashes[fileDefinition.Name] {
							fileHashes[fileDefinition.Name] = hash

							contentChannel <- FileContent{
								Name:    fileDefinition.Name,
								Content: content,
							}
						}

					} else {
						log.Println("Error extracting Mermaid content:", err)
					}
				}
			case err, ok := <-sp.watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			}
		}
	}()
}

func (sp *SourceProcessor) Stop() {
	_ = sp.watcher.Close()
}

func (sp *SourceProcessor) extractMermaidContent(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	var mermaidContent strings.Builder
	scanner := bufio.NewScanner(file)
	inMermaidBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "```mermaid") {
			inMermaidBlock = true
			continue
		}
		if strings.Contains(line, "```") && inMermaidBlock {
			break
		}
		if inMermaidBlock {
			mermaidContent.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return mermaidContent.String(), nil
}

func (sp *SourceProcessor) getFullContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (sp *SourceProcessor) getContentHash(content string) uint64 {
	h := xxhash.New()
	_, _ = h.Write([]byte(content))

	return h.Sum64()
}

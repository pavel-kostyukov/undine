package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/undine-project/undine/src/builder"
	"github.com/undine-project/undine/src/config"
)

var watchFlag = flag.Bool("watch", false, "Watch for changes in the diagram.md file")

func main() {
	flag.Parse()
	c := loadConfig()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)

		return
	}

	files := c.Files
	files = append(files, config.File{
		Name:  "template",
		Path:  c.TemplatePath,
		Title: "Template",
	})
	sp := builder.NewSourceProcessor(files, watcher)

	if _, err := os.Stat("public"); os.IsNotExist(err) {
		err := os.Mkdir("public", 0755)
		if err != nil {
			panic(err)
		}
	}

	fg := builder.NewFileGenerator(
		c.TemplatePath,
		"public/index.html",
		*watchFlag,
		files,
	)
	for content := range sp.Process() {
		fg.SetContent(content)
	}
	err = fg.Generate()
	if err != nil {
		log.Fatal(err)

		return
	}

	if *watchFlag {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			sig := <-sigChan
			log.Printf("Received signal %s, exiting...", sig)
			os.Exit(0)
		}()

		go func() {
			contentsChannel := make(chan builder.FileContent, 2)
			sp.Watch(contentsChannel)
			defer sp.Stop()

			wh := &builder.WebHandler{}
			wh.StartServer()

			for content := range contentsChannel {
				fmt.Printf("Generating HTML file with type %s\n", content.Name)
				fg.SetContent(content)
				err := fg.Generate()
				if err != nil {
					log.Fatal(err)

					return
				}

				fmt.Println("Sending content...")
				wh.SendContent(content)
			}
		}()

		// Keep the main goroutine running to prevent the program from exiting
		select {}
	} else {
		fmt.Println("HTML generated without watching.")
	}
}

func loadConfig() *config.Config {
	viper.SetConfigName("docs-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	var cfg config.Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	return &cfg
}

package config

type Config struct {
	TemplatePath string `mapstruct:"templatePath"`
	Files        []File `mapstruct:"files"`
}

type File struct {
	Name  string `mapstruct:"name"`
	Path  string `mapstruct:"path"`
	Title string `mapstruct:"title"`
}

func NewConfig(templatePath string, files []File) *Config {
	return &Config{
		TemplatePath: templatePath,
		Files:        files,
	}
}

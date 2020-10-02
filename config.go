package report_receiver_bot

import (
	"os"
    "log"
    "strings"
    "io/ioutil"
    "path/filepath"
    "gopkg.in/yaml.v2"
    "github.com/mdigger/translit"
)

type Student struct {
	FirstName  string `yaml:"first_name"`
	LastName   string `yaml:"last_name"`
	SecondName string `yaml:"second_name"`
	GroupName  string `yaml:"group_name"`
	UserName   string `yaml:"user_name"`
	WorkDir	string
}

type Admin struct {
	Group  string `yaml:"group"`
	User   string `yaml:"user"`
	ChatID int64  `yaml:"chat_id"`
}

type ReportType struct {
	Format	string `yaml:"format"`
	MaxSizeMb int	`yaml:"max_size_mb"`
	Notify	bool   `yaml:"notify"`
}

type Config struct {
	Student  []Student	`yaml:"student"`
	Admin	[]Admin	  `yaml:"admin"`
	Work	 []string	 `yaml:"work"`
	Report   []ReportType `yaml:"report_type"`
	WorkDir  string	   `yaml:"work_dir"`
	BotToken string	   `yaml:"bot_token"`
}

func ReadConfig(filePath string) (*Config, error) {
	byteYaml, err := ReadFileBytes(filePath)
	if err != nil {
		return nil, err
	}

	conf := Config{}

	err = yaml.Unmarshal(byteYaml, &conf)
	if err != nil {
		return nil, err
	}

	// conf.PrepareStudentWorkDir()

	return &conf, nil
}


func (conf *Config) Save(filePath string) error {
	dump, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filePath, dump, 0755)
	if err != nil {
		return err
	}
	return nil
}

func (conf *Config) PrepareStudentWorkDir() error {
	for i, s := range conf.Student {
		studDir := filepath.Join(
			conf.WorkDir,
			s.GroupName,
			strings.ToLower(translit.Ru(string([]rune(s.FirstName)[0]) + s.LastName)))

		if _, err := os.Stat(studDir); os.IsNotExist(err) {
			log.Printf("[info] Create student directory: %s", studDir)
			err := os.MkdirAll(studDir, 0755)
			if err != nil {
				return err
			}
		}
		conf.Student[i].WorkDir = studDir

		for _, work := range conf.Work {
			studWorkDir := filepath.Join(studDir, work)
			if _, err := os.Stat(studWorkDir); os.IsNotExist(err) {
				err := os.Mkdir(studWorkDir, 0755)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
} 

func (conf *Config) GetAdmin(userName string) *Admin {
	userName = strings.ToLower(userName)
	for i, admin := range conf.Admin {
		if strings.ToLower(admin.User) == userName {
			return &conf.Admin[i]
		}
	}
	return nil
}

func (conf *Config) GetAdminByGroup(group string) *Admin {
	for i, admin := range conf.Admin {
		if admin.Group == group {
			return &conf.Admin[i]
		}
	}
	return nil
}

func (conf *Config) GetStudent(userName string) *Student {
	userName = strings.ToLower(userName)
	for i, stud := range conf.Student {
		if strings.ToLower(stud.UserName) == userName {
			return &conf.Student[i]
		}
	}
	return nil
}

func (conf *Config) GetReportType(format string) *ReportType {
	format = strings.ToLower(format)
	for i, report := range conf.Report {
		if strings.ToLower(report.Format) == format {
			return &conf.Report[i]
		}
	}
	return nil
}

package seekret

import (
	"fmt"
	"github.com/apuigsech/seekret/models"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Seekret struct {
	ruleList      []models.Rule
	objectList    []models.Object
	secretList    []models.Secret
	exceptionList []models.Exception
}

func NewSeekret() *Seekret {
	s := &Seekret{}
	return s
}

func (s *Seekret) GroupObjectsByMetadata(k string) map[string][]models.Object {
	return models.GroupObjectsByMetadata(s.objectList, k)
}

func (s *Seekret) GroupObjectsByPrimaryKeyHash() map[string][]models.Object {
	return models.GroupObjectsByPrimaryKeyHash(s.objectList)
}

type ruleYaml struct {
	ObjectMatch string
	Match       string
	Unmatch     []string
}

const DefaultRulesDir = "$GOPATH/src/github.com/apuigsech/seekret/rules"

func DefaultRulesPath() string {
	rulesPath := os.Getenv("SEEKRET_RULES_PATH")
	if rulesPath == "" {
		rulesPath = os.ExpandEnv(DefaultRulesDir)
	}
	return rulesPath
}

func (s *Seekret) AddRule(rule models.Rule, enabled bool) {
	if enabled {
		rule.Enable()
	}
	s.ruleList = append(s.ruleList, rule)
}

func (s *Seekret) LoadRulesFromFile(file string, defaulEnabled bool) error {
	var ruleYamlMap map[string]ruleYaml

	if file == "" {
		return nil
	}

	filename, _ := filepath.Abs(file)

	ruleBase := filepath.Base(filename)
	if filepath.Ext(ruleBase) == ".rule" {
		ruleBase = ruleBase[0 : len(ruleBase)-5]
	}

	yamlData, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlData, &ruleYamlMap)
	if err != nil {
		return err
	}

	for k, v := range ruleYamlMap {
		rule, err := models.NewRule(ruleBase+"."+k, v.Match)
		if err != nil {
			return err
		}

		for _, e := range v.Unmatch {
			rule.AddUnmatch(e)
		}
		s.AddRule(*rule, defaulEnabled)
	}

	return nil
}

func (s *Seekret) LoadRulesFromDir(dir string, defaulEnabled bool) error {
	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		err := fmt.Errorf("%s is not a directory", dir)
		return err
	}

	fileList, err := filepath.Glob(dir + "/*")
	if err != nil {
		return err
	}
	for _, file := range fileList {
		if strings.HasSuffix(file, ".rule") {
			err := s.LoadRulesFromFile(file, defaulEnabled)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Seekret) LoadRulesFromPath(path string, defaulEnabled bool) error {
	if path == "" {
		path = os.ExpandEnv(DefaultRulesDir)
	}
	dirList := strings.Split(path, ":")
	for _, dir := range dirList {
		err := s.LoadRulesFromDir(dir, defaulEnabled)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Seekret) ListRules() []models.Rule {
	return s.ruleList
}

func (s *Seekret) EnableRule(name string) error {
	return setRuleEnabled(s.ruleList, name, true)
}

func (s *Seekret) DisableRule(name string) error {
	return setRuleEnabled(s.ruleList, name, false)
}

func setRuleEnabled(ruleList []models.Rule, name string, enabled bool) error {
	found := false
	for i, r := range ruleList {
		if r.Name == name {
			found = true
			ruleList[i].Enabled = enabled
		}
	}
	if !found {
		err := fmt.Errorf("%s rule not found", name)
		return err
	}

	return nil
}

type exceptionYaml struct {
	Rule    *string
	Object  *string
	Line    *int
	Content *string
}

func (s *Seekret) AddException(exception models.Exception) {
	s.exceptionList = append(s.exceptionList, exception)
}

func (s *Seekret) LoadExceptionsFromFile(file string) error {
	var exceptionYamlList []exceptionYaml

	if file == "" {
		return nil
	}

	filename, _ := filepath.Abs(file)
	yamlData, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlData, &exceptionYamlList)
	if err != nil {
		return err
	}

	for _, v := range exceptionYamlList {
		x := models.NewException()

		if v.Rule != nil {
			err := x.SetRule(*v.Rule)
			if err != nil {
				return err
			}
		}

		if v.Object != nil {
			err := x.SetObject(*v.Object)
			if err != nil {
				return err
			}
		}

		if v.Line != nil {
			err := x.SetNline(*v.Line)
			if err != nil {
				return err
			}
		}

		if v.Content != nil {
			err := x.SetContent(*v.Content)
			if err != nil {
				return err
			}
		}

		s.AddException(*x)
	}

	return nil
}

func exceptionCheck(exceptionList []models.Exception, secret models.Secret) bool {
	for _, x := range exceptionList {
		match := x.Run(&secret)

		if match {
			return true
		}
	}
	return false
}

func (s *Seekret) ListSecrets() []models.Secret {
	return s.secretList
}

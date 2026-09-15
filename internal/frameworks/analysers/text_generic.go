package analysers

import (
	"bufio"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type textGeneric struct {
	baseDir string
	fn      string
	emails  []string
	domain  string
	vars    map[string]bool
}

// List of recognized email providers
var recognizedEmailProviders = []string{
	"gmail.com",
	"yahoo.com",
	"hotmail.com",
	"outlook.com",
	"protonmail.com",
	"zoho.com",
	"gmx.com",
	"icloud.com",
	"yandex.com",
	"aol.com",
	"mail.com",
	"tutanota.com",
	"fastmail.com",
}

func (t *textGeneric) Load() error {
	filePath := filepath.Join(t.baseDir, t.fn)
	logger.Info("Loading file: " + filePath)

	t.emails = []string{}
	t.domain = ""
	t.vars = map[string]bool{}

	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("Failed to open the file", err)
		return err
	}
	defer func() { _ = file.Close() }()

	// Regular expression to identify email addresses
	emailRegexPattern := `\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`
	emailRegex := regexp.MustCompile(emailRegexPattern)
	unrecognizedDomains := map[string]bool{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		emails := emailRegex.FindAllString(line, -1)
		for _, email := range emails {
			if t.isPartOfURLorEnv(email, line) {
				continue
			}
			if !contains(t.emails, email) {
				t.emails = append(t.emails, email)
				logger.Info("Email found: " + email)
				t.vars["author_email"] = true

				// Check and create abstract candidate for private domain
				parts := strings.Split(email, "@")
				if len(parts) == 2 {
					domain := parts[1]
					if !isRecognizedProvider(domain) {
						logger.Info("Unrecognized domain found: " + domain)
						t.vars["project_domain"] = true
						unrecognizedDomains[domain] = true
					}
				}
			}
		}
	}

	if len(unrecognizedDomains) == 1 {
		for domain := range unrecognizedDomains {
			t.domain = domain
			logger.Info("Single unrecognized domain: " + t.domain)
		}
	}

	if err = scanner.Err(); err != nil {
		logger.Error("Error while scanning the file", err)
		return err
	}

	return nil
}

func (t *textGeneric) GetAbstractCandidates() ([]*entities.AbstractMappingCandidate, error) {
	candidates := []*entities.AbstractMappingCandidate{}

	if t.domain != "" {
		candidates = append(candidates, &entities.AbstractMappingCandidate{
			For:      entities.MappingContent,
			Name:     "project_domain",
			Key:      t.domain,
			Value:    "{{ .project_domain }}",
			Priority: 1,
		})
		logger.Info("Abstract candidate for project domain: " + t.domain)

		for _, email := range t.emails {
			if strings.HasSuffix(email, "@"+t.domain) {
				candidates = append(candidates, &entities.AbstractMappingCandidate{
					For:      entities.MappingContent,
					Name:     "author_email",
					Key:      email,
					Value:    "{{ .author_email }}",
					Priority: 2,
				})
				logger.Info("Abstract candidate for author email: " + email)
				break
			}
		}
	}

	return candidates, nil
}

func (t *textGeneric) GetFileAnalysis() ([]*entities.FileAnalysis, error) {
	emailMapping := entities.Mappings{}
	for _, email := range t.emails {
		emailMapping[email] = "{{ .author_email }}"
	}

	varsList := make([]string, 0, len(t.vars))
	for key := range t.vars {
		varsList = append(varsList, key)
	}

	logger.Info("File analysis prepared for: " + t.fn)
	return []*entities.FileAnalysis{
		{
			Pattern: &entities.Pattern{
				Pattern: entities.Wildcard(t.fn),
				Content: emailMapping,
			},
			Vars: varsList,
		},
	}, nil
}

// isPartOfURLorEnv filters out email-like strings that are part of URLs or environment variables
func (t *textGeneric) isPartOfURLorEnv(email, text string) bool {
	// Surrounding symbols that might indicate it's a part of URL or variable
	precedingSymbols := []string{"$", ":", "/", "@"}
	followingSymbols := []string{"/", "?", ":"}

	// Find the position of the email in the text
	pos := strings.Index(text, email)
	if pos == -1 {
		return false
	}

	// Check the character before and after the email
	if pos > 0 {
		precedingChar := string(text[pos-1])
		for _, symbol := range precedingSymbols {
			if precedingChar == symbol {
				return true // Likely part of an env variable or URL
			}
		}
	}

	if pos+len(email) < len(text) {
		followingChar := string(text[pos+len(email)])
		for _, symbol := range followingSymbols {
			if followingChar == symbol {
				return true // Likely part of an env variable or URL
			}
		}
	}

	return false
}

func isRecognizedProvider(domain string) bool {
	for _, provider := range recognizedEmailProviders {
		if domain == provider {
			return true
		}
	}
	return false
}

func (t *textGeneric) GetFileName() entities.File {
	return entities.File(t.fn)
}

func newTextGenericFile(baseDir, fn string) (usecases.LocalFileAnalyserPort, error) {
	obj := &textGeneric{baseDir: baseDir, fn: fn}
	logger.Info("Creating new textGeneric file object")
	if err := obj.Load(); err != nil {
		logger.Error("Failed to load the textGeneric file", err)
		return nil, err
	}
	logger.Info("textGeneric file object successfully created")
	return obj, nil
}

func acceptTextGenericFile(_, fn string) bool {
	// Any non-binary file will be handled by TextGeneric analyser
	logger.Info("Accepting file for textGeneric analysis: " + fn)
	return true
}

var _ usecases.LocalFileAnalyserPort = (*textGeneric)(nil)

func init() {
	// Registering the generic text file analyser with a lower priority
	Register(acceptTextGenericFile, newTextGenericFile, 100)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

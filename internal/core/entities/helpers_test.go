package entities

import "testing"

func TestDeriveID(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{name: "host and org", uri: "github.com/your-org/your-template", want: "your-template"},
		{name: "dot git suffix", uri: "github.com/your-org/your-template.git", want: "your-template"},
		{name: "trailing slash", uri: "github.com/your-org/your-template/", want: "your-template"},
		{name: "scheme", uri: "https://github.com/your-org/your-template", want: "your-template"},
		{name: "local path", uri: "/Users/me/templates/my-tpl", want: "my-tpl"},
		{name: "invalid characters", uri: "github.com/org/my.template repo", want: "my-template-repo"},
		{name: "only invalid characters", uri: "github.com/org/...", want: "template"},
		{name: "empty", uri: "", want: "template"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveID(tt.uri); got != tt.want {
				t.Errorf("DeriveID(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestValidID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{id: "api", want: true},
		{id: "my-template_2", want: true},
		{id: "123", want: true},
		{id: "", want: false},
		{id: "my template", want: false},
		{id: "my.template", want: false},
		{id: "my/template", want: false},
	}

	for _, tt := range tests {
		if got := ValidID(tt.id); got != tt.want {
			t.Errorf("ValidID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestSombraDefHasID(t *testing.T) {
	def := &SombraDef{Templates: []*TemplateConfig{{ID: "api"}, {URI: "github.com/org/tpl"}}}

	if !def.HasID("api") {
		t.Error("expected HasID(api) to be true")
	}
	if def.HasID("missing") {
		t.Error("expected HasID(missing) to be false")
	}
	if def.HasID("") {
		t.Error("expected HasID(empty) to be false")
	}

	var nilDef *SombraDef
	if nilDef.HasID("api") {
		t.Error("expected nil def to have no ids")
	}
}

func TestSombraDefUniqueID(t *testing.T) {
	def := &SombraDef{Templates: []*TemplateConfig{{ID: "repo"}, {ID: "repo-2"}}}

	if got := def.UniqueID("repo"); got != "repo-3" {
		t.Errorf("UniqueID(repo) = %q, want repo-3", got)
	}
	if got := def.UniqueID("fresh"); got != "fresh" {
		t.Errorf("UniqueID(fresh) = %q, want fresh", got)
	}
}

func TestResolveTemplates(t *testing.T) {
	api := &TemplateConfig{ID: "api", URI: "github.com/org/repo", Path: "services/api"}
	web := &TemplateConfig{ID: "web", URI: "github.com/org/repo", Path: "services/web"}
	legacy := &TemplateConfig{URI: "github.com/org/legacy"}
	def := &SombraDef{Templates: []*TemplateConfig{api, web, legacy}}

	tests := []struct {
		name string
		ref  string
		want []*TemplateConfig
	}{
		{name: "by id", ref: "api", want: []*TemplateConfig{api}},
		{name: "by id targets one entry sharing a uri", ref: "web", want: []*TemplateConfig{web}},
		{name: "by uri returns all matches", ref: "github.com/org/repo", want: []*TemplateConfig{api, web}},
		{name: "uri fallback for legacy entries", ref: "github.com/org/legacy", want: []*TemplateConfig{legacy}},
		{name: "unknown", ref: "nope", want: nil},
		{name: "empty", ref: "", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTemplates(def, tt.ref)
			if len(got) != len(tt.want) {
				t.Fatalf("ResolveTemplates(%q) returned %d entries, want %d", tt.ref, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ResolveTemplates(%q)[%d] = %+v, want %+v", tt.ref, i, got[i], tt.want[i])
				}
			}
		})
	}

	if got := ResolveTemplates(nil, "api"); got != nil {
		t.Errorf("ResolveTemplates(nil) = %+v, want nil", got)
	}
}

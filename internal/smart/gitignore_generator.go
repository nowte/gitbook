package smart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ── Akıllı .gitignore Üreteci ─────────────────────────────────────────────────
//
// Proje dizinini tarayarak hangi teknoloji yığınının kullanıldığını
// tespit eder ve buna uygun .gitignore şablonunu otomatik üretir.

// Stack tespit edilen teknoloji yığınını temsil eder.
type Stack struct {
	Name      string
	Indicator string // hangi dosya varlığı tetikledi
	Priority  int    // yüksek = daha önemli
}

// GitignoreResult üretme işleminin sonucunu tutar.
type GitignoreResult struct {
	Stacks       []Stack
	Content      string
	AlreadyExist bool
	WasMerged    bool // mevcut .gitignore ile birleştirildi
}

// GenerateGitignore proje dizinini analiz ederek .gitignore içeriği üretir.
func GenerateGitignore(projectDir string) *GitignoreResult {
	result := &GitignoreResult{}

	// Mevcut .gitignore var mı?
	existingPath := filepath.Join(projectDir, ".gitignore")
	existingContent := ""
	if data, err := os.ReadFile(existingPath); err == nil {
		existingContent = string(data)
		result.AlreadyExist = true
	}

	// Yığınları tespit et
	result.Stacks = detectStacks(projectDir)

	if len(result.Stacks) == 0 {
		// Hiçbir şey tespit edilemedi → genel şablon
		result.Stacks = []Stack{{Name: "general", Indicator: "varsayılan", Priority: 1}}
	}

	// Her yığın için şablon topla
	var sections []string
	seen := make(map[string]bool)
	for _, stack := range result.Stacks {
		tmpl := templateFor(stack.Name)
		if tmpl != "" && !seen[stack.Name] {
			seen[stack.Name] = true
			sections = append(sections, fmt.Sprintf("# ── %s ──────────────────────────────────────\n%s", stack.Name, tmpl))
		}
	}

	// Evrensel kısım her zaman eklenir
	sections = append(sections, universalTemplate())

	newContent := strings.Join(sections, "\n\n")

	// Mevcut içerik varsa: yeni satırları ekle, tekrarları atla
	if existingContent != "" {
		merged := mergeGitignore(existingContent, newContent)
		result.Content = merged
		result.WasMerged = true
	} else {
		result.Content = newContent
	}

	return result
}

// detectStacks proje dizinindeki dosyalara bakarak teknoloji yığınlarını bulur.
func detectStacks(dir string) []Stack {
	var stacks []Stack

	indicators := []struct {
		file     string
		stack    string
		priority int
	}{
		// Go
		{"go.mod", "Go", 10},
		{"go.sum", "Go", 9},
		// Node / JavaScript / TypeScript
		{"package.json", "Node", 10},
		{"package-lock.json", "Node", 8},
		{"yarn.lock", "Node", 8},
		{"pnpm-lock.yaml", "Node", 8},
		{"tsconfig.json", "TypeScript", 9},
		// Python
		{"requirements.txt", "Python", 10},
		{"setup.py", "Python", 10},
		{"pyproject.toml", "Python", 10},
		{"Pipfile", "Python", 9},
		{"poetry.lock", "Python", 8},
		// Rust
		{"Cargo.toml", "Rust", 10},
		{"Cargo.lock", "Rust", 9},
		// Java / Kotlin
		{"pom.xml", "Maven", 10},
		{"build.gradle", "Gradle", 10},
		{"build.gradle.kts", "Gradle", 10},
		// .NET / C#
		{"*.sln", "DotNet", 10},
		{"*.csproj", "DotNet", 10},
		// Ruby
		{"Gemfile", "Ruby", 10},
		{"Gemfile.lock", "Ruby", 9},
		// PHP
		{"composer.json", "PHP", 10},
		// Swift / iOS
		{"Package.swift", "Swift", 10},
		{"*.xcodeproj", "Xcode", 10},
		{"*.xcworkspace", "Xcode", 10},
		// Docker
		{"Dockerfile", "Docker", 7},
		{"docker-compose.yml", "Docker", 7},
		{"docker-compose.yaml", "Docker", 7},
		// Terraform
		{"main.tf", "Terraform", 9},
		{"*.tf", "Terraform", 8},
		// Flutter / Dart
		{"pubspec.yaml", "Flutter", 10},
		// Elixir
		{"mix.exs", "Elixir", 10},
		// Unity
		{"ProjectSettings", "Unity", 10},
		// macOS / OS
		{".DS_Store", "macOS", 5},
	}

	seen := make(map[string]bool)

	for _, ind := range indicators {
		stackKey := ind.stack
		if seen[stackKey] {
			continue
		}

		var found bool
		if strings.Contains(ind.file, "*") {
			// Glob pattern
			matches, err := filepath.Glob(filepath.Join(dir, ind.file))
			found = err == nil && len(matches) > 0
		} else {
			_, err := os.Stat(filepath.Join(dir, ind.file))
			found = err == nil
		}

		if found {
			stacks = append(stacks, Stack{
				Name:      stackKey,
				Indicator: ind.file,
				Priority:  ind.priority,
			})
			seen[stackKey] = true
		}
	}

	// IDE dosyalarını da tespit et
	ideIndicators := []struct {
		dir   string
		stack string
	}{
		{".idea", "JetBrains"},
		{".vscode", "VSCode"},
		{"nbproject", "NetBeans"},
	}
	for _, ide := range ideIndicators {
		if _, err := os.Stat(filepath.Join(dir, ide.dir)); err == nil {
			if !seen[ide.stack] {
				stacks = append(stacks, Stack{Name: ide.stack, Indicator: ide.dir, Priority: 5})
				seen[ide.stack] = true
			}
		}
	}

	return stacks
}

// mergeGitignore mevcut .gitignore ile yeni içeriği birleştirir,
// zaten var olan satırları tekrar eklemez.
func mergeGitignore(existing, newContent string) string {
	existingLines := make(map[string]bool)
	for _, line := range strings.Split(existing, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			existingLines[trimmed] = true
		}
	}

	var toAdd []string
	currentSection := ""
	sectionHasNew := false
	var pendingSection string

	for _, line := range strings.Split(newContent, "\n") {
		trimmed := strings.TrimSpace(line)

		// Bölüm başlığı
		if strings.HasPrefix(trimmed, "# ──") {
			if sectionHasNew && pendingSection != "" {
				toAdd = append(toAdd, pendingSection)
			}
			currentSection = line
			pendingSection = line
			sectionHasNew = false
			_ = currentSection
			continue
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			pendingSection += "\n" + line
			continue
		}

		if !existingLines[trimmed] {
			pendingSection += "\n" + line
			sectionHasNew = true
		}
	}

	// Son bölümü ekle
	if sectionHasNew && pendingSection != "" {
		toAdd = append(toAdd, pendingSection)
	}

	if len(toAdd) == 0 {
		return existing // Eklenecek yeni şey yok
	}

	merged := strings.TrimRight(existing, "\n") +
		"\n\n# ── gitBook tarafından eklendi ─────────────────────────\n" +
		strings.Join(toAdd, "\n")
	return merged
}

// templateFor belirtilen yığın için .gitignore şablonu döner.
func templateFor(stack string) string {
	templates := map[string]string{
		"Go": `# Derlenmiş ikili dosyalar
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out

# Go build cache
/vendor/
/dist/

# Test çıktıları
coverage.out
coverage.html

# Go workspace
go.work.sum`,

		"Node": `# Bağımlılıklar
node_modules/
npm-debug.log*
yarn-debug.log*
yarn-error.log*
pnpm-debug.log*

# Build çıktıları
dist/
build/
.next/
.nuxt/
out/

# Çalışma zamanı dosyaları
.npm
.yarn/cache
.yarn/unplugged
.pnp.*

# Ortam dosyaları
.env
.env.local
.env.development.local
.env.test.local
.env.production.local`,

		"TypeScript": `# TypeScript derleme çıktıları
*.js.map
*.d.ts.map
tsconfig.tsbuildinfo`,

		"Python": `# Byte-compiled / optimized
__pycache__/
*.py[cod]
*$py.class
*.pyc

# Dağıtım / paketleme
.Python
build/
develop-eggs/
dist/
downloads/
eggs/
.eggs/
lib/
lib64/
parts/
sdist/
var/
wheels/
*.egg-info/
.installed.cfg
*.egg
MANIFEST

# Sanal ortamlar
.env
.venv
env/
venv/
ENV/
env.bak/
venv.bak/

# Test & coverage
.tox/
.nox/
.coverage
.coverage.*
.cache
htmlcov/
.pytest_cache/
nosetests.xml
coverage.xml

# Jupyter Notebook
.ipynb_checkpoints

# mypy / pyright
.mypy_cache/
.dmypy.json
dmypy.json
.pyright/`,

		"Rust": `# Cargo build çıktıları
/target/
Cargo.lock

# Benchmark çıktıları
criterion/`,

		"Maven": `# Maven build çıktıları
target/
pom.xml.tag
pom.xml.releaseBackup
pom.xml.versionsBackup
pom.xml.next
release.properties`,

		"Gradle": `# Gradle
.gradle/
build/
!gradle/wrapper/gradle-wrapper.jar
!**/src/main/**/build/
!**/src/test/**/build/

# Gradle wrapper
gradle-wrapper.jar`,

		"DotNet": `# .NET build
[Bb]in/
[Oo]bj/
[Ll]og/
[Ll]ogs/

# NuGet
*.nupkg
*.snupkg
project.lock.json
project.fragment.lock.json
artifacts/

# Visual Studio
.vs/
*.user
*.suo
*.userosscache
*.sln.docstates`,

		"Ruby": `# Ruby gems
/.bundle/
/vendor/bundle
Gemfile.lock
*.gem

# Rails
/log/*
/tmp/*
!/log/.keep
!/tmp/.keep
/public/system
/public/uploads
/storage/*`,

		"PHP": `# Composer
/vendor/
composer.lock
.phpunit.result.cache

# Laravel
/bootstrap/cache/
/public/storage
.env
Homestead.json
Homestead.yaml`,

		"Swift": `# Swift Package Manager
.build/
/Packages/
/*.xcodeproj
xcuserdata/
DerivedData/`,

		"Xcode": `# Xcode
build/
DerivedData/
*.pbxuser
!default.pbxuser
*.mode1v3
!default.mode1v3
*.mode2v3
!default.mode2v3
*.perspectivev3
!default.perspectivev3
xcuserdata/
*.xccheckout
*.moved-aside
*.xcuserstate
*.xcscmblueprint`,

		"Docker": `# Docker
.dockerignore
docker-compose.override.yml`,

		"Terraform": `# Terraform
.terraform/
.terraform.lock.hcl
*.tfstate
*.tfstate.*
*.tfvars
crash.log
override.tf
override.tf.json
*_override.tf
*_override.tf.json`,

		"Flutter": `# Flutter
.dart_tool/
.flutter-plugins
.flutter-plugins-dependencies
.packages
.pub-cache/
.pub/
build/
flutter_*.png`,

		"Elixir": `# Elixir
/_build/
/cover/
/deps/
/doc/
/.fetch
erl_crash.dump
*.ez
*.beam
/config/*.secret.exs`,

		"Unity": `# Unity
/[Ll]ibrary/
/[Tt]emp/
/[Oo]bj/
/[Bb]uild/
/[Bb]uilds/
/[Ll]ogs/
/[Uu]ser[Ss]ettings/
*.pidb.meta
*.pdb.meta
*.mdb.meta`,

		"JetBrains": `# JetBrains IDE
.idea/
*.iml
*.ipr
*.iws
out/
.idea_modules/`,

		"VSCode": `# Visual Studio Code
.vscode/
!.vscode/settings.json
!.vscode/tasks.json
!.vscode/launch.json
!.vscode/extensions.json
*.code-workspace
.history/`,

		"macOS": `# macOS
.DS_Store
.AppleDouble
.LSOverride
Icon
._*
.DocumentRevisions-V100
.fseventsd
.Spotlight-V100
.TemporaryItems
.Trashes
.VolumeIcon.icns
.com.apple.timemachine.donotpresent`,
	}

	if tmpl, ok := templates[stack]; ok {
		return tmpl
	}
	return ""
}

// universalTemplate her projede olması gereken evrensel kurallar.
func universalTemplate() string {
	return `# ── Evrensel ──────────────────────────────────────────────────────────────
# Ortam ve gizli dosyalar
.env
.env.*
!.env.example
*.secret
*.secrets
secrets/

# Log dosyaları
*.log
logs/

# Geçici dosyalar
*.tmp
*.temp
*.swp
*.bak
*~

# OS dosyaları
.DS_Store
Thumbs.db
desktop.ini

# Editor dosyaları
.idea/
.vscode/
*.sublime-project
*.sublime-workspace`
}

// FormatGitignoreResult sonucu ekranda gösterilecek şekilde formatlar.
func FormatGitignoreResult(r *GitignoreResult) string {
	var sb strings.Builder

	if len(r.Stacks) > 0 {
		stackNames := make([]string, len(r.Stacks))
		for i, s := range r.Stacks {
			stackNames[i] = s.Name
		}
		sb.WriteString(fmt.Sprintf("  [?] Tespit edilen yığınlar: %s\n", strings.Join(stackNames, ", ")))
	}

	if r.AlreadyExist {
		if r.WasMerged {
			sb.WriteString("  [~] Mevcut .gitignore güncellendi (eksik kurallar eklendi)\n")
		} else {
			sb.WriteString("  [ok] .gitignore zaten güncel, eklenecek yeni kural yok\n")
		}
	} else {
		sb.WriteString("  [*] Yeni .gitignore oluşturuldu\n")
	}

	lineCount := len(strings.Split(r.Content, "\n"))
	sb.WriteString(fmt.Sprintf("  [>] %d satır kural yazıldı\n", lineCount))

	return sb.String()
}

# variables

# updates a module repository
#update_provider:

# creates a module repository and builds it's file structure(creation incomplete rn)
#create_module: build_provider

# generates the filestructure for a module repository
build_provider:

		mkdir .changelog .github examples provider_domain scripts version website

		mkdir website/docs scripts/affectedtests .github/ISSUE_TEMPLATE .github/workflows 

		mkdir website/docs/guides

		touch .gitignore .go-version .golangci.yml .goreleaser.yml CHANGELOG.md Makefile LICENSE README.md main.go tools.go \ 
		version/version.go scripts/affectedtests/affectedtests.go scripts/diff.go scripts/docscheck.sh scripts/gofmtcheck.sh \ 
		scripts/gogetcookie.sh scripts/run_diff.sh scripts/go111modulecheck.sh .github/ISSUE_TEMPLATE/bug.md \ 
		.github/ISSUE_TEMPLATE/enhancement.md .github/ISSUE_TEMPLATE/question.md .github/ISSUE_TEMPLATE/test_failure.md \ 
		.github/workflows/go.yml .github/workflows/issue-comment-created.yml .github/workflows/labeler.yml .github/workflows/lock.yml \
		 .github/workflows/pull-request-reviewer.yml .github/workflows/pull-request-size.yml .github/workflows/release.yml \ 
		 .github/workflows/stale.yml .github/CODE_OF_CONDUCT.md .github/CONTRIBUTING.md .github/ISSUE_TEMPLATE.md .github/SUPPORT.md \ 
		 .github/dependabot.yml .github/labeler.yml .github/reviewer-lottery.yml website/docs/guides/getting_started.md \ 
		 website/docs/guides/provider_reference.md website/docs/guides/provider_versions.md

# touch LICENSE README main.tf

# destroys module repository file structure
#destroy_provider:

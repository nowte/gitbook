package lang

// Example language file for developers
// Copy this file to create new language translations
// Replace "Example" with your language code (e.g., "tr", "fr", "de")

type ExampleLanguage struct{}

func (e ExampleLanguage) GetTranslations() map[string]string {
	return map[string]string{
		// Commands
		"cmd_new":        "Clear session",
		"cmd_exit":       "Exit the app",
		"cmd_help":       "Show this help",
		"cmd_tutorial":   "Git basics tutorial for beginners",
		"cmd_setup":      "Set git identity (name & email)",
		"cmd_config":     "Show current identity and repo info",
		"cmd_cd":         "Change working directory (supports ~)",
		"cmd_path":       "Open file browser to select directory",
		"cmd_pwd":        "Print current directory",
		"cmd_init":       "Initialise git + gitBook in current dir",
		"cmd_status":     "Git status + branch + sync info",
		"cmd_branch":     "List local branches",
		"cmd_log":        "Show recent commits",
		"cmd_review":     "Show diff of staged & unstaged changes",
		"cmd_diff":       "Show diff (unstaged or vs branch)",
		"cmd_blame":      "Line-by-line authorship for a file",
		"cmd_remote":     "List configured remotes",
		"cmd_push":       "Push current branch",
		"cmd_pull":       "Pull current branch",
		"cmd_fetch":      "Fetch all remotes",
		"cmd_clone":      "Clone a repository",
		"cmd_sync":       "Fetch and show ahead/behind count",
		"cmd_github":     "Connect repo to GitHub and push",
		"cmd_tag":        "List or create a tag",
		"cmd_tag_push":   "Push all tags to remote",
		"cmd_stash":      "Stash uncommitted changes",
		"cmd_stash_list": "List all stashes",
		"cmd_stash_pop":  "Apply and remove most recent stash",
		"cmd_reset":      "Soft-reset last n commits",
		"cmd_reset_hard": "Hard-reset last n commits",
		"cmd_revert":     "Create a revert commit",
		"cmd_cherry_pick": "Apply a commit to current branch",
		"cmd_amend":      "Amend last commit message",
		"cmd_stage":      "Stage files",
		"cmd_unstage":    "Unstage a file",
		"cmd_commit":     "Commit staged changes",
		"cmd_start":      "Start a new feature branch",
		"cmd_finish":     "Finish feature and merge into base",
		"cmd_cleanup":    "Delete merged feature branch",

		// Messages
		"msg_session_cleared":     "Session cleared.",
		"msg_git_identity_setup": "Git identity setup. What is your full name?",
		"msg_working_directory":  "Working directory: %s",
		"msg_not_git_repo":      "Not a git repository.",
		"msg_repository_initialised": "Repository initialised. Run /setup to set your identity.",
		"msg_already_git_repo":   "Already inside a git repository.",
		"msg_gitbook_initialised": "gitBook: initialised",
		"msg_working_tree_clean": "Working tree clean.",
		"msg_up_to_date":       "Up to date",
		"msg_select_directory":  "Select a directory to navigate to:",
		"msg_select_dir_not_file": "Please select a directory, not a file.",
		"msg_cancelled":         "Cancelled.",

		// File browser
		"file_browser_help": " ↑↓ Navigate  Enter: Select  Esc: Cancel  Tab: Change Directory",
		"file_browser_parent": "..",

		// Help sections
		"help_session": "Session",
		"help_setup_config": "Setup & Config",
		"help_repository": "Repository",
		"help_feature_workflow": "Feature Workflow",
		"help_staging_committing": "Staging & Committing",
		"help_remote_github": "Remote & GitHub",
		"help_stash": "Stash",
		"help_tags": "Tags",
		"help_undo_history": "Undo & History",

		// Help title
		"help_title": "gitBook Commands",
		"help_separator": "─────────────────────────────────────────────",

		// Usage messages
		"usage_cd": "Usage: /cd <path>",
		"usage_clone": "Usage: /clone <url> [directory]",
		"usage_github": "Usage: /github <url>",
		"usage_start": "Usage: /start <feature-name>",
		"usage_unstage": "Usage: /unstage <path>",
		"usage_amend": "Usage: /amend <new message>",
		"usage_revert": "Usage: /revert <commit-hash>",
		"usage_cherry_pick": "Usage: /cherry-pick <commit-hash>",
		"usage_blame": "Usage: /blame <file>",
	}
}

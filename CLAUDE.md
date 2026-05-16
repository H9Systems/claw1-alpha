
## Agent conventions

When working with Claw1, keep these conventions in mind:

- All infrastructure is declared in Terraform (HCL) and deployed via `terraform apply`
- Contracts live in `contracts/src/` and are tested with `forge test`
- The provider lives in `terraform-provider-claw1/` and is installed via `make install`
- Demo state is in `~/.claw1/{name}/network.json` — never commit this file
- Reset between demos with `demo/reset.sh`

## Skill routing

When the user's request matches an available skill, invoke it via the Skill tool. When in doubt, invoke the skill.

Key routing rules:
- Product ideas/brainstorming → invoke /office-hours
- Strategy/scope → invoke /plan-ceo-review
- Architecture → invoke /plan-eng-review
- Design system/plan review → invoke /design-consultation or /plan-design-review
- Full review pipeline → invoke /autoplan
- Bugs/errors → invoke /investigate
- QA/testing site behavior → invoke /qa or /qa-only
- Code review/diff check → invoke /review
- Visual polish → invoke /design-review
- Ship/deploy/PR → invoke /ship or /land-and-deploy
- Update or audit public docs → invoke /pub-docs (runs privacy check against .private/blocklist.txt)
- Save progress → invoke /context-save
- Resume context → invoke /context-restore

## Pre-commit doc sync

**Before every `git commit` that touches `cli/`, `terraform/`, `contracts/`, `run.sh`, `preflight.sh`, or `demo/`:**
invoke `/update-docs` first. The skill checks staged changes, updates `DOCS.md` and `DOCS.en.md` to match, and stages the updated files. Then proceed with the commit.

Skip `/update-docs` only when the staged diff contains exclusively `*.md` files, test fixtures, or gitignore changes with no operational behavior change.


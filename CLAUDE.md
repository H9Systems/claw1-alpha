
## Agent conventions

When working with Claw1, keep these conventions in mind:

- All infrastructure is declared in Terraform (HCL) and deployed via `terraform apply`
- Contracts live in `contracts/src/` and are tested with `forge test`
- The provider lives in `terraform/providers/terraform-provider-claw1/` and is installed via `make install`
- Demo state is in `~/.claw1/{name}/network.json` — never commit this file
- Reset between demos with `scripts/reset.sh`
- `AGENTS.md` is a symlink to this file so Codex uses the same repo rules
- Current product surface is the Go `claw1` TUI/CLI, not a web wizard
- Root `/` is a static Spanish pitch deck generated from `PITCH.md`
- `TODOS.md` is English-only and frequently updated by gstack; do not include it in translation workflows or translation skills
- Blockscout and MetaMask must not be required for the critical demo path
- OCI destroy flows must fail closed: dry-run, inventory, destroy, verify, and show remaining resource IDs if cleanup is imperfect
- `--preserve-evidence` is local-only; `--evidence-bucket` is the only explicit cloud retention mode

## Privacy guard

Files in `.private/` are gitignored and must never be committed. They contain contact info, pricing, GTM strategy, and origin context that must stay local.

When writing or editing any `.md` file outside `.private/`, check against `.private/blocklist.txt` before finalizing. If a blocklisted term appears, replace it:

| Category | Public replacement |
|-----------|-------------------|
| Specific pricing | "*por definir*" / "*TBD*" |
| Revenue projections | "*por definir*" / "*TBD*" |
| ARR / ACV | "Ingresos" / "Revenue" |
| Hackathon references | "lanzamiento" / "launch" or "MVP inicial" / "initial MVP" |
| Event judges/sponsors | "evaluadores" / "evaluators" |
| Founder's home market | just "Panama wedge" (no personal detail) |
| Sales tools (Loom) | "video" |
| Partner internals (ISV, ecosystem funds) | generic "cloud partner" references |
| Fundraising | "siguientes etapas" / "next stage" |
| Business model claims | add ⚠️ disclaimer: "not an official statement and could change entirely" |
| Named prospect companies | remove entirely |
| Named prospect people | remove entirely |

When in doubt, move the detail to `.private/context.md` and generalize in the public file.

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

**Before every `git commit` that touches `cli/`, `terraform/`, `contracts/`, `run.sh`, `preflight.sh`, or `scripts/`:**
invoke `/update-docs` first. The skill checks staged changes, updates `DOCS.md` and `DOCS.en.md` to match, and stages the updated files. Then proceed with the commit.

Skip `/update-docs` only when the staged diff contains exclusively `*.md` files, test fixtures, or gitignore changes with no operational behavior change.

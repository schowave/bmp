# Agent Readiness: bmp

2026-10-23 · commit `a2f21af` with uncommitted changes · branch `main` · skill 0.38.2

**Level 3 – Agent-ready** · overall 44 / 99 passed (44 %)

> **What agents may do now:** Agents implement clearly bounded tickets on their own, reviewed by pull request.

## Levels

| Level | Passed / evaluated / total | Cleared |
| --- | --- | --- |
| L1 Baseline | 12 / 12 / 12 | ✅ |
| L2 Documented | 13 / 23 / 25 | ✅ |
| L3 Agent-ready | 16 / 31 / 33 | ✅ |
| L4 Optimized | 3 / 25 / 27 | ❌ pass rate |
| L5 Autonomous | 0 / 8 / 11 | ❌ pass rate |

## Achieved

44 criteria in place.

| Level | State | What it gives | Suitable agent use |
| --- | --- | --- | --- |
| L1 Baseline | ✅ achieved | The repo has formatter, linter, types, and unit tests running locally. | Analyse code, write docs and tests; a human reviews every change. |
| L2 Documented | ✅ achieved | The repo writes its workflows down: agent guide, env template, schema, ownership. | Develop features with an agent, a human steering closely. |
| L3 Agent-ready | ✅ achieved | The repo enforces its standards through automation: CI, hooks, tracing, releases. | Agents implement clearly bounded tickets on their own, reviewed by pull request. |
| L4 Optimized | ➡️ next milestone, +10 to unlock | Unlocks: the repo measures its feedback loops: coverage gates, flaky tests, build times. | Several agents work in parallel in separate workspaces. |
| L5 Autonomous | 🔒 locked | Unlocks: the repo feeds production signals back into the loop: canaries, rollback, error-to-issue. | Agents react to production signals: error, issue, fix pull request. |

### Progress since 2026-10-23

Level 1 → 3 · 21 newly passing

- L1 **Lint Config** (`lint_config`): The agent sees its own mistakes in seconds instead of in review.
- L1 **Unit Tests Exist** (`unit_tests_exist`): Tests are the strongest signal an agent can check its change against.
- L1 **README** (`readme`): It is the agent's first orientation in the repo.
- L1 **Comprehensive .gitignore** (`gitignore_comprehensive`): Secrets and build noise stay out of what the agent commits.
- L2 **Check Command** (`check_command`): One mise task runs lint and every unit test, locally and in CI.
- L2 **Test Naming Conventions** (`test_naming_conventions`): The agent finds and extends the right tests by convention.
- L2 **Unit Tests Runnable** (`unit_tests_runnable`): Tests the agent can run itself let it verify every change.
- L2 **Test Stage Rules** (`test_stage_rules`): Named test stages tell the agent when to run the slower suites, so CI confirms its work instead of surprising it.
- L2 **Agent Guide** (`agents_md`): An agent guide is the single most effective readiness artifact.
- L2 **Agent Guide Substance** (`agents_md_size`): A guide with substance but no bulk teaches the agent the essentials and leaves room for the code.
- L2 **Structured Logging** (`structured_logging`): Parseable logs are evidence the agent can debug from.
- L2 **CODEOWNERS** (`codeowners`): Routes each change to the people who must review it.
- L2 **Agent Setup Owner** (`agent_setup_owner`): Without an owner, the agent setup drifts into what each developer happens to know, and nobody removes what no longer helps.
- L3 **Test Isolation** (`test_isolation`): Isolated tests give the agent pass/fail signals it can trust.
- L3 **Agent Guide Completeness** (`agents_md_content`): A guide that names the test command lets the agent verify every change.
- L3 **Agent Guide Freshness** (`agents_md_freshness`): Agents trust the guide, and a current guide earns that trust.
- L3 **Agent Boundaries** (`agent_boundaries`): What an agent may not decide alone it has to know before it acts, not after.
- L3 **Automated Doc Generation** (`automated_doc_generation`): Generated docs cannot drift, so the agent can trust them.
- L3 **Domain Docs** (`domain_docs`): Glossary and ADRs give the game terms and the reasons behind embedding and the C random generator.
- L3 **Skills** (`skills`): Screen comparison with the original and decompilation lookups are written down as skills.
- L4 **Agent Guide Structure** (`agents_md_validation`): Sections let the agent jump to what it needs.

### In place, by category

**Style & Validation** (6 of 12 assessed pass)

- L1 **Formatter** (`formatter`): Style churn disappears from diffs and the agent matches house style for free.
- L1 **Lint Config** (`lint_config`): The agent sees its own mistakes in seconds instead of in review.
- L1 **Strict Typing** (`strict_typing`): Strict types reject plausible-but-wrong edits before they run.
- L1 **Type Check** (`type_check`): Types are the cheapest correctness signal an agent can verify itself.
- L3 **Pre-commit Hooks** (`pre_commit_hooks`): Mistakes surface at commit time, while the change is still fresh.
- L3 **Code Modularization** (`code_modularization`): Enforced boundaries keep each agent edit inside one module.

**Build System** (11 of 24 assessed pass)

- L1 **Dependencies Pinned** (`deps_pinned`): The agent, the team, and CI resolve the same versions.
- L1 **Toolchain Pinned** (`toolchain_pinned`): The agent, the team, and CI run the same runtime versions, so every failure the agent sees is one everyone can reproduce.
- L1 **Documented Build Commands** (`build_cmd_doc`): With the canonical command written down, the agent can verify its own work.
- L1 **Git Hosting CLI** (`vcs_cli_tools`): The agent can open PRs and read CI results without a browser.
- L1 **Single Command Setup** (`single_command_setup`): One setup command gets the agent past the step where it most often gets stuck.
- L2 **Check Command** (`check_command`): One mise task runs lint and every unit test, locally and in CI.
- L3 **Agentic Development** (`agentic_development`): Existing agent config and skills are something to build on.
- L3 **Fast CI Feedback** (`fast_ci_feedback`): Short CI keeps the agent's edit-verify loop tight.
- L3 **Release Automation** (`release_automation`): Releases tag and publish only once both images exist.
- L4 **Build Performance Tracking** (`build_performance_tracking`): Tracked build times keep every agent loop fast as the codebase grows.
- L4 **Release Notes Automation** (`release_notes_automation`): Generated notes give commits a format the agent can follow.

**Testing** (6 of 11 assessed pass)

- L1 **Unit Tests Exist** (`unit_tests_exist`): Tests are the strongest signal an agent can check its change against.
- L2 **Test Naming Conventions** (`test_naming_conventions`): The agent finds and extends the right tests by convention.
- L2 **Unit Tests Runnable** (`unit_tests_runnable`): Tests the agent can run itself let it verify every change.
- L2 **Test Stage Rules** (`test_stage_rules`): Named test stages tell the agent when to run the slower suites, so CI confirms its work instead of surprising it.
- L3 **Integration Tests Exist** (`integration_tests_exist`): Both images are exercised end to end in CI and the remake's screens are driven headless.
- L3 **Test Isolation** (`test_isolation`): Isolated tests give the agent pass/fail signals it can trust.

**Documentation** (12 of 15 assessed pass)

- L1 **README** (`readme`): It is the agent's first orientation in the repo.
- L2 **Agent Guide** (`agents_md`): An agent guide is the single most effective readiness artifact.
- L2 **Agent Guide Substance** (`agents_md_size`): A guide with substance but no bulk teaches the agent the essentials and leaves room for the code.
- L2 **Documentation Freshness** (`documentation_freshness`): Current docs lead the agent along paths that still exist.
- L3 **Agent Guide Completeness** (`agents_md_content`): A guide that names the test command lets the agent verify every change.
- L3 **Agent Guide Freshness** (`agents_md_freshness`): Agents trust the guide, and a current guide earns that trust.
- L3 **Agent Boundaries** (`agent_boundaries`): What an agent may not decide alone it has to know before it acts, not after.
- L3 **Automated Doc Generation** (`automated_doc_generation`): Generated docs cannot drift, so the agent can trust them.
- L3 **Service Flow Documented** (`service_flow_documented`): Flows give the cross-service picture code alone does not.
- L3 **Domain Docs** (`domain_docs`): Glossary and ADRs give the game terms and the reasons behind embedding and the C random generator.
- L3 **Skills** (`skills`): Screen comparison with the original and decompilation lookups are written down as skills.
- L4 **Agent Guide Structure** (`agents_md_validation`): Sections let the agent jump to what it needs.

**Dev Environment** (3 of 6 assessed pass)

- L2 **Env Template** (`env_template`): A complete template makes every required variable visible at setup.
- L2 **Reproducible Test Data** (`test_data_reproducible`): An agent can only verify what it can run against known data.
- L3 **Agent App Verification** (`agent_app_verification`): An agent that can start the application and use it like a user verifies behaviour, not just tests.

**Debugging & Observability** (1 of 10 assessed pass)

- L2 **Structured Logging** (`structured_logging`): Parseable logs are evidence the agent can debug from.

**Security** (5 of 15 assessed pass)

- L1 **Comprehensive .gitignore** (`gitignore_comprehensive`): Secrets and build noise stay out of what the agent commits.
- L2 **CODEOWNERS** (`codeowners`): Routes each change to the people who must review it.
- L2 **Secrets Management** (`secrets_management`): Credentials stay out of the code the agent reads and writes.
- L2 **Agent Setup Owner** (`agent_setup_owner`): Without an owner, the agent setup drifts into what each developer happens to know, and nobody removes what no longer helps.
- L3 **Dependency Update Automation** (`dependency_update_automation`): Keeps the dependencies the agent reasons about current.

## Blocking level 4

10 more passing criteria clear level 4.

- Secret Scanning (`secret_scanning`) (trivial effort)
- Automated Security Review (`automated_security_review`) (trivial effort)
- Tech Debt Tracking (`tech_debt_tracking`) (trivial effort)
- Unused Dependencies Detection (`unused_dependencies_detection`) (trivial effort)
- Cyclomatic Complexity (`cyclomatic_complexity`) (small effort)
- Test Coverage Thresholds (`test_coverage_thresholds`) (small effort)
- Dead Code Detection (`dead_code_detection`) (small effort)
- Duplicate Code Detection (`duplicate_code_detection`) (small effort)
- Agent Outcome Tracking (`agent_outcome_tracking`) (small effort)
- Deployment Frequency (`deployment_frequency`) (small effort)
- Test Performance Tracking (`test_performance_tracking`) (small effort)
- Agent Rules Enforced (`agent_rules_enforced`) (small effort)
- Parallel Workspaces (`parallel_workspaces`) (small effort)
- Code Quality Metrics (`code_quality_metrics`) (small effort)
- Runbooks Documented (`runbooks_documented`) (small effort)
- License Compliance (`license_compliance`) (small effort)
- Log Scrubbing (`log_scrubbing`) (small effort)
- Gate Ratchet (`gate_ratchet`) (medium effort)
- Post-Deploy Verification (`post_deploy_verification`) (medium effort)
- Flaky Test Detection (`flaky_test_detection`) (medium effort)
- Mutation Testing (`mutation_testing`) (medium effort)
- Devcontainer Runnable (`devcontainer_runnable`)

## Critical gaps

Failing critical criteria: a high level does not make up for them, so close them first.

- L2 **Branch Protection** (`branch_protection`) (trivial effort): Protect main with a ruleset: pull requests required, CI checks required (one approval for outside contributors; as sole maintainer you may allow yourself to bypass).
- L3 **Agent Secret Access** (`agent_secret_access`) (small effort): Commit .claude/settings.json denying Read of .env*, *.pem, *.key and similar, and enable the sandbox.
- L3 **Agent Action Guardrails** (`agent_action_guardrails`) (small effort): In .claude/settings.json, put `gh workflow run`, `mise run release`, `git push --force`, `docker push`/`podman push` under ask.

## Priority fixes

The fixes are suggestions: check each against your stack before adopting a tool or library, and prefer a deterministic check over an instruction.

1. **Branch Protection** (`branch_protection`) (trivial effort): main takes direct pushes, so nothing the agent guide or CI asks for stops a change from landing.
   Suggested fix: Protect main with a ruleset: pull requests required, CI checks required (one approval for outside contributors; as sole maintainer you may allow yourself to bypass).
2. **Agent Action Guardrails** (`agent_action_guardrails`) (small effort): The guide says to ask before `mise run release` and pushes, but no agent configuration makes the agent wait.
   Suggested fix: In .claude/settings.json, put `gh workflow run`, `mise run release`, `git push --force`, `docker push`/`podman push` under ask.
3. **Agent Usage Agreement** (`agent_usage_agreement`) (trivial effort): A public repo with an agent guide but no word on which agent use is welcome; one paragraph closes it.
   Suggested fix: Add a short note on agent use (allowed tools, whether agent-made contributions are welcome) to the README or a CONTRIBUTING.md and link it from AGENTS.md.
4. **Secret Scanning** (`secret_scanning`) (trivial effort): Free for public repos and one setting; the cheapest step towards level 4.
   Suggested fix: Enable secret scanning and push protection in the repository settings (free for public repos).
5. **Automated Security Review** (`automated_security_review`) (trivial effort): CodeQL default setup covers Go, JavaScript and the workflows with one setting.
   Suggested fix: Enable CodeQL default setup (Go, JavaScript, Actions) on the repository.

## Quick wins

Failing criteria with a trivial fix (under an hour), lowest level first.

- L2 **Branch Protection** (`branch_protection`): Protect main with a ruleset: pull requests required, CI checks required (one approval for outside contributors; as sole maintainer you may allow yourself to bypass).
- L2 **Agent Usage Agreement** (`agent_usage_agreement`): Add a short note on agent use (allowed tools, whether agent-made contributions are welcome) to the README or a CONTRIBUTING.md and link it from AGENTS.md.
- L2 **Naming Consistency** (`naming_consistency`): Rename the German identifiers in wasm/server.go (immerNachfragen, dateien) to English and enforce Go naming with revive in .golangci.yml.
- L2 **Issue Templates** (`issue_templates`): Add bug and feature templates under .github/ISSUE_TEMPLATE/.
- L2 **PR Templates** (`pr_templates`): Add .github/pull_request_template.md asking for summary, checks run, screenshots for screen changes.
- L3 **Agent Change Traceability** (`agent_change_traceability`): Pick a marking rule (Co-Authored-By trailer or an `agent` PR label), write it in the agent guide and follow it.
- L3 **Generated Code Marked** (`generated_code_marked`): Mark remake/internal/embedded/data as linguist-generated in .gitattributes and deny it in a shared .claude/settings.json; AGENTS.md already names remake:extract.
- L3 **Profiling Instrumentation** (`profiling_instrumentation`): Register net/http/pprof on a local-only port in wasm/server.go.
- L4 **Secret Scanning** (`secret_scanning`): Enable secret scanning and push protection in the repository settings (free for public repos).
- L4 **Automated Security Review** (`automated_security_review`): Enable CodeQL default setup (Go, JavaScript, Actions) on the repository.
- L4 **Tech Debt Tracking** (`tech_debt_tracking`): Require an issue reference on TODO/FIXME via the godox linter in golangci-lint (today the Go code has no TODOs, so it passes at once).
- L4 **Unused Dependencies Detection** (`unused_dependencies_detection`): Add a CI step `go mod tidy && git diff --exit-code go.mod go.sum` in remake/; it would already flag that every requirement in go.mod is marked // indirect.
- L5 **Version Drift Detection** (`version_drift_detection`): Check in CI that the golang image of wasm/Dockerfile matches go.mod (one line of shell).

## Build with harness-build

The harness-build skill plans these gaps with you, then builds what you chose:

### For level 4 – Optimized (22)

- L4 Secret Scanning (`secret_scanning`) (trivial effort)
- L4 Automated Security Review (`automated_security_review`) (trivial effort)
- L4 Tech Debt Tracking (`tech_debt_tracking`) (trivial effort)
- L4 Unused Dependencies Detection (`unused_dependencies_detection`) (trivial effort)
- L4 Cyclomatic Complexity (`cyclomatic_complexity`) (small effort)
- L4 Test Coverage Thresholds (`test_coverage_thresholds`) (small effort)
- L4 Dead Code Detection (`dead_code_detection`) (small effort)
- L4 Duplicate Code Detection (`duplicate_code_detection`) (small effort)
- L4 Agent Outcome Tracking (`agent_outcome_tracking`) (small effort)
- L4 Deployment Frequency (`deployment_frequency`) (small effort)
- L4 Test Performance Tracking (`test_performance_tracking`) (small effort)
- L4 Agent Rules Enforced (`agent_rules_enforced`) (small effort)
- L4 Parallel Workspaces (`parallel_workspaces`) (small effort)
- L4 Code Quality Metrics (`code_quality_metrics`) (small effort)
- L4 Runbooks Documented (`runbooks_documented`) (small effort)
- L4 License Compliance (`license_compliance`) (small effort)
- L4 Log Scrubbing (`log_scrubbing`) (small effort)
- L4 Gate Ratchet (`gate_ratchet`) (medium effort)
- L4 Post-Deploy Verification (`post_deploy_verification`) (medium effort)
- L4 Flaky Test Detection (`flaky_test_detection`) (medium effort)
- L4 Mutation Testing (`mutation_testing`) (medium effort)
- L4 Devcontainer Runnable (`devcontainer_runnable`)

### Completing the levels below (25)

- L2 Branch Protection (`branch_protection`) (trivial effort)
- L2 Agent Usage Agreement (`agent_usage_agreement`) (trivial effort)
- L2 Naming Consistency (`naming_consistency`) (trivial effort)
- L2 Issue Templates (`issue_templates`) (trivial effort)
- L2 PR Templates (`pr_templates`) (trivial effort)
- L2 Large File Detection (`large_file_detection`) (small effort)
- L2 Health Checks (`health_checks`) (small effort)
- L2 Backlog Health (`backlog_health`) (small effort)
- L2 Issue Labeling System (`issue_labeling_system`) (small effort)
- L2 Metrics Collection (`metrics_collection`) (medium effort)
- L3 Agent Change Traceability (`agent_change_traceability`) (trivial effort)
- L3 Generated Code Marked (`generated_code_marked`) (trivial effort)
- L3 Profiling Instrumentation (`profiling_instrumentation`) (trivial effort)
- L3 Agent Action Guardrails (`agent_action_guardrails`) (small effort)
- L3 Scoped Agent Guides (`scoped_agent_guides`) (small effort)
- L3 Agent Secret Access (`agent_secret_access`) (small effort)
- L3 Automated PR Review (`automated_pr_review`) (small effort)
- L3 Infrastructure as Code (`infrastructure_as_code`) (small effort)
- L3 API Schema Docs (`api_schema_docs`) (small effort)
- L3 Devcontainer (`devcontainer`) (small effort)
- L3 Distributed Tracing (`distributed_tracing`) (small effort)
- L3 Feature Flag Infrastructure (`feature_flag_infrastructure`) (medium effort)
- L3 Alerting Configured (`alerting_configured`) (medium effort)
- L3 Deployment Observability (`deployment_observability`) (medium effort)
- L3 Error Tracking (`error_tracking_contextualized`) (medium effort)

### Higher levels (8)

- L5 Version Drift Detection (`version_drift_detection`) (trivial effort)
- L5 Monorepo Tooling (`monorepo_tooling`) (small effort)
- L5 Rollback Automation (`rollback_automation`) (small effort)
- L5 DAST Scanning (`dast_scanning`) (small effort)
- L5 Progressive Rollout (`progressive_rollout`) (medium effort)
- L5 Error-to-Insight Pipeline (`error_to_insight_pipeline`) (medium effort)
- L5 Product Analytics Instrumentation (`product_analytics_instrumentation`) (medium effort)
- L5 Dead Feature Flag Detection (`dead_feature_flag_detection`)

Prompt for the next level and the open gaps below it:

```
/swa-harness:harness-build Close these gaps from the agent-readiness report 2026-10-23: branch_protection, agent_usage_agreement, naming_consistency, issue_templates, pr_templates, large_file_detection, health_checks, backlog_health, issue_labeling_system, metrics_collection, agent_change_traceability, generated_code_marked, profiling_instrumentation, agent_action_guardrails, scoped_agent_guides, agent_secret_access, automated_pr_review, infrastructure_as_code, api_schema_docs, devcontainer, distributed_tracing, feature_flag_infrastructure, alerting_configured, deployment_observability, error_tracking_contextualized, secret_scanning, automated_security_review, tech_debt_tracking, unused_dependencies_detection, cyclomatic_complexity, test_coverage_thresholds, dead_code_detection, duplicate_code_detection, agent_outcome_tracking, deployment_frequency, test_performance_tracking, agent_rules_enforced, parallel_workspaces, code_quality_metrics, runbooks_documented, license_compliance, log_scrubbing, gate_ratchet, post_deploy_verification, flaky_test_detection, mutation_testing, devcontainer_runnable
```

## Strengths

- **Check Command** (`check_command`): One mise task runs lint and every unit test, locally and in CI.
- **Integration Tests Exist** (`integration_tests_exist`): Both images are exercised end to end in CI and the remake's screens are driven headless.
- **Skills** (`skills`): Screen comparison with the original and decompilation lookups are written down as skills.
- **Domain Docs** (`domain_docs`): Glossary and ADRs give the game terms and the reasons behind embedding and the C random generator.
- **Release Automation** (`release_automation`): Releases tag and publish only once both images exist.

## Notes

- Assessed the branch `remake` at a2f21af with the uncommitted harness-build changes (--worktree).
- deployed_service is `operated` on the user's statement: the maintainer runs the images on a Synology NAS through Watchtower; others self-host the same images.
- Dependabot covers Actions, npm and Docker but not the Go module in remake/ (no `gomod` entry).
- Several operations criteria (flags, canary, metrics, analytics) fit a one-host hobby deployment poorly; they are candidates for waivers rather than work.
- remake/go.mod marks every requirement // indirect, a sign that `go mod tidy` has not run.

## Changes since 2026-10-23

Level 1 → 3

- **Newly passed:** Lint Config, Check Command, Unit Tests Exist, Test Naming Conventions, Unit Tests Runnable, Test Stage Rules, Test Isolation, README, Agent Guide, Agent Guide Substance, Agent Guide Completeness, Agent Guide Freshness, Agent Boundaries, Automated Doc Generation, Domain Docs, Skills, Agent Guide Structure, Structured Logging, Comprehensive .gitignore, CODEOWNERS, Agent Setup Owner

## Categories

| Category | Passed / evaluated / total |
| --- | --- |
| Style & Validation | 6 / 12 / 13 |
| Build System | 11 / 24 / 25 |
| Testing | 6 / 11 / 11 |
| Documentation | 12 / 15 / 16 |
| Dev Environment | 3 / 6 / 9 |
| Debugging & Observability | 1 / 10 / 11 |
| Security | 5 / 15 / 17 |
| Task Discovery | 0 / 4 / 4 |
| Product & Analytics | 0 / 2 / 2 |

## Perspectives

| Perspective | Passed / evaluated / total |
| --- | --- |
| Agent | 36 / 79 / 85 |
| Architecture | 3 / 4 / 5 |
| Governance | 5 / 16 / 18 |

## All criteria

### Style & Validation

| Criterion | Level | Status | Evidence |
| --- | --- | --- | --- |
| Formatter (`formatter`) | L1 | ✅ pass | file: `mise.toml:101-104` – gofmt -l in remake:test<br>file: `mise.toml:85-88` – gofmt -l in wasm:test<br>file: `.githooks/pre-commit:14-17` – gofmt on staged remake |
| Lint Config (`lint_config`) | L1 | ✅ pass | file: `.golangci.yml` – golangci-lint standard linters<br>file: `mise.toml:141-143` – lint task over remake/ and wasm/<br>file: `mise.toml:5-6` – golangci-lint 2.14.0 pinned |
| Strict Typing (`strict_typing`) | L1 | ✅ pass | profile: `languages` – Go is strict by default |
| Type Check (`type_check`) | L1 | ✅ pass | profile: `languages` – Go is statically typed and compiled |
| Large File Detection (`large_file_detection`) | L2 | ❌ fail | command: `git ls-files \| grep gitattributes` – no .gitattributes, no size check in .githooks/pre-commit or ci.yml<br>Suggested fix (small effort): Add a size check for new files (a few lines in .githooks/pre-commit and CI), with an allowlist for the original game files in bmp/ and remake/internal/embedded/data. |
| Naming Consistency (`naming_consistency`) | L2 | ❌ fail | file: `AGENTS.md:61-62` – English identifiers written down<br>command: `git grep -n immerNachfragen wasm/server.go` – the save server does not follow it yet<br>Suggested fix (trivial effort): Rename the German identifiers in wasm/server.go (immerNachfragen, dateien) to English and enforce Go naming with revive in .golangci.yml. |
| Pre-commit Hooks (`pre_commit_hooks`) | L3 | ✅ pass | file: `.githooks/pre-commit`<br>file: `mise.toml:149-151` – mise run hooks |
| Code Modularization (`code_modularization`) | L3 | ✅ pass | file: `remake/internal/` – Go internal/ packages: assets, game, ui, embedded |
| Cyclomatic Complexity (`cyclomatic_complexity`) | L4 | ❌ fail | command: `git ls-files \| grep golangci` – no complexity limit configured<br>Suggested fix (small effort): Enable gocyclo or gocognit in the golangci-lint config with a threshold at today's maximum and lower it over time. |
| Dead Code Detection (`dead_code_detection`) | L4 | ❌ fail | file: `.github/workflows/ci.yml:46-56` – CI runs vet, gofmt and tests only<br>Suggested fix (small effort): Run golang.org/x/tools/cmd/deadcode (or golangci-lint unused) in the CI remake job. |
| Duplicate Code Detection (`duplicate_code_detection`) | L4 | ❌ fail | file: `.github/workflows/ci.yml:46-56` – no duplication check<br>Suggested fix (small effort): Enable the dupl linter in golangci-lint with a sensible threshold. |
| Tech Debt Tracking (`tech_debt_tracking`) | L4 | ❌ fail | command: `git grep -cE 'TODO\|FIXME' -- '*.go'` – no TODOs, but no rule<br>Suggested fix (trivial effort): Require an issue reference on TODO/FIXME via the godox linter in golangci-lint (today the Go code has no TODOs, so it passes at once). |
| N+1 Detection (`n_plus_one_detection`) | L5 | ⏭️ not applicable | profile: `orm` – no ORM or database |

### Build System

| Criterion | Level | Status | Evidence |
| --- | --- | --- | --- |
| Dependencies Pinned (`deps_pinned`) | L1 | ✅ pass | file: `remake/go.sum`<br>file: `vnc/bench/package-lock.json` |
| Toolchain Pinned (`toolchain_pinned`) | L1 | ✅ pass | file: `remake/go.mod:3` – go 1.27.2<br>file: `wasm/go.mod:3` – go 1.27.2<br>file: `.github/workflows/ci.yml:31-33` – setup-go reads go.mod<br>file: `mise.toml:5-6` – golangci-lint pinned, installed by mise-action in CI |
| Documented Build Commands (`build_cmd_doc`) | L1 | ✅ pass | file: `AGENTS.md:7-8`<br>file: `README.md:136-142` – task table<br>file: `remake/README.md:35-45` |
| Git Hosting CLI (`vcs_cli_tools`) | L1 | ✅ pass | file: `AGENTS.md:35-38` – gh for pull requests and CI results<br>file: `mise.toml:158-161` – release task uses gh |
| Single Command Setup (`single_command_setup`) | L1 | ✅ pass | file: `remake/README.md:24-31` – prerequisites: Go, mise, Linux GL/X11 headers<br>file: `remake/README.md:119-123` – mise run remake:test runs from a fresh clone; go fetches modules, test data is committed |
| Check Command (`check_command`) | L2 | ✅ pass | file: `mise.toml:145-147` – check: lint, remake:test, wasm:test<br>file: `AGENTS.md:12-15` – named in the agent guide |
| Agentic Development (`agentic_development`) | L3 | ✅ pass | file: `AGENTS.md`<br>file: `.claude/` – skills |
| Automated PR Review (`automated_pr_review`) | L3 | ❌ fail | command: `ls .github/workflows` – ci.yml and release.yml only; no reviewer, no review rules<br>Suggested fix (small effort): Run an AI or rule-based reviewer on pull requests fed by a committed REVIEW.md (for example the claude-code-action with the project's review rules). |
| Fast CI Feedback (`fast_ci_feedback`) | L3 | ✅ pass | file: `.github/workflows/ci.yml:12` – runs on pull_request<br>file: `.github/workflows/ci.yml:53-54` – calls mise run check |
| Feature Flag Infrastructure (`feature_flag_infrastructure`) | L3 | ❌ fail | command: `git grep -niE 'feature.?flag\|openfeature'` – nothing found<br>Suggested fix (medium effort): Introduce a small flag mechanism (OpenFeature or a flag file read at start) for behaviour that should ship dark, or waive this for a single-user game. |
| Release Automation (`release_automation`) | L3 | ✅ pass | file: `.github/workflows/release.yml` – versioned tags, images and GitHub releases |
| Infrastructure as Code (`infrastructure_as_code`) | L3 | ❌ fail | file: `vnc/docker-compose.yml` – the operated setup is defined as compose<br>file: `.github/workflows/ci.yml` – no validation of the compose files<br>Suggested fix (small effort): Validate both compose files in CI (`docker compose -f vnc/docker-compose.yml config`, same for wasm) and show the image/compose diff on the pull request. |
| Generated Code Marked (`generated_code_marked`) | L3 | ❌ fail | file: `AGENTS.md:71-72` – generator named<br>command: `git ls-files \| grep gitattributes` – no .gitattributes, no deny rule<br>Suggested fix (trivial effort): Mark remake/internal/embedded/data as linguist-generated in .gitattributes and deny it in a shared .claude/settings.json; AGENTS.md already names remake:extract. |
| Build Performance Tracking (`build_performance_tracking`) | L4 | ✅ pass | file: `.github/workflows/ci.yml:96` – Docker layer cache in GHA<br>file: `.github/workflows/ci.yml:34` – Go module cache |
| Agent Outcome Tracking (`agent_outcome_tracking`) | L4 | ❌ fail | command: `git log -200 --format='%(trailers:key=Co-Authored-By)'` – no agent markers to measure<br>Suggested fix (small effort): Mark agent commits (see agent_change_traceability) and add a small script that reports agent share and reverts from git log. |
| Deployment Frequency (`deployment_frequency`) | L4 | ❌ fail | file: `.github/workflows/release.yml:9-15` – workflow_dispatch only<br>Suggested fix (small effort): Release automatically (for example on every push to main that passes CI, or on a schedule) instead of only by manual dispatch; Watchtower then deploys. |
| Post-Deploy Verification (`post_deploy_verification`) | L4 | ❌ fail | file: `README.md:68` – Watchtower pulls new images, nothing verifies them<br>Suggested fix (medium effort): After Watchtower updates the NAS container, run the smoke/bench check against it (a small scheduled job or a Watchtower post-update hook) and roll back on failure. |
| Release Notes Automation (`release_notes_automation`) | L4 | ✅ pass | file: `.github/workflows/release.yml:155-159` – generate_release_notes: true |
| Unused Dependencies Detection (`unused_dependencies_detection`) | L4 | ❌ fail | file: `remake/go.mod:5-12` – all requirements marked // indirect, so go.mod is not tidy<br>Suggested fix (trivial effort): Add a CI step `go mod tidy && git diff --exit-code go.mod go.sum` in remake/; it would already flag that every requirement in go.mod is marked // indirect. |
| Dead Feature Flag Detection (`dead_feature_flag_detection`) | L5 | ❌ fail | criterion: `feature_flag_infrastructure` – `feature_flag_infrastructure` fails |
| Heavy Dependency Detection (`heavy_dependency_detection`) | L5 | ⏭️ not applicable | profile: `frontend_bundle` – static pages; js-dos is loaded from a CDN (wasm/index.html:42), no bundler |
| Monorepo Tooling (`monorepo_tooling`) | L5 | ❌ fail | file: `mise.toml:145-147` – check spans remake and the save server, not the images<br>Suggested fix (small effort): Let `mise run check` (or a `check:all`) also depend on wasm:smoke and vnc:bench so one task graph spans all three products. |
| Progressive Rollout (`progressive_rollout`) | L5 | ❌ fail | file: `.github/workflows/release.yml:149-153` – latest moves for every host at once<br>Suggested fix (medium effort): Release to a canary tag first (for example `next`) that one instance follows before `latest` moves, or waive this for a single-host deployment. |
| Rollback Automation (`rollback_automation`) | L5 | ❌ fail | command: `git grep -n -i rollback` – no rollback command or workflow<br>Suggested fix (small effort): Add a workflow (or mise task) that moves `latest`/`wasm` back to a given earlier version with `docker buildx imagetools create`, and document it. |
| Version Drift Detection (`version_drift_detection`) | L5 | ❌ fail | file: `remake/go.mod:3` – go 1.27.2<br>file: `wasm/Dockerfile:5` – golang:1.27-alpine, not tied to go.mod<br>Suggested fix (trivial effort): Check in CI that the golang image of wasm/Dockerfile matches go.mod (one line of shell). |

### Testing

| Criterion | Level | Status | Evidence |
| --- | --- | --- | --- |
| Unit Tests Exist (`unit_tests_exist`) | L1 | ✅ pass | file: `remake/internal/game/sim_test.go` – remake: unit tests in assets, game, ui<br>file: `wasm/server_test.go` – save server: httptest unit tests<br>profile: `products` – the VNC image has no code of its own beyond the start script of a third-party emulator, covered by the bench |
| Test Naming Conventions (`test_naming_conventions`) | L2 | ✅ pass | file: `wasm/server_test.go` – *_test.go next to the code, Test<Behaviour> names<br>file: `remake/internal/ui/flow_test.go` – same layout in remake |
| Unit Tests Runnable (`unit_tests_runnable`) | L2 | ✅ pass | file: `AGENTS.md:12-15` – mise run check runs all unit tests<br>file: `README.md:140` – documented in the README |
| Test Stage Rules (`test_stage_rules`) | L2 | ✅ pass | file: `AGENTS.md:17-26` – vnc:bench, wasm:smoke and screen comparison with the change that requires them |
| Integration Tests Exist (`integration_tests_exist`) | L3 | ✅ pass | file: `vnc/bench/bench.mjs` – Playwright run against the VNC image with --check<br>file: `wasm/smoke.sh` – page, bundle and save API of the WASM image<br>file: `remake/internal/ui/flow_test.go` – screens driven headless |
| Test Isolation (`test_isolation`) | L3 | ✅ pass | profile: `languages` – go test runs packages in parallel by default<br>file: `wasm/server_test.go` – each test gets its own temporary data directory |
| Flaky Test Detection (`flaky_test_detection`) | L4 | ❌ fail | command: `git grep -n -i flaky` – nothing tracks flaky tests<br>Suggested fix (medium effort): Record failed-then-passed CI runs (for example a weekly job running `go test -count=3` that files an issue on a flaky test). |
| Test Coverage Thresholds (`test_coverage_thresholds`) | L4 | ❌ fail | file: `mise.toml:101-104` – tests run without coverage<br>Suggested fix (small effort): Run `go test -coverprofile` in the check, publish it in CI and fail below the measured value. |
| Mutation Testing (`mutation_testing`) | L4 | ❌ fail | command: `git grep -n -i mutat -- .github` – no mutation testing<br>Suggested fix (medium effort): Run a Go mutation tester (for example go-mutesting or gremlins) on changed packages in a CI job and publish the score. |
| Test Performance Tracking (`test_performance_tracking`) | L4 | ❌ fail | file: `.github/workflows/ci.yml:53-54` – no timing report<br>Suggested fix (small effort): Upload `go test -json` timings from CI (or `gotestsum --junitfile`) and watch the slowest tests. |
| Gate Ratchet (`gate_ratchet`) | L4 | ❌ fail | file: `.github/CODEOWNERS` – covers only the agent setup<br>command: `gh api repos/schowave/bmp/branches/main/protection` – 404 Branch not protected<br>Suggested fix (medium effort): Add CODEOWNERS entries for .golangci.yml, mise.toml, .github/ and *_test.go, require code-owner review on main, and pin a coverage floor. |

### Documentation

| Criterion | Level | Status | Evidence |
| --- | --- | --- | --- |
| README (`readme`) | L1 | ✅ pass | file: `README.md:9` – points to the remake<br>file: `README.md:122-133` – layout with remake/, docs and AGENTS.md<br>file: `README.md:136-142` – commands incl. mise run check |
| Agent Guide (`agents_md`) | L2 | ✅ pass | file: `AGENTS.md` – agent guide<br>file: `CLAUDE.md` – imports AGENTS.md |
| Agent Guide Substance (`agents_md_size`) | L2 | ✅ pass | command: `wc -l AGENTS.md` – 90 lines |
| Documentation Freshness (`documentation_freshness`) | L2 | ✅ pass | command: `git log -1 --format=%cs -- README* docs remake/CLAUDE.md` – 2026-10-23, same day as the latest commit |
| Agent Guide Completeness (`agents_md_content`) | L3 | ✅ pass | file: `AGENTS.md:7-8` – run commands<br>file: `AGENTS.md:12-26` – check command and slower stages<br>file: `AGENTS.md:51-65` – conventions incl. the game/ui layering<br>file: `AGENTS.md:40-49` – definition of done<br>file: `AGENTS.md:80-87` – layout and docs |
| Agent Guide Freshness (`agents_md_freshness`) | L3 | ✅ pass | command: `git log -1 --format=%cs -- AGENTS.md` – uncommitted, changed 2026-10-23 |
| Agent Boundaries (`agent_boundaries`) | L3 | ✅ pass | file: `AGENTS.md:67-79` – ask before releases, CI changes, dependencies, bmp/; stop when spec and decompilation disagree |
| Scoped Agent Guides (`scoped_agent_guides`) | L3 | ❌ fail | file: `remake/CLAUDE.md` – scoped guide for remake/<br>command: `git ls-files -co --exclude-standard \| grep -E '(AGENTS\|CLAUDE).md'` – none for wasm/ or vnc/<br>Suggested fix (small effort): Give wasm/ and vnc/ a short scoped guide each (build, smoke/bench command, ports) and link them from AGENTS.md; remake/CLAUDE.md is one already. |
| API Schema Docs (`api_schema_docs`) | L3 | ❌ fail | file: `wasm/server.go:121` – save API registered without a schema<br>Suggested fix (small effort): Commit a short OpenAPI file for /api/saves/{name} (GET/PUT, name pattern, 4 MB limit, status codes) next to wasm/server.go. |
| Automated Doc Generation (`automated_doc_generation`) | L3 | ✅ pass | file: `.github/workflows/ci.yml:57-66` – go doc -all of the remake packages uploaded as remake-godoc |
| Service Flow Documented (`service_flow_documented`) | L3 | ✅ pass | file: `README.md:42-52` – architecture diagram of the VNC image |
| Target Architecture (`target_architecture`) | L3 | ⏭️ not applicable | statement: `user, 2026-10-23` – The code follows one style throughout, with no legacy patterns to avoid. |
| Domain Docs (`domain_docs`) | L3 | ✅ pass | file: `docs/glossary.md` – glossary<br>file: `docs/adr/0001-record-architecture-decisions.md` – ADRs 0001-0004<br>file: `AGENTS.md:85-86` – linked from the guide |
| Skills (`skills`) | L3 | ✅ pass | file: `.claude/skills/compare-screen/SKILL.md`<br>file: `.claude/skills/query-original/SKILL.md` |
| Agent Rules Enforced (`agent_rules_enforced`) | L4 | ❌ fail | file: `AGENTS.md:67-79` – hard limits<br>command: `git ls-files .claude/settings.json` – no agent permissions or hooks<br>Suggested fix (small effort): Back the hard limits with .claude/settings.json: ask before `mise run release`, `gh workflow run`, `git push` to main and `podman push`; a Stop hook that runs `mise run check`. |
| Agent Guide Structure (`agents_md_validation`) | L4 | ✅ pass | file: `AGENTS.md:10` – sections: Verify your work, Working with the platform, Definition of done, Conventions, Hard limits, Where to find context |

### Dev Environment

| Criterion | Level | Status | Evidence |
| --- | --- | --- | --- |
| Env Template (`env_template`) | L2 | ✅ pass | file: `remake/README.md:58-62` – BMP_DATA documented; the only variable the application code reads (remake/cmd/bmp/main.go:92) |
| Database Schema (`database_schema`) | L2 | ⏭️ not applicable | profile: `database` – no database |
| Local Services Setup (`local_services_setup`) | L2 | ⏭️ not applicable | profile: `database` – no external services; remake runs standalone, the images are self-contained |
| Reproducible Test Data (`test_data_reproducible`) | L2 | ✅ pass | file: `remake/internal/embedded/data/` – test data versioned<br>file: `remake/README.md:125-128` – tests run against the committed original files |
| Devcontainer (`devcontainer`) | L3 | ❌ fail | command: `git ls-files .devcontainer` – none<br>Suggested fix (small effort): Add .devcontainer/devcontainer.json with Go, mise and the GL/X11 headers plus xvfb. |
| Agent App Verification (`agent_app_verification`) | L3 | ✅ pass | file: `remake/CLAUDE.md:6` – remake:shot with input scripts to drive and screenshot screens<br>file: `remake/README.md:58-68` – -demo, -shot, -script flags<br>file: `vnc/bench/README.md` – Playwright drives the VNC image |
| Symbol Navigation (`symbol_navigation`) | L3 | ⏭️ not applicable | command: `report.py --scan` – Go is the one main language, 121 .go files (< 500) |
| Devcontainer Runnable (`devcontainer_runnable`) | L4 | ❌ fail | criterion: `devcontainer` – `devcontainer` fails |
| Parallel Workspaces (`parallel_workspaces`) | L4 | ❌ fail | file: `remake/CLAUDE.md` – worktrees for remake screens only<br>file: `mise.toml:9-12` – fixed CONTAINER_NAME and PORT<br>Suggested fix (small effort): Say in AGENTS.md that agents work in their own worktree (remake/CLAUDE.md does for screens), and derive the container name and ports of vnc:run/wasm:run-docker from the worktree. |

### Debugging & Observability

| Criterion | Level | Status | Evidence |
| --- | --- | --- | --- |
| Health Checks (`health_checks`) | L2 | ❌ fail | file: `wasm/Dockerfile:29-30` – liveness only: GET version.txt<br>file: `vnc/Dockerfile:63-64` – liveness only<br>Suggested fix (small effort): Add a readiness check that also checks dependencies: for WASM a /healthz that writes and removes a probe in /data, for VNC one that checks Xvnc and DOSBox run; point HEALTHCHECK at it. |
| Metrics Collection (`metrics_collection`) | L2 | ❌ fail | command: `git grep -n prometheus` – none<br>Suggested fix (medium effort): Expose a few Prometheus metrics from wasm/server.go (saves written, errors, request durations), or waive for a home deployment. |
| Structured Logging (`structured_logging`) | L2 | ✅ pass | file: `wasm/server.go:123` – slog JSON handler<br>file: `docs/adr/0004-structured-logs-with-slog.md` |
| Alerting Configured (`alerting_configured`) | L3 | ❌ fail | command: `git ls-files \| grep -iE 'alert\|monitor'` – none<br>Suggested fix (medium effort): Define an alert on the container health (Synology notifications or an Uptime Kuma monitor) as code in the repo, or waive. |
| Deployment Observability (`deployment_observability`) | L3 | ❌ fail | file: `.github/workflows/release.yml:149-153` – no deploy marker<br>Suggested fix (medium effort): Have the release workflow (or Watchtower notifications) record each deploy in whatever monitoring is used. |
| Distributed Tracing (`distributed_tracing`) | L3 | ❌ fail | command: `git grep -n -i otel` – none<br>Suggested fix (small effort): Add a request ID to the logs of wasm/server.go (full OpenTelemetry is overkill here), or waive. |
| Error Tracking (`error_tracking_contextualized`) | L3 | ❌ fail | file: `wasm/server.go:51` – errors only logged<br>Suggested fix (medium effort): Report server errors with the release version (Sentry or similar), or waive. |
| Profiling Instrumentation (`profiling_instrumentation`) | L3 | ❌ fail | command: `git grep -n pprof` – none<br>Suggested fix (trivial effort): Register net/http/pprof on a local-only port in wasm/server.go. |
| Circuit Breakers (`circuit_breakers`) | L4 | ⏭️ not applicable | command: `git grep -nE 'http\.(Get\|Post)\|http\.Client\|net\.Dial' -- '*.go'` – no outbound calls in the Go code |
| Code Quality Metrics (`code_quality_metrics`) | L4 | ❌ fail | file: `.github/workflows/ci.yml:55-56` – tests without coverage<br>Suggested fix (small effort): Upload `go test -coverprofile` results per pull request (Codecov or a job summary). |
| Runbooks Documented (`runbooks_documented`) | L4 | ❌ fail | file: `README.md:168-183` – troubleshooting for one case, no restart or rollback steps<br>Suggested fix (small effort): Write a short RUNBOOK.md: container unhealthy, saves fail, roll back to the previous image; the save-permission section of the README is a start. |

### Security

| Criterion | Level | Status | Evidence |
| --- | --- | --- | --- |
| Comprehensive .gitignore (`gitignore_comprehensive`) | L1 | ✅ pass | file: `.gitignore` – build output, .idea/.vscode, .env*, *.pem, *.key |
| Branch Protection (`branch_protection`) | L2 | ❌ fail | command: `gh api repos/schowave/bmp/branches/main/protection` – 404 Branch not protected; rulesets: []<br>command: `git log main --first-parent -40` – 1 merge, 0 pull-request commits in the last 40<br>Suggested fix (trivial effort): Protect main with a ruleset: pull requests required, CI checks required (one approval for outside contributors; as sole maintainer you may allow yourself to bypass). |
| CODEOWNERS (`codeowners`) | L2 | ✅ pass | file: `.github/CODEOWNERS` |
| Secrets Management (`secrets_management`) | L2 | ✅ pass | file: `.github/workflows/release.yml:75-78` – Docker Hub credentials from GitHub secrets<br>command: `git grep -nIE 'PRIVATE KEY\|AKIA\|ghp_\|password='` – no hits |
| Agent Usage Agreement (`agent_usage_agreement`) | L2 | ❌ fail | statement: `user, 2026-10-23` – Nothing is written down about which agent tools may be used.<br>Suggested fix (trivial effort): Add a short note on agent use (allowed tools, whether agent-made contributions are welcome) to the README or a CONTRIBUTING.md and link it from AGENTS.md. |
| Agent Setup Owner (`agent_setup_owner`) | L2 | ✅ pass | file: `AGENTS.md:89-90` – owner named<br>file: `.github/CODEOWNERS` – agent setup paths owned by @schowave |
| Agent Secret Access (`agent_secret_access`) | L3 | ❌ fail | command: `git ls-files .claude` – no shared agent configuration<br>Suggested fix (small effort): Commit .claude/settings.json denying Read of .env*, *.pem, *.key and similar, and enable the sandbox. |
| Agent Action Guardrails (`agent_action_guardrails`) | L3 | ❌ fail | command: `git ls-files .claude` – no shared agent configuration<br>file: `mise.toml:140-143` – mise run release starts a production release<br>Suggested fix (small effort): In .claude/settings.json, put `gh workflow run`, `mise run release`, `git push --force`, `docker push`/`podman push` under ask. |
| Agent Change Traceability (`agent_change_traceability`) | L3 | ❌ fail | command: `git log -200 --format='%(trailers:key=Co-Authored-By)'` – no trailers in the last 200 commits<br>Suggested fix (trivial effort): Pick a marking rule (Co-Authored-By trailer or an `agent` PR label), write it in the agent guide and follow it. |
| Dependency Update Automation (`dependency_update_automation`) | L3 | ✅ pass | file: `.github/dependabot.yml` – actions, npm and docker; gomod for remake/ missing |
| Automated Security Review (`automated_security_review`) | L4 | ❌ fail | command: `gh api repos/schowave/bmp/code-scanning/default-setup` – state: not-configured<br>Suggested fix (trivial effort): Enable CodeQL default setup (Go, JavaScript, Actions) on the repository. |
| License Compliance (`license_compliance`) | L4 | ❌ fail | command: `git ls-files \| grep -i licen` – no policy, no check<br>Suggested fix (small effort): Write the allowed licenses down and add go-licenses (or dependency-review-action with allow-licenses) to CI. |
| Log Scrubbing (`log_scrubbing`) | L4 | ❌ fail | file: `wasm/server.go:53` – no redaction<br>Suggested fix (small effort): Add a ReplaceAttr to the slog handler that redacts tokens and request bodies; today the server logs save names and errors only. |
| PII Handling (`pii_handling`) | L4 | ⏭️ not applicable | profile: `personal_data` – no personal data processed |
| Secret Scanning (`secret_scanning`) | L4 | ❌ fail | command: `gh api repos/schowave/bmp --jq .security_and_analysis` – secret_scanning and push_protection disabled<br>Suggested fix (trivial effort): Enable secret scanning and push protection in the repository settings (free for public repos). |
| DAST Scanning (`dast_scanning`) | L5 | ❌ fail | file: `.github/workflows/ci.yml:119-142` – image started, not scanned<br>Suggested fix (small effort): Run a ZAP baseline scan against the WASM image in CI (it is already started for the smoke test). |
| Privacy Compliance (`privacy_compliance`) | L5 | ⏭️ not applicable | profile: `personal_data` – no personal data processed |

### Task Discovery

| Criterion | Level | Status | Evidence |
| --- | --- | --- | --- |
| Backlog Health (`backlog_health`) | L2 | ❌ fail | command: `gh issue list -R schowave/bmp --state open` – [] no open issues<br>Suggested fix (small effort): Track planned work (remake screens still to port, known deviations) as labelled GitHub issues with expected outcomes. |
| Issue Labeling System (`issue_labeling_system`) | L2 | ❌ fail | command: `gh label list -R schowave/bmp` – GitHub default labels plus Dependabot labels; no definition of ready<br>Suggested fix (small effort): Define labels as code (labels.yml with a sync action) including one for agent-sized issues, e.g. `agent-ready`. |
| Issue Templates (`issue_templates`) | L2 | ❌ fail | command: `git ls-files .github/ISSUE_TEMPLATE` – none<br>Suggested fix (trivial effort): Add bug and feature templates under .github/ISSUE_TEMPLATE/. |
| PR Templates (`pr_templates`) | L2 | ❌ fail | command: `git ls-files .github` – no PR template<br>Suggested fix (trivial effort): Add .github/pull_request_template.md asking for summary, checks run, screenshots for screen changes. |

### Product & Analytics

| Criterion | Level | Status | Evidence |
| --- | --- | --- | --- |
| Error-to-Insight Pipeline (`error_to_insight_pipeline`) | L5 | ❌ fail | command: `git grep -n -i sentry` – none<br>Suggested fix (medium effort): Have the health check or an error tracker open a GitHub issue on repeated failures, or waive. |
| Product Analytics Instrumentation (`product_analytics_instrumentation`) | L5 | ❌ fail | command: `git grep -niE 'posthog\|matomo\|analytics'` – none<br>Suggested fix (medium effort): Count game starts and saves in the WASM server (privacy-friendly, no personal data), or waive for a hobby project. |


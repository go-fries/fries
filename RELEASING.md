# Releasing Fries

This guide describes how maintainers publish a stable Fries v4 release, from version selection to the GitHub release page. Run commands from the repository root. Major-version module-path migrations are outside its scope.

## Release conventions

The [Makefile](Makefile) provides the release commands. [versions.yaml](versions.yaml) defines the module sets, their versions, and the modules excluded from tagging. Release tools are pinned in [internal/tools/go.mod](internal/tools/go.mod).

| Setting | Convention |
| --- | --- |
| Repository | `go-fries/fries` |
| Main branch | `4.x` |
| Remote | `origin` |
| Module set | `stable` |
| Release branch | `release/<version>` |
| Generated preparation branch | `prerelease_stable_<version>` |
| Root module tag | `<version>` |
| Component module tag | `<module-directory>/<version>` |
| Release PR title | `Release <version>` |
| GitHub release title | `<version>` |

All modules in `module-sets.stable.modules` share one release version. The `internal/tools` module and the two standalone example modules are excluded from published tags. Preparation still updates references to stable modules in their `go.mod` files. Exclusion from tagging does not mean exclusion from every development command.

Use a Go toolchain supported by the repository, Git, Make, and an authenticated GitHub CLI (`gh`) with permission to push branches and tags and publish releases. The current minimum Go version is `1.26.0`; check [go.mod](go.mod) when preparing a release. Make builds the required tools in `.tools` as needed.

Configure a Git author identity and working tag signing before tagging. The pinned `multimod` invokes `git tag -a -s`, so its tags are signed annotated tags. The signing key and signing program must be accessible in the release shell.

The current workflows validate branches and PRs; they do not create release tags or publish GitHub releases. PR merging, tag publication, and GitHub release publication are separate steps.

Set these variables in your shell:

```sh
BASE_BRANCH=4.x
MODSET=stable
```

The version examples below use the completed `v4.2.0` release and its predecessor, `v4.1.0`. Replace them with the target version and its previous published version. Do not rerun preparation or tagging for an already published version. Restore the variables if you resume in a new shell.

## 1. Choose a version

Start with a clean working tree. Resolve tracked and untracked work before switching branches or running preparation:

```sh
git status --short --branch
git switch "$BASE_BRANCH"
git pull --ff-only --tags origin "$BASE_BRANCH"
make gorelease
```

Read each module's inferred base version, suggested version, and diagnostics. Review the changes since the previous release, including public APIs, behavior, dependencies, and Go requirements. Choose a shared version appropriate for the stable set, rather than using the root module's suggestion alone.

Important limitations:

- `make gorelease` checks every module except `internal/tools`, including the excluded example modules. Example-module suggestions do not determine the stable version.
- The Makefile catches individual gorelease failures. A zero exit code does not mean every module passed; investigate the output before proceeding.
- Newly added modules may have no published base version. Check their diagnostics separately and include them in the release inventory and expected tags.
- Backward compatibility is the default within v4. Review incompatible-change diagnostics against [the contribution guidelines](CONTRIBUTING.md#compatibility-and-review). Resolve release-blocking issues and document migration steps for any approved incompatibilities; do not silently discard diagnostics to obtain a preferred version.

After selecting the version, set:

```sh
VERSION=v4.2.0
PREVIOUS_VERSION=v4.1.0
```

Confirm that `$PREVIOUS_VERSION` is the intended published baseline, and that the target version is not already published.

## 2. Prepare the release

Create the release branch:

```sh
git switch -c "release/$VERSION"
```

Edit `module-sets.stable.version` in `versions.yaml` to match `$VERSION`, including its `v` prefix. Check that all public modules being released are listed in the stable set and that exclusions remain intentional. Commit the configuration before running preparation:

```sh
git add versions.yaml
git commit -m "chore(release): prepare $VERSION"
git status --short --branch
make prerelease MODSET="$MODSET"
```

Preparation requires a clean working tree. The command:

1. Verifies the module-set configuration and checks for existing version tags.
2. Updates existing module-local `version.go` files for the stable set.
3. Updates stable-module requirements across the repository's `go.mod` files, including excluded modules.
4. Runs module tidying for the configured release modules.
5. Creates `prerelease_stable_<version>` and commits the generated changes there.
6. Returns to the original `release/<version>` branch after a successful run.

Here, `prerelease` means release preparation. It does not publish a GitHub prerelease or imply an alpha, beta, or release-candidate version.

Merge the generated branch into the release branch and review the complete preparation diff:

```sh
git merge --ff-only "prerelease_${MODSET}_${VERSION}"
git diff --check "origin/$BASE_BRANCH...HEAD"
git diff "origin/$BASE_BRANCH...HEAD"
git status --short --branch
```

Confirm that `versions.yaml`, existing `Version()` helpers, and intra-repository module requirements agree. `Version()` returns the version without the `v` prefix. Modules without a `version.go` file are still released and tagged when they belong to the stable set. Review any `go.sum` changes as well.

Validate the prepared branch using [the repository validation commands](CONTRIBUTING.md#validation). A local baseline is:

```sh
make lint
make build
make test-unit
git status --short --branch
```

`make lint` runs module tidying and may modify dependency files. Review and commit any required changes before pushing. Run `make test-integration` with Redis and MySQL provisioned as described in [the integration test instructions](CONTRIBUTING.md#service-independent-and-integration-suites). Short-mode tests alone do not validate service-backed behavior.

## 3. Open and merge the release PR

Push the prepared branch:

```sh
git push -u origin "release/$VERSION"
```

Write a PR description that separates:

- **Release preparation:** version alignment and generated dependency changes, using the PR base-to-head diff.
- **Changes being released:** the previous release tag to the PR base commit, including features, fixes, compatibility changes, and dependency updates.
- **Validation:** commands actually run and the state of the PR checks.

Record commit SHAs for the range endpoints when recording the release inventory. Verify PR membership against the commit history, including direct commits. The release-preparation PR is not part of its own base history; it will be included in the final tag-to-tag changelog.

Create the ignored draft directory, write and review `.docs/releases/release-pr-$VERSION.md`, then open the PR:

```sh
mkdir -p .docs/releases
# Write and review the PR body file before running gh pr create.
gh pr create \
  --repo go-fries/fries \
  --base "$BASE_BRANCH" \
  --head "release/$VERSION" \
  --title "Release $VERSION" \
  --body-file ".docs/releases/release-pr-$VERSION.md"

PR_NUMBER=$(gh pr view "release/$VERSION" --repo go-fries/fries --json number --jq '.number')
gh pr checks "$PR_NUMBER" --repo go-fries/fries --watch
```

Check the current workflows rather than assuming the local baseline covers the full matrix:

- [Go Lint](.github/workflows/lint.yml) runs lint, builds, and the tracked-file cleanliness check.
- [Go Test](.github/workflows/test.yml) runs unit, Redis, MySQL latest, and MySQL 5.7 suites on Go `1.26.x` and `1.27.x`, with race detection, uncached execution, and coverage collection. It then merges coverage separately for each Go version and uploads it to Codecov.

All expected checks should complete successfully before merging. Pending, skipped, or cancelled jobs are not evidence that a suite passed. Record failures accurately and resolve them rather than describing incomplete validation as successful.

Merge the reviewed PR into `4.x` after the checks pass.

## 4. Update the main branch and create tags

After the PR is merged, update the main branch again immediately before tagging:

```sh
git switch "$BASE_BRANCH"
git pull --ff-only --tags origin "$BASE_BRANCH"
git status --short --branch
git log -1 --oneline
```

Confirm that the worktree is clean, the branch is synchronized with `origin/4.x`, and `HEAD` is the intended release commit. If more commits have landed since the release PR, review whether they belong in the release and have completed validation. Check that `versions.yaml` still specifies `$VERSION` and the intended module membership. Stop if these conditions are not met.

Pin the checked commit explicitly, then create local tags:

```sh
RELEASE_COMMIT=$(git rev-parse HEAD)
make add-tags MODSET="$MODSET" COMMIT="$RELEASE_COMMIT"
git tag --points-at "$RELEASE_COMMIT" --list "$VERSION" "*/$VERSION"
```

`add-tags` verifies the module set and creates a signed annotated tag for every stable module at the specified commit. `COMMIT` defaults to `HEAD` when omitted. It does not push tags.

For example, the root, Cache, Redis Cache, and Kratos OpenTelemetry modules use these tag names:

```text
v4.2.0
cache/v4.2.0
cache/redis/v4.2.0
kratos/middleware/otel/v4.2.0
```

This is only an illustration, not the full tag list. Generate the expected list from the stable set in `versions.yaml` and compare it with the actual local tags:

```sh
mkdir -p .docs/releases
awk -v version="$VERSION" '
  /^excluded-modules:/ { exit }
  /^  stable:/ { selected = 1; next }
  selected && /^  [^ ]/ { selected = 0 }
  selected && $1 == "-" {
    module = $2
    sub(/^github[.]com\/go-fries\/fries/, "", module)
    sub(/\/v4$/, "", module)
    print (module == "" ? "" : substr(module, 2) "/") version
  }
' versions.yaml | LC_ALL=C sort > ".docs/releases/$VERSION.expected-tags"

git tag --list "$VERSION" "*/$VERSION" | LC_ALL=C sort > ".docs/releases/$VERSION.local-tags"
diff -u ".docs/releases/$VERSION.expected-tags" ".docs/releases/$VERSION.local-tags"

while IFS= read -r tag; do
  git for-each-ref --format='%(refname:strip=2) %(objecttype) %(*objectname)' "refs/tags/$tag"
done < ".docs/releases/$VERSION.expected-tags"
```

The diff should be empty. Each row in the final listing should have object type `tag` and the same peeled commit, `$RELEASE_COMMIT`. Do not rely on a fixed tag count: module membership changes over time. To audit an older release, use `versions.yaml` from that release's commit rather than today's module list.

## 5. Push tags

The Makefile selects tags using a version-suffix regular expression over all local tag names, not `MODSET`. Compare that selection with the expected list before pushing:

```sh
git tag --list | grep "${VERSION}$" | LC_ALL=C sort > ".docs/releases/$VERSION.push-tags"
diff -u ".docs/releases/$VERSION.expected-tags" ".docs/releases/$VERSION.push-tags"
```

The diff must be empty. Only push after the complete local tag set and target commit have been verified:

```sh
make push-tags TAG="$VERSION"
git ls-remote --tags origin \
  "refs/tags/$VERSION" "refs/tags/*/$VERSION" \
  "refs/tags/$VERSION^{}" "refs/tags/*/$VERSION^{}"
```

`push-tags` pushes the selected local tags in parallel; it does not provide an atomic all-or-nothing push.

Compare the remote refs with the expected tag list from step 4. Every expected tag must have reached `origin`; entries ending in `^{}` must all resolve to `$RELEASE_COMMIT`. Do not continue to the GitHub release while tags are missing or point to different commits.

Pushing module tags makes the Go module versions available independently of the GitHub release page. Treat published version tags as immutable.

## 6. Publish the GitHub release

Generate a starting changelog for the two published tags:

```sh
mkdir -p .docs/releases
gh api repos/go-fries/fries/releases/generate-notes \
  -f tag_name="$VERSION" \
  -f previous_tag_name="$PREVIOUS_VERSION" \
  --jq .body > ".docs/releases/release-notes-$VERSION.md"

git log --oneline "$PREVIOUS_VERSION..$VERSION"
```

Edit the draft using the [release notes conventions](#release-notes-conventions). Verify the generated inventory against the tag-to-tag history, including the merged release-preparation PR. Preserve included PRs and author attribution, and account for direct commits that generated notes may omit.

Publish the reviewed notes against the existing root tag as a stable release and mark it Latest:

```sh
gh release create "$VERSION" \
  --repo go-fries/fries \
  --verify-tag \
  --title "$VERSION" \
  --latest \
  --notes-file ".docs/releases/release-notes-$VERSION.md"
```

`--verify-tag` prevents the CLI from creating a missing root tag automatically. It does not verify the component tags; that verification belongs to step 5. This stable-release command is not intended for alpha, beta, or release-candidate publication.

Verify the published body, tag, and release status:

```sh
gh release view "$VERSION" --repo go-fries/fries \
  --json url,tagName,name,isDraft,isPrerelease,body
gh api repos/go-fries/fries/releases/latest --jq '.tag_name'
```

The release should use `$VERSION` for its tag and title, have both `isDraft` and `isPrerelease` set to `false`, and appear as Latest. Open the returned URL to inspect the rendered headings, code blocks, and links.

To correct published wording, edit the same notes file and update the release:

```sh
gh release edit "$VERSION" --repo go-fries/fries \
  --notes-file ".docs/releases/release-notes-$VERSION.md"
```

Read the published body again after an update. Wording corrections do not require moving or recreating tags.

## Release notes conventions

Use concise English prose and organize the release page around user impact:

1. A one-sentence summary of the actual changes.
2. **Highlights:** notable additions and fixes, with component names and PR links.
3. **Upgrade:** the required Go version and commands for the modules used by the application.
4. **Breaking changes and migration:** only when needed; identify affected modules and concrete migration steps.
5. **What's Changed:** the complete PR inventory, grouped by the changes present, preserving author attribution. Include direct changes without PRs where applicable.
6. **Full Changelog:** a link comparing the previous root tag with the new root tag.

Fries is a multi-module repository. Upgrading `github.com/go-fries/fries/v4` alone does not upgrade every component module used by an application. Show representative component commands and tell readers to replace them with their directly used modules:

```sh
go get github.com/go-fries/fries/v4@v4.2.0
go get github.com/go-fries/fries/cache/v4@v4.2.0
go get github.com/go-fries/fries/cache/redis/v4@v4.2.0
go mod tidy
go test ./...
```

Keep related Fries modules aligned to the release being adopted. Explain new module paths and compatibility changes when they require action. State the Go requirement once in Upgrade and use inline code for module names and versions.

Keep CI execution records and inventory counts in the release PR, not the release page. Avoid repeated compatibility claims and routine publication details. Use the comparison URL format `https://github.com/go-fries/fries/compare/<previous-version>...<version>`.

For a completed release in this repository, see [release preparation PR #2911](https://github.com/go-fries/fries/pull/2911) and the [v4.2.0 release](https://github.com/go-fries/fries/releases/tag/v4.2.0). Their historical module membership may differ from the current stable set.

## Troubleshooting

### gorelease reports missing checksums or cannot determine a base

For checksum errors, run the affected module's tidy target, such as `make go-mod-tidy/cache`, inspect its `go.mod` and `go.sum` changes, and rerun its check with `make gorelease/cache`. Commit reviewed changes before preparation, which requires a clean worktree.

For base-version errors, confirm that tags were fetched and check whether the component has been published before. A newly added component has no previous release tag. Investigate other download, proxy, or compatibility errors separately rather than treating the Make exit status as a passing result.

### prerelease reports a dirty worktree

Inspect both tracked and untracked files:

```sh
git status --short --branch --untracked-files=all
git diff
git diff --cached
git status --short --ignored
```

Commit the version configuration and any intended source changes before retrying. Keep drafts under the repository-ignored `.docs/` directory. If Git reports a clean worktree but the tool disagrees, inspect the relevant ignore rules with `git check-ignore -v <path>`; the tool uses a Git library whose ignore handling can differ from the Git CLI. Do not discard unrelated work to satisfy the check.

### The generated preparation branch already exists

A repeated preparation run can fail when `prerelease_stable_<version>` already exists. It may have modified version files before attempting to create the branch, and other failures can also leave preparation incomplete.

```sh
git status --short --branch
git diff
git diff --cached
git log --oneline "HEAD..prerelease_${MODSET}_${VERSION}"
git diff "HEAD...prerelease_${MODSET}_${VERSION}"
```

Inspect the current branch as well as the existing preparation branch. If its changes are correct, reuse it after resolving any failed-run changes in the worktree. If regeneration is necessary, preserve the existing branch under an unused backup name, undo only changes produced by the failed run, and retry from the clean release branch. Do not delete a branch or reset the worktree without first checking for unique work.

### Tag creation fails or version tags already exist

For signing errors, check the configured signing key, signing program, and agent access. The pinned tagger attempts to remove tags it created during a failed run, but inspect the actual local tag set before retrying.

Existing local tags can make preparation skip an already tagged set or reject a partially tagged set. Fetch and compare local and remote refs before deciding what happened. If the version is already published, do not reuse it. Fix unpublished local tagging mistakes only after confirming that the affected tags were not pushed; never force-update published module versions.

### Only some tags reached origin

Recheck the remote refs and their peeled commits against the expected list and `$RELEASE_COMMIT`. If existing remote tags are correct and the verified local set is unchanged, rerun `make push-tags TAG="$VERSION"` to push missing tags. This retry does not require regenerating local tags.

If any remote tag points to the wrong commit, stop and investigate. Do not repair a published release by moving tags; coordinate a new version instead.

## Completion checklist

- The release PR is merged and its validation evidence is recorded.
- Every module in the release commit's stable set has the expected remote tag at the intended commit.
- The GitHub release uses the root tag, is published as a stable release, and is marked Latest.
- The rendered release notes contain the intended upgrade and migration instructions and complete change inventory.
- Local release and generated preparation branches may be removed after verifying their changes are preserved. Squash merges can prevent Git's ancestry-based merged-branch check from recognizing them; inspect their changes before forcing local deletion.
